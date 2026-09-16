"""Bluetooth radio recovery when the adapter is powered off or stuck."""

from __future__ import annotations

import asyncio
import sys
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from pathlib import Path

from tp.debug_log import write as debug_write
from tp.debug_log import write_exception as debug_write_exception

BT_RADIO_OFF_SETTLE = 2.0
BT_RADIO_ON_SETTLE = 4.0
BT_RADIO_RESTART_COOLDOWN = 90.0
# WinRT set_state_async / get_radios_async can hang indefinitely; bound each op.
BT_RADIO_OP_TIMEOUT = 15.0
BT_PERMISSION_DENIED_COOLDOWN = 300.0
BT_STACK_RESET_COOLDOWN = 900.0
BT_STACK_RESET_SETTLE = 8.0

_radio_restart_lock: asyncio.Lock | None = None
_last_radio_restart_at: float | None = None
_last_permission_denied_at: float | None = None
_last_stack_reset_at: float | None = None
_permission_callback: BluetoothPermissionCallback | None = None


@dataclass(frozen=True)
class BluetoothPermissionRequest:
    title: str
    body: str
    action: str


BluetoothPermissionCallback = Callable[
    [BluetoothPermissionRequest], Awaitable[bool] | bool
]

BT_DISABLED_REQUEST = BluetoothPermissionRequest(
    title="Bluetooth is turned off",
    body="Enable Bluetooth so TemPy can discover and poll your sensors?",
    action="enable",
)

BT_STACK_RESET_REQUEST = BluetoothPermissionRequest(
    title="Bluetooth stack appears stuck",
    body=(
        "A sensor keeps failing after a Bluetooth radio restart. "
        "Reset Windows Bluetooth adapters now (same as reset_bluetooth.ps1)? "
        "This needs administrator approval (UAC)."
    ),
    action="stack_reset",
)


def windows_process_is_elevated() -> bool:
    """True when this process already has an elevated admin token."""
    if sys.platform != "win32":
        return False
    try:
        import ctypes

        return bool(ctypes.windll.shell32.IsUserAnAdmin())
    except Exception:  # noqa: BLE001
        return False


def windows_uac_elevation_prompts() -> bool:
    """True when elevating would show a UAC consent/credential dialog.

    Matches the common "Never notify" posture (ConsentPromptBehaviorAdmin=0)
    and fully disabled UAC (EnableLUA=0), where RunAs elevates silently.
    Always reads the native 64-bit policy view (KEY_WOW64_64KEY).
    """
    if sys.platform != "win32":
        return True
    try:
        import winreg

        access = winreg.KEY_READ
        if hasattr(winreg, "KEY_WOW64_64KEY"):
            access |= winreg.KEY_WOW64_64KEY
        with winreg.OpenKey(
            winreg.HKEY_LOCAL_MACHINE,
            r"SOFTWARE\Microsoft\Windows\CurrentVersion\Policies\System",
            0,
            access,
        ) as key:
            try:
                enable_lua, _ = winreg.QueryValueEx(key, "EnableLUA")
                if int(enable_lua) == 0:
                    return False
            except OSError:
                pass
            try:
                consent, _ = winreg.QueryValueEx(key, "ConsentPromptBehaviorAdmin")
                # 0 = Elevate without prompting (slider: Never notify).
                if int(consent) == 0:
                    return False
            except OSError:
                pass
    except OSError:
        return True
    return True


def windows_stack_reset_needs_user_consent() -> bool:
    """Whether TemPy should ask before launching the elevated Bluetooth reset.

    Skip the in-app prompt when elevation will not interrupt (already elevated,
    UAC off, or admin elevation without prompting).
    """
    if sys.platform != "win32":
        return True
    if windows_process_is_elevated():
        return False
    return windows_uac_elevation_prompts()


def set_bluetooth_permission_callback(
    callback: BluetoothPermissionCallback | None,
) -> None:
    """Register a UI handler that asks the user before changing Bluetooth state."""
    global _permission_callback
    _permission_callback = callback


def _get_radio_restart_lock() -> asyncio.Lock:
    global _radio_restart_lock
    if _radio_restart_lock is None:
        _radio_restart_lock = asyncio.Lock()
    return _radio_restart_lock


def is_bluetooth_powered_off_error(exc: Exception) -> bool:
    """True when bleak/OS reports the Bluetooth radio is off."""
    try:
        from bleak.exc import (
            BleakBluetoothNotAvailableError,
            BleakBluetoothNotAvailableReason,
        )

        if isinstance(exc, BleakBluetoothNotAvailableError):
            return exc.reason == BleakBluetoothNotAvailableReason.POWERED_OFF
    except ImportError:
        pass

    text = str(exc).lower()
    return any(
        phrase in text
        for phrase in (
            "radio is not powered on",
            "turn on bluetooth",
            "bluetooth radio is not powered",
            "bluetooth is turned off",
            "bluetooth powered off",
            "bluetooth off",
        )
    )


async def is_bluetooth_radio_disabled() -> bool:
    """True when the system Bluetooth radio is currently off."""
    if sys.platform == "win32":
        try:
            from winrt.windows.devices.radios import Radio, RadioKind, RadioState

            radios = await asyncio.wait_for(
                Radio.get_radios_async(),
                timeout=BT_RADIO_OP_TIMEOUT,
            )
            bt_radio = next(
                (radio for radio in radios if radio.kind == RadioKind.BLUETOOTH),
                None,
            )
            if bt_radio is None:
                return False
            return bt_radio.state != RadioState.ON
        except asyncio.TimeoutError:
            debug_write(
                f"ble: radio state check timed out after {BT_RADIO_OP_TIMEOUT:.0f}s"
            )
            return False
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: radio state check failed", exc)
            return False

    if sys.platform == "linux":
        try:
            proc = await asyncio.create_subprocess_exec(
                "bluetoothctl",
                "show",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.DEVNULL,
            )
            stdout, _stderr = await proc.communicate()
            if proc.returncode != 0:
                return False
            text = stdout.decode("utf-8", errors="ignore").lower()
            return "powered: no" in text
        except FileNotFoundError:
            return False
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: radio state check failed", exc)
            return False

    return False


async def _request_bluetooth_permission(
    request: BluetoothPermissionRequest,
) -> bool:
    """Ask the registered UI callback; returns False when declined or unavailable."""
    global _last_permission_denied_at

    now = time.monotonic()
    if (
        _last_permission_denied_at is not None
        and now - _last_permission_denied_at < BT_PERMISSION_DENIED_COOLDOWN
    ):
        debug_write(
            f"ble: Bluetooth {request.action} skipped (permission recently declined)"
        )
        return False

    if _permission_callback is None:
        debug_write(f"ble: Bluetooth {request.action} skipped (no permission handler)")
        return False

    maybe = _permission_callback(request)
    approved = await maybe if asyncio.iscoroutine(maybe) else bool(maybe)
    if approved:
        debug_write(f"ble: user approved Bluetooth {request.action}")
        return True

    _last_permission_denied_at = time.monotonic()
    debug_write(f"ble: user declined Bluetooth {request.action}")
    return False


async def _winrt_set_bluetooth_enabled(enabled: bool) -> bool:
    from winrt.windows.devices.radios import Radio, RadioAccessStatus, RadioKind, RadioState

    try:
        radios = await asyncio.wait_for(
            Radio.get_radios_async(),
            timeout=BT_RADIO_OP_TIMEOUT,
        )
        bt_radio = next(
            (radio for radio in radios if radio.kind == RadioKind.BLUETOOTH),
            None,
        )
        if bt_radio is None:
            debug_write("ble: Bluetooth radio not found")
            return False

        target = RadioState.ON if enabled else RadioState.OFF
        if bt_radio.state == target:
            return enabled

        result = await asyncio.wait_for(
            bt_radio.set_state_async(target),
            timeout=BT_RADIO_OP_TIMEOUT,
        )
        if result != RadioAccessStatus.ALLOWED:
            debug_write(f"ble: set Bluetooth {target.name} denied ({result.name})")
            return False

        if enabled:
            await asyncio.sleep(BT_RADIO_ON_SETTLE)
        else:
            await asyncio.sleep(BT_RADIO_OFF_SETTLE)

        radios = await asyncio.wait_for(
            Radio.get_radios_async(),
            timeout=BT_RADIO_OP_TIMEOUT,
        )
        bt_radio = next(
            (radio for radio in radios if radio.kind == RadioKind.BLUETOOTH),
            None,
        )
        ready = bt_radio is not None and bt_radio.state == target
        debug_write(
            f"ble: set Bluetooth {target.name} "
            f"{'ok' if ready else 'failed'} "
            f"(state={bt_radio.state.name if bt_radio else 'missing'})"
        )
        return ready
    except asyncio.TimeoutError:
        debug_write(
            f"ble: set Bluetooth {'ON' if enabled else 'OFF'} timed out "
            f"after {BT_RADIO_OP_TIMEOUT:.0f}s"
        )
        return False


async def _linux_set_bluetooth_enabled(enabled: bool) -> bool:
    command = "on" if enabled else "off"
    try:
        proc = await asyncio.create_subprocess_exec(
            "bluetoothctl",
            "power",
            command,
            stdout=asyncio.subprocess.DEVNULL,
            stderr=asyncio.subprocess.DEVNULL,
        )
        await proc.wait()
        if proc.returncode != 0:
            debug_write(f"ble: bluetoothctl power {command} failed")
            return False
        await asyncio.sleep(BT_RADIO_ON_SETTLE if enabled else BT_RADIO_OFF_SETTLE)
        return not await is_bluetooth_radio_disabled() if enabled else True
    except FileNotFoundError:
        if not enabled:
            return False
        try:
            block = await asyncio.create_subprocess_exec(
                "rfkill",
                "unblock",
                "bluetooth",
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL,
            )
            await block.wait()
            await asyncio.sleep(BT_RADIO_ON_SETTLE)
            return block.returncode == 0
        except FileNotFoundError:
            debug_write("ble: enable Bluetooth unsupported on this Linux system")
            return False


async def enable_bluetooth_radio() -> bool:
    """Turn the Bluetooth radio on."""
    if sys.platform == "win32":
        return await _winrt_set_bluetooth_enabled(True)
    if sys.platform == "linux":
        return await _linux_set_bluetooth_enabled(True)
    debug_write("ble: enable Bluetooth unsupported on this platform")
    return False


async def _winrt_restart_bluetooth_radio() -> bool:
    from winrt.windows.devices.radios import Radio, RadioKind, RadioState

    try:
        radios = await asyncio.wait_for(
            Radio.get_radios_async(),
            timeout=BT_RADIO_OP_TIMEOUT,
        )
    except asyncio.TimeoutError:
        debug_write(
            f"ble: radio restart skipped (get_radios timed out after "
            f"{BT_RADIO_OP_TIMEOUT:.0f}s)"
        )
        return False

    bt_radio = next((radio for radio in radios if radio.kind == RadioKind.BLUETOOTH), None)
    if bt_radio is None:
        debug_write("ble: radio restart skipped (no Bluetooth radio found)")
        return False

    debug_write(f"ble: radio restart start (state={bt_radio.state.name})")

    if bt_radio.state == RadioState.ON:
        if not await _winrt_set_bluetooth_enabled(False):
            return False
    return await _winrt_set_bluetooth_enabled(True)


async def _linux_restart_bluetooth_radio() -> bool:
    """Best-effort Bluetooth toggle via rfkill or bluetoothctl."""
    if await _linux_set_bluetooth_enabled(False):
        return await _linux_set_bluetooth_enabled(True)

    try:
        off = await asyncio.create_subprocess_exec(
            "rfkill",
            "block",
            "bluetooth",
            stdout=asyncio.subprocess.DEVNULL,
            stderr=asyncio.subprocess.DEVNULL,
        )
        await off.wait()
        await asyncio.sleep(BT_RADIO_OFF_SETTLE)
        on = await asyncio.create_subprocess_exec(
            "rfkill",
            "unblock",
            "bluetooth",
            stdout=asyncio.subprocess.DEVNULL,
            stderr=asyncio.subprocess.DEVNULL,
        )
        await on.wait()
        await asyncio.sleep(BT_RADIO_ON_SETTLE)
        if on.returncode == 0:
            debug_write("ble: radio restart ok (rfkill)")
            return True
    except FileNotFoundError:
        pass

    debug_write("ble: radio restart unsupported on this Linux system")
    return False


async def restart_bluetooth_radio() -> bool:
    """Power-cycle the Bluetooth radio (off, then on)."""
    if sys.platform == "win32":
        return await _winrt_restart_bluetooth_radio()
    if sys.platform == "linux":
        return await _linux_restart_bluetooth_radio()
    debug_write("ble: radio restart unsupported on this platform")
    return False


async def _enable_with_cooldown() -> bool:
    global _last_radio_restart_at

    now = time.monotonic()
    if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
        debug_write("ble: Bluetooth enable skipped (cooldown)")
        return False

    lock = _get_radio_restart_lock()
    async with lock:
        now = time.monotonic()
        if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
            debug_write("ble: Bluetooth enable skipped (cooldown)")
            return False
        try:
            enabled = await enable_bluetooth_radio()
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: Bluetooth enable failed", exc)
            return False
        if enabled:
            _last_radio_restart_at = time.monotonic()
            debug_write("ble: Bluetooth enabled")
        return enabled


async def _restart_bluetooth_with_cooldown(reason: str) -> bool:
    """Power-cycle Bluetooth if outside cooldown. Returns True when radio is back on."""
    global _last_radio_restart_at

    now = time.monotonic()
    if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
        debug_write(f"ble: radio restart skipped (cooldown, {reason})")
        return False

    lock = _get_radio_restart_lock()
    async with lock:
        now = time.monotonic()
        if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
            debug_write(f"ble: radio restart skipped (cooldown, {reason})")
            return False
        try:
            restarted = await restart_bluetooth_radio()
        except Exception as restart_exc:  # noqa: BLE001
            debug_write_exception(f"ble: radio restart failed ({reason})", restart_exc)
            return False
        if restarted:
            _last_radio_restart_at = time.monotonic()
            debug_write(f"ble: radio restart complete ({reason})")
        return restarted


async def ensure_bluetooth_enabled_for_polling() -> bool:
    """If Bluetooth is off, ask permission and enable it before BLE work."""
    if not await is_bluetooth_radio_disabled():
        return True
    if not await _request_bluetooth_permission(BT_DISABLED_REQUEST):
        return False
    return await _enable_with_cooldown()


async def maybe_restart_bluetooth_radio(exc: Exception) -> bool:
    """Enable Bluetooth when the OS radio is off, after user approval."""
    disabled = await is_bluetooth_radio_disabled()
    if not disabled:
        if is_bluetooth_powered_off_error(exc):
            debug_write(
                "ble: bleak reports powered off but radio is on; retrying without enable"
            )
        return False
    if not await _request_bluetooth_permission(BT_DISABLED_REQUEST):
        return False
    return await _enable_with_cooldown()


async def maybe_restart_bluetooth_radio_after_total_failure() -> bool:
    """Power-cycle Bluetooth after fleet-wide or stuck-device failure (no permission prompt)."""
    return await _restart_bluetooth_with_cooldown("total fetch failure")


def reset_bluetooth_script_path() -> Path:
    """Path to the elevated Windows Bluetooth PnP reset script."""
    return Path(__file__).resolve().parent.parent / "reset_bluetooth.ps1"


async def reset_windows_bluetooth_stack() -> bool:
    """Run reset_bluetooth.ps1 elevated. Returns True when the reset finished."""
    if sys.platform != "win32":
        debug_write("ble: stack reset unsupported on this platform")
        return False

    script = reset_bluetooth_script_path()
    if not script.is_file():
        debug_write(f"ble: stack reset script missing ({script})")
        return False

    script_text = str(script)
    if windows_process_is_elevated():
        # Already admin — run the script directly (no RunAs / UAC).
        debug_write(f"ble: launching stack reset in-process elevated ({script.name})")
        try:
            proc = await asyncio.create_subprocess_exec(
                "powershell.exe",
                "-NoProfile",
                "-ExecutionPolicy",
                "Bypass",
                "-File",
                script_text,
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL,
            )
            await proc.wait()
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: stack reset launch failed", exc)
            return False
    else:
        # Nested Start-Process -Verb RunAs elevates; -Wait blocks until finished.
        # With ConsentPromptBehaviorAdmin=0 this elevates silently (no UAC UI).
        argument_list = (
            f"-NoProfile -ExecutionPolicy Bypass -File \\\"{script_text}\\\""
        )
        command = (
            "Start-Process -FilePath powershell.exe -Verb RunAs -Wait "
            f"-ArgumentList '{argument_list}'"
        )
        debug_write(f"ble: launching elevated stack reset ({script.name})")
        try:
            proc = await asyncio.create_subprocess_exec(
                "powershell.exe",
                "-NoProfile",
                "-ExecutionPolicy",
                "Bypass",
                "-Command",
                command,
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL,
            )
            await proc.wait()
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: stack reset launch failed", exc)
            return False

    if proc.returncode not in (0, None):
        debug_write(f"ble: stack reset launcher exited {proc.returncode}")
        # UAC cancel often yields non-zero; treat as failure.
        return False

    await asyncio.sleep(BT_STACK_RESET_SETTLE)
    debug_write("ble: stack reset finished; settled")
    return True


async def maybe_reset_bluetooth_stack_after_radio_failure() -> bool:
    """Run the Windows PnP Bluetooth reset after stuck-device radio recovery fails.

    No in-app Y/N: Windows UAC is the consent UI when the OS is set to notify.
    With UAC Never notify (or an already-elevated process), elevation is silent
    and monitoring recovers without interruption.
    """
    global _last_stack_reset_at, _last_radio_restart_at

    if sys.platform != "win32":
        return False

    now = time.monotonic()
    if (
        _last_stack_reset_at is not None
        and now - _last_stack_reset_at < BT_STACK_RESET_COOLDOWN
    ):
        debug_write("ble: stack reset skipped (cooldown)")
        return False

    # Log posture for diagnostics; never block on an in-app prompt.
    if windows_process_is_elevated():
        debug_write("ble: stack reset starting (process already elevated)")
    elif windows_uac_elevation_prompts():
        debug_write(
            "ble: stack reset starting (Windows may show a UAC prompt)"
        )
    else:
        debug_write(
            "ble: stack reset starting (UAC will not prompt; auto-elevating)"
        )

    lock = _get_radio_restart_lock()
    async with lock:
        now = time.monotonic()
        if (
            _last_stack_reset_at is not None
            and now - _last_stack_reset_at < BT_STACK_RESET_COOLDOWN
        ):
            debug_write("ble: stack reset skipped (cooldown)")
            return False
        ok = await reset_windows_bluetooth_stack()
        if ok:
            settled = time.monotonic()
            _last_stack_reset_at = settled
            # Avoid an immediate WinRT power-cycle right after the heavy PnP reset.
            _last_radio_restart_at = settled
        return ok
