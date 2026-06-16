"""BLE access for ThermoPro TP35x sensors via bleak."""

from __future__ import annotations

import asyncio
import logging
import sys
from dataclasses import dataclass
from typing import Any

from bleak import BleakClient, BleakScanner
from bleak.backends.device import BLEDevice
from bleak.backends.scanner import AdvertisementData

from tp.config import normalize_mac
from tp.debug_log import write as debug_write
from tp.debug_log import write_exception as debug_write_exception

UUID_READ = "00010203-0405-0607-0809-0a0b0c0d2b10"

CONNECT_TIMEOUT = 45.0
DEVICE_SCAN_TIMEOUT = 10.0
POST_RESOLVE_DELAY = 1.0
INTER_STRATEGY_DELAY = 2.0
NOTIFY_TIMEOUT = 30.0
GATT_SETTLE_DELAY = 2.0
GATT_DISCOVERY_RETRIES = 8
READ_RETRIES = 3
READ_RETRY_DELAY = 3.0

DEVICE_READ_TIMEOUT = 60.0

NOW_OPCODE = 194

# WinRT / BlueZ adapters handle one active GATT session reliably.
_ble_session_lock: asyncio.Lock | None = None
_bleak_log_handler: logging.Handler | None = None


def _get_ble_session_lock() -> asyncio.Lock:
    global _ble_session_lock
    if _ble_session_lock is None:
        _ble_session_lock = asyncio.Lock()
    return _ble_session_lock


@dataclass(frozen=True)
class _ConnectStrategy:
    name: str
    use_scanned_device: bool
    client_kwargs: dict[str, Any]


def _winrt_kwargs(use_cached_services: bool | None) -> dict[str, Any]:
    if sys.platform != "win32" or use_cached_services is None:
        return {}
    return {"winrt": {"use_cached_services": use_cached_services}}


def _connect_strategies() -> list[_ConnectStrategy]:
    """Ordered connect attempts; Windows uses BLEDevice-only (address connect never wins)."""
    device_strategies = [
        _ConnectStrategy("device-default", True, {}),
        _ConnectStrategy("device-uncached", True, _winrt_kwargs(False)),
        _ConnectStrategy("device-cached", True, _winrt_kwargs(True)),
    ]
    if sys.platform == "win32":
        return device_strategies
    return [
        device_strategies[0],
        _ConnectStrategy("address-default", False, {}),
    ]


def configure_bleak_debug_logging(enabled: bool, log_path) -> None:
    """Mirror bleak backend logs into debug.log when troubleshooting."""
    global _bleak_log_handler
    bleak_logger = logging.getLogger("bleak")
    if _bleak_log_handler is not None:
        bleak_logger.removeHandler(_bleak_log_handler)
        _bleak_log_handler = None
    if not enabled or log_path is None:
        return
    handler = logging.FileHandler(log_path, encoding="utf-8")
    handler.setFormatter(
        logging.Formatter("%(asctime)s.%(msecs)03d bleak %(levelname)s %(message)s")
    )
    bleak_logger.addHandler(handler)
    bleak_logger.setLevel(logging.DEBUG)
    _bleak_log_handler = handler


@dataclass
class ScannedDevice:
    address: str
    name: str
    rssi: int | None = None


@dataclass
class NowReading:
    temp_f: float
    humidity_pct: int


def _celsius_to_fahrenheit(celsius: float) -> float:
    return celsius * 9.0 / 5.0 + 32.0


def _raw_temp_to_fahrenheit(raw: int) -> float:
    """Convert GATT raw temperature to Fahrenheit."""
    celsius = raw / 10.0
    return round(_celsius_to_fahrenheit(celsius), 1)


def _normalize_uuid(uuid: str) -> str:
    return uuid.lower().replace("-", "")


def _is_tp35_name(name: str | None) -> bool:
    return bool(name and name.upper().startswith("TP35"))


def _device_name(device: BLEDevice, adv: AdvertisementData | None = None) -> str:
    """Best available display name from OS device record or advertisement."""
    if device.name:
        return device.name
    if adv and adv.local_name:
        return adv.local_name
    return ""


def _address_matches(device: BLEDevice, target_mac: str) -> bool:
    return normalize_mac(device.address) == normalize_mac(target_mac)


async def _resolve_device(address: str) -> BLEDevice:
    """Find a BLE device record before connecting (required on Windows WinRT)."""
    target = normalize_mac(address)
    debug_write(f"ble: resolving device {target}")

    device = await BleakScanner.find_device_by_address(target, timeout=DEVICE_SCAN_TIMEOUT)
    if device is not None:
        debug_write(f"ble: found by address {device.address} name={device.name!r}")
        return device

    debug_write(f"ble: address lookup failed, scanning with filter for {target}")
    device = await BleakScanner.find_device_by_filter(
        lambda d, _adv: _address_matches(d, target),
        timeout=DEVICE_SCAN_TIMEOUT,
    )
    if device is not None:
        debug_write(f"ble: found by filter {device.address} name={device.name!r}")
        return device

    debug_write(f"ble: device not found after scan ({DEVICE_SCAN_TIMEOUT}s)")
    raise RuntimeError(f"Device with address {target} was not found")


async def scan_devices(timeout: float = 10.0) -> list[ScannedDevice]:
    debug_write(f"ble: scan_devices timeout={timeout}s")
    discovered = await BleakScanner.discover(timeout=timeout, return_adv=True)
    debug_write(f"ble: scan_devices saw {len(discovered)} advertisement(s)")
    results: list[ScannedDevice] = []
    seen: set[str] = set()
    for _address, (device, adv) in discovered.items():
        name = _device_name(device, adv)
        if not _is_tp35_name(name):
            continue
        address = normalize_mac(device.address)
        if address in seen:
            continue
        seen.add(address)
        rssi = adv.rssi if adv is not None else None
        results.append(ScannedDevice(address=address, name=name, rssi=rssi))
        debug_write(f"ble: scan match {name} @ {address} rssi={rssi}")
    results.sort(key=lambda d: d.name)
    debug_write(f"ble: scan_devices returning {len(results)} TP35 device(s)")
    return results


def _discovered_characteristic_uuids(client: BleakClient) -> list[str]:
    uuids: list[str] = []
    for service in client.services:
        for char in service.characteristics:
            uuids.append(char.uuid)
    return uuids


async def _wait_for_gatt_services(client: BleakClient) -> None:
    """Allow WinRT/BlueZ time to populate the GATT table after connect."""
    debug_write("ble: waiting for GATT services")
    try:
        services = list(client.services)
        if services:
            debug_write(f"ble: GATT immediately ready ({len(services)} service(s))")
            return
    except Exception as exc:  # noqa: BLE001
        debug_write_exception("ble: initial GATT service read failed", exc)

    await asyncio.sleep(GATT_SETTLE_DELAY)
    for attempt in range(GATT_DISCOVERY_RETRIES):
        try:
            services = list(client.services)
            if services:
                debug_write(f"ble: GATT ready ({len(services)} service(s)) on attempt {attempt + 1}")
                return
        except Exception as exc:  # noqa: BLE001
            debug_write_exception("ble: GATT discovery poll failed", exc)
        await asyncio.sleep(0.5)
    raise RuntimeError("GATT service discovery timed out")


async def _find_characteristic(client: BleakClient, uuid: str):
    target = _normalize_uuid(uuid)
    last_seen: list[str] = []
    for attempt in range(GATT_DISCOVERY_RETRIES):
        last_seen = _discovered_characteristic_uuids(client)
        for service in client.services:
            for char in service.characteristics:
                if _normalize_uuid(char.uuid) == target:
                    debug_write(f"ble: found characteristic {uuid} on attempt {attempt + 1}")
                    return char
        if attempt + 1 < GATT_DISCOVERY_RETRIES:
            await asyncio.sleep(0.5 * (attempt + 1))
    seen = ", ".join(last_seen[:8])
    if len(last_seen) > 8:
        seen += ", …"
    raise RuntimeError(
        f"Characteristic {uuid} not found"
        + (f" (discovered: {seen})" if seen else " (no characteristics discovered)")
    )


def _is_transient_ble_error(exc: Exception) -> bool:
    if isinstance(exc, (TimeoutError, asyncio.TimeoutError)):
        return True
    text = str(exc).lower()
    return any(
        token in text
        for token in (
            "timeout",
            "timed out",
            "disconnected",
            "connection",
            "unreachable",
            "gatt service discovery timed out",
            "was not found",
            "could not get gatt services",
            "incomplete reading",
        )
    )


def _raise_if_past_deadline(deadline: float | None, address: str) -> None:
    if deadline is None:
        return
    if asyncio.get_running_loop().time() >= deadline:
        raise TimeoutError(
            f"read_now({address}) timed out ({DEVICE_READ_TIMEOUT:.0f}s per device)"
        )


async def _retry_ble(
    operation,
    *,
    label: str,
    deadline: float | None = None,
    address: str = "",
):
    last_exc: Exception | None = None
    attempts = READ_RETRIES + 1
    for attempt in range(attempts):
        _raise_if_past_deadline(deadline, address)
        try:
            debug_write(f"ble: {label} attempt {attempt + 1}/{attempts}")
            return await operation()
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001
            last_exc = exc
            debug_write_exception(f"ble: {label} failed", exc)
            if attempt + 1 >= attempts or not _is_transient_ble_error(exc):
                raise
            _raise_if_past_deadline(deadline, address)
            await asyncio.sleep(READ_RETRY_DELAY * (attempt + 1))
    if last_exc is not None:
        raise last_exc
    raise RuntimeError(f"{label} failed")


def format_ble_error(exc: Exception) -> str:
    """Human-readable BLE/GATT error for the TUI."""
    if isinstance(exc, TimeoutError):
        return "Read timed out"
    text = str(exc).strip()
    if "Unreachable" in text:
        return "BLE unreachable (try moving sensor closer or removing it from Windows Bluetooth settings)"
    if exc.args:
        last = exc.args[-1]
        if isinstance(last, str) and last:
            return last
    if text.startswith("(") and "," in text:
        parts = text.rsplit(",", 1)
        if len(parts) == 2:
            return parts[1].strip(" )'\"")
    return text or exc.__class__.__name__


def _parse_now_buffer(buffer: list[int]) -> NowReading:
    if len(buffer) < 6:
        raise RuntimeError(f"Incomplete reading from sensor ({len(buffer)} bytes)")
    temp_raw = buffer[3] + buffer[4] * 256
    humidity = buffer[5]
    return NowReading(temp_f=_raw_temp_to_fahrenheit(temp_raw), humidity_pct=int(humidity))


async def _read_now_on_client(client: BleakClient, read_char) -> NowReading:
    loop = asyncio.get_running_loop()
    done = loop.create_future()
    buffer: list[int] = []

    def handler(_handle: int, data: bytearray) -> None:
        if not data:
            return
        debug_write(f"ble: notify {data.hex()}")
        if data[0] == NOW_OPCODE:
            buffer.extend(data)
            if not done.done():
                done.set_result(None)

    debug_write("ble: starting notify for live reading")
    await client.start_notify(read_char, handler)
    try:
        await asyncio.wait_for(done, timeout=NOTIFY_TIMEOUT)
    finally:
        await client.stop_notify(read_char)
    reading = _parse_now_buffer(buffer)
    debug_write(f"ble: reading temp={reading.temp_f}F humidity={reading.humidity_pct}%")
    return reading


async def _connect_and_read(
    address: str,
    device: BLEDevice | None,
    strategy: _ConnectStrategy,
) -> NowReading:
    if strategy.use_scanned_device:
        if device is None:
            raise RuntimeError("Missing BLEDevice for device connect strategy")
        target: str | BLEDevice = device
    else:
        target = normalize_mac(address)
    target_label = target.address if isinstance(target, BLEDevice) else target
    debug_write(f"ble: connect [{strategy.name}] -> {target_label}")
    async with BleakClient(target, timeout=CONNECT_TIMEOUT, **strategy.client_kwargs) as client:
        debug_write(f"ble: connected [{strategy.name}] is_connected={client.is_connected}")
        await _wait_for_gatt_services(client)
        read_char = await _find_characteristic(client, UUID_READ)
        return await _read_now_on_client(client, read_char)


async def _prepare_device(address: str) -> BLEDevice:
    """Resolve a BLEDevice and pause before connect."""
    device = await _resolve_device(address)
    debug_write(
        f"ble: resolved {device.address} name={device.name!r}; "
        f"waiting {POST_RESOLVE_DELAY}s before connect"
    )
    await asyncio.sleep(POST_RESOLVE_DELAY)
    return device


async def _read_now_session(address: str, *, deadline: float | None = None) -> NowReading:
    strategies = _connect_strategies()
    last_exc: Exception | None = None
    device: BLEDevice | None = None
    if strategies[0].use_scanned_device:
        _raise_if_past_deadline(deadline, address)
        device = await _prepare_device(address)

    for index, strategy in enumerate(strategies):
        _raise_if_past_deadline(deadline, address)
        try:
            connect_device = device if strategy.use_scanned_device else None
            return await _connect_and_read(address, connect_device, strategy)
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001
            last_exc = exc
            debug_write_exception(f"ble: strategy {strategy.name} failed", exc)
            if not _is_transient_ble_error(exc) or index + 1 >= len(strategies):
                break
            _raise_if_past_deadline(deadline, address)
            await asyncio.sleep(INTER_STRATEGY_DELAY)
            next_strategy = strategies[index + 1]
            if next_strategy.use_scanned_device:
                device = await _prepare_device(address)

    if last_exc is not None:
        raise last_exc
    raise RuntimeError(f"read_now({address}) failed")


async def read_now(address: str) -> NowReading:
    """Connect and read current temperature/humidity."""
    loop = asyncio.get_running_loop()
    deadline = loop.time() + DEVICE_READ_TIMEOUT
    async with _get_ble_session_lock():
        try:
            return await asyncio.wait_for(
                _retry_ble(
                    lambda: _read_now_session(address, deadline=deadline),
                    label=f"read_now({address})",
                    deadline=deadline,
                    address=address,
                ),
                timeout=DEVICE_READ_TIMEOUT,
            )
        except TimeoutError:
            debug_write(
                f"ble: read_now({address}) timed out after {DEVICE_READ_TIMEOUT:.0f}s"
            )
            raise


def inter_device_delay_seconds() -> float:
    """Pause between device reads so the adapter can settle."""
    return 2.0
