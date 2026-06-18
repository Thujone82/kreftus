"""Bluetooth radio recovery when the adapter is powered off or stuck."""

from __future__ import annotations

import asyncio
import sys
import time

from tp.debug_log import write as debug_write
from tp.debug_log import write_exception as debug_write_exception

BT_RADIO_OFF_SETTLE = 2.0
BT_RADIO_ON_SETTLE = 4.0
BT_RADIO_RESTART_COOLDOWN = 90.0

_radio_restart_lock: asyncio.Lock | None = None
_last_radio_restart_at: float | None = None


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


async def _winrt_restart_bluetooth_radio() -> bool:
    from winrt.windows.devices.radios import Radio, RadioAccessStatus, RadioKind, RadioState

    radios = await Radio.get_radios_async()
    bt_radio = next((radio for radio in radios if radio.kind == RadioKind.BLUETOOTH), None)
    if bt_radio is None:
        debug_write("ble: radio restart skipped (no Bluetooth radio found)")
        return False

    debug_write(f"ble: radio restart start (state={bt_radio.state.name})")

    if bt_radio.state == RadioState.ON:
        result = await bt_radio.set_state_async(RadioState.OFF)
        if result != RadioAccessStatus.ALLOWED:
            debug_write(f"ble: radio power off denied ({result.name})")
            return False
        await asyncio.sleep(BT_RADIO_OFF_SETTLE)

    result = await bt_radio.set_state_async(RadioState.ON)
    if result != RadioAccessStatus.ALLOWED:
        debug_write(f"ble: radio power on denied ({result.name})")
        return False
    await asyncio.sleep(BT_RADIO_ON_SETTLE)

    radios = await Radio.get_radios_async()
    bt_radio = next((radio for radio in radios if radio.kind == RadioKind.BLUETOOTH), None)
    ready = bt_radio is not None and bt_radio.state == RadioState.ON
    debug_write(f"ble: radio restart {'ok' if ready else 'failed'} (state={bt_radio.state.name if bt_radio else 'missing'})")
    return ready


async def _linux_restart_bluetooth_radio() -> bool:
    """Best-effort Bluetooth toggle via rfkill or bluetoothctl."""
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

    try:
        for args in (
            ("bluetoothctl", "power", "off"),
            ("bluetoothctl", "power", "on"),
        ):
            proc = await asyncio.create_subprocess_exec(
                *args,
                stdout=asyncio.subprocess.DEVNULL,
                stderr=asyncio.subprocess.DEVNULL,
            )
            await proc.wait()
            if proc.returncode != 0:
                debug_write(f"ble: radio restart failed ({' '.join(args)})")
                return False
            if args[-1] == "off":
                await asyncio.sleep(BT_RADIO_OFF_SETTLE)
            else:
                await asyncio.sleep(BT_RADIO_ON_SETTLE)
        debug_write("ble: radio restart ok (bluetoothctl)")
        return True
    except FileNotFoundError:
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


async def maybe_restart_bluetooth_radio(exc: Exception) -> bool:
    """Toggle Bluetooth off/on once when the radio is off, with cooldown."""
    global _last_radio_restart_at

    if not is_bluetooth_powered_off_error(exc):
        return False

    now = time.monotonic()
    if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
        debug_write("ble: radio restart skipped (cooldown)")
        return False

    lock = _get_radio_restart_lock()
    async with lock:
        now = time.monotonic()
        if _last_radio_restart_at is not None and now - _last_radio_restart_at < BT_RADIO_RESTART_COOLDOWN:
            debug_write("ble: radio restart skipped (cooldown)")
            return False
        try:
            restarted = await restart_bluetooth_radio()
        except Exception as restart_exc:  # noqa: BLE001
            debug_write_exception("ble: radio restart failed", restart_exc)
            return False
        if restarted:
            _last_radio_restart_at = time.monotonic()
        return restarted
