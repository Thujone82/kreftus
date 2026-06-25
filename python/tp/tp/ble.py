"""BLE access for ThermoPro TP35x sensors via bleak."""

from __future__ import annotations

import asyncio
import logging
import sys
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import Any

from bleak import BleakClient, BleakScanner
from bleak.backends.device import BLEDevice
from bleak.backends.scanner import AdvertisementData

from tp.config import normalize_mac
from tp.debug_log import write as debug_write
from tp.debug_log import write_exception as debug_write_exception
from tp.ble_radio import is_bluetooth_powered_off_error, maybe_restart_bluetooth_radio
from tp.history import BLE_HISTORY_HOURS, Reading

UUID_READ = "00010203-0405-0607-0809-0a0b0c0d2b10"
UUID_WRITE = "00010203-0405-0607-0809-0a0b0c0d2b11"

CONNECT_TIMEOUT = 45.0
DEVICE_SCAN_TIMEOUT = 5.0
DEVICE_SCAN_TIMEOUT_EXTENDED = 10.0
POST_RESOLVE_DELAY = 0.35
POST_RESOLVE_DELAY_CACHED = 0.0
INTER_STRATEGY_DELAY = 2.0
NOTIFY_TIMEOUT = 30.0
NOW_SYNC_NOTIFY_TIMEOUT = 10.0
NOW_PASSIVE_NOTIFY_TIMEOUT = NOTIFY_TIMEOUT
GATT_SETTLE_DELAY = 1.0
GATT_DISCOVERY_RETRIES = 8
READ_RETRIES = 3
READ_RETRY_DELAY = 3.0
RESOLVE_CACHE_TTL = 120.0

DEVICE_READ_TIMEOUT = 60.0
DAY_READ_TIMEOUT = 180.0
DAY_STREAM_IDLE_SECONDS = 3.0
DAY_CMD_RESPONSE_SECONDS = 12.0

NOW_OPCODE = 194
NOW_READ_CONNECTING = "connecting"
NOW_READ_SYNC = "sync"
NOW_READ_PASSIVE = "passive"
DAY_OPCODE = 0xA7
DAY_CMD_PRIMARY = bytes([DAY_OPCODE, 0x01, 0x00, 0x7A])
DAY_CMD_FALLBACK = bytes([DAY_OPCODE, 0x00, 0x00, 0x00, 0x00, 0x7A])

DATETIME_SYNC_OPCODE = 0xA5
STREAM_MAGIC_PREFIX = b"\xcc\xcc"
STREAM_MAGIC_SUFFIX = b"\x66\x66"
DAY_STREAM_RECORD_COUNT = int(BLE_HISTORY_HOURS * 60)
INCREMENTAL_HISTORY_MAX_RECORDS = 1440
RECENT_HISTORY_BASE_TIMEOUT = 60.0
RECENT_HISTORY_PER_RECORD_TIMEOUT = 0.05
RECENT_HISTORY_MAX_TIMEOUT = 120.0
DAY_STREAM_CMD_DELAY = 0.2
DATETIME_SYNC_DELAY = 1.0

def recent_history_timeout(record_count: int) -> float:
    """BLE timeout scaled to the number of minute records requested."""
    bounded = max(1, min(record_count, INCREMENTAL_HISTORY_MAX_RECORDS))
    return min(
        RECENT_HISTORY_MAX_TIMEOUT,
        RECENT_HISTORY_BASE_TIMEOUT + bounded * RECENT_HISTORY_PER_RECORD_TIMEOUT,
    )


NowReadPhaseCallback = Callable[[str], Awaitable[None] | None]

# WinRT / BlueZ adapters handle one active GATT session reliably.
_ble_session_lock: asyncio.Lock | None = None
_bleak_log_handler: logging.Handler | None = None


@dataclass
class _DeviceCacheEntry:
    device: BLEDevice
    cached_at: float
    preferred_strategy: str | None = None


_device_cache: dict[str, _DeviceCacheEntry] = {}


def _cache_mac(address: str) -> str:
    return normalize_mac(address)


def _loop_time() -> float:
    return time.monotonic()


def get_cached_ble_device(address: str) -> BLEDevice | None:
    """Return a recently resolved BLEDevice, if still within TTL."""
    entry = _device_cache.get(_cache_mac(address))
    if entry is None:
        return None
    if _loop_time() - entry.cached_at > RESOLVE_CACHE_TTL:
        _device_cache.pop(_cache_mac(address), None)
        return None
    return entry.device


def _preferred_strategy_name(address: str) -> str | None:
    entry = _device_cache.get(_cache_mac(address))
    if entry is None:
        return None
    return entry.preferred_strategy


def _remember_ble_device(
    address: str,
    device: BLEDevice,
    *,
    strategy: str | None = None,
) -> None:
    mac = _cache_mac(address)
    previous = _device_cache.get(mac)
    preferred = strategy or (previous.preferred_strategy if previous else None)
    _device_cache[mac] = _DeviceCacheEntry(
        device=device,
        cached_at=_loop_time(),
        preferred_strategy=preferred,
    )


def _invalidate_ble_device_cache(address: str) -> None:
    _device_cache.pop(_cache_mac(address), None)


def clear_ble_device_cache() -> None:
    """Drop all cached BLEDevice records (e.g. after radio restart)."""
    _device_cache.clear()


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


def _connect_strategies_for(address: str | None = None) -> list[_ConnectStrategy]:
    """Prefer the last successful WinRT strategy for repeat connects."""
    strategies = _connect_strategies()
    if not address:
        return strategies
    preferred_name = _preferred_strategy_name(address)
    if not preferred_name:
        return strategies
    preferred = next((item for item in strategies if item.name == preferred_name), None)
    if preferred is None:
        return strategies
    rest = [item for item in strategies if item.name != preferred_name]
    return [preferred, *rest]


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


@dataclass
class DayHistoryProgress:
    phase: str
    packets: int = 0
    samples: int = 0
    message: str = ""


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


async def _await_with_radio_recovery(operation):
    """Run a scan operation; power-cycle Bluetooth once if the radio is off."""
    try:
        return await operation()
    except Exception as exc:  # noqa: BLE001
        if await maybe_restart_bluetooth_radio(exc):
            _device_cache.clear()
            debug_write("ble: retrying scan after Bluetooth radio restart")
            return await operation()
        raise


async def _resolve_device(
    address: str,
    *,
    timeout: float = DEVICE_SCAN_TIMEOUT,
) -> BLEDevice:
    """Find a BLE device record before connecting (required on Windows WinRT)."""
    target = normalize_mac(address)
    debug_write(f"ble: resolving device {target} (timeout={timeout:.0f}s)")

    async def find_by_address() -> BLEDevice | None:
        return await BleakScanner.find_device_by_address(target, timeout=timeout)

    device = await _await_with_radio_recovery(find_by_address)
    if device is not None:
        debug_write(f"ble: found by address {device.address} name={device.name!r}")
        return device

    debug_write(f"ble: address lookup failed, scanning with filter for {target}")

    async def find_by_filter() -> BLEDevice | None:
        return await BleakScanner.find_device_by_filter(
            lambda d, _adv: _address_matches(d, target),
            timeout=timeout,
        )

    device = await _await_with_radio_recovery(find_by_filter)
    if device is not None:
        debug_write(f"ble: found by filter {device.address} name={device.name!r}")
        return device

    debug_write(f"ble: device not found after scan ({timeout:.0f}s)")
    raise RuntimeError(f"Device with address {target} was not found")


async def prefetch_ble_device(address: str) -> None:
    """Warm the resolution cache during idle gaps between device reads."""
    if get_cached_ble_device(address) is not None:
        debug_write(f"ble: prefetch skip (cached) {_cache_mac(address)}")
        return
    try:
        device = await _resolve_device(address, timeout=DEVICE_SCAN_TIMEOUT)
        _remember_ble_device(address, device)
        debug_write(f"ble: prefetch cached {device.address}")
    except Exception as exc:  # noqa: BLE001
        debug_write_exception(f"ble: prefetch failed {_cache_mac(address)}", exc)


async def scan_devices(timeout: float = 10.0) -> list[ScannedDevice]:
    debug_write(f"ble: scan_devices timeout={timeout}s")

    async def discover() -> dict:
        return await BleakScanner.discover(timeout=timeout, return_adv=True)

    discovered = await _await_with_radio_recovery(discover)
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


def _raise_if_past_deadline(
    deadline: float | None,
    address: str,
    *,
    label: str = "read",
    timeout: float = DEVICE_READ_TIMEOUT,
) -> None:
    if deadline is None:
        return
    if asyncio.get_running_loop().time() >= deadline:
        raise TimeoutError(f"{label}({address}) timed out ({timeout:.0f}s)")


async def _emit_day_progress(
    progress: DayHistoryProgressCallback | None,
    update: DayHistoryProgress,
) -> None:
    if progress is None:
        return
    maybe = progress(update)
    if asyncio.iscoroutine(maybe):
        await maybe


async def _emit_now_phase(
    progress: NowReadPhaseCallback | None,
    phase: str,
) -> None:
    if progress is None:
        return
    maybe = progress(phase)
    if asyncio.iscoroutine(maybe):
        await maybe


def _day_history_t0(now: datetime | None = None) -> datetime:
    anchor = now or datetime.now()
    return anchor - timedelta(days=1) + timedelta(minutes=1)


def _parse_day_packet(data: list[int], t0: datetime) -> list[Reading]:
    if len(data) < 4 or data[0] != DAY_OPCODE:
        return []
    packet_index = data[1] + data[2] * 256
    readings: list[Reading] = []
    for sample_offset in range(5):
        ofs = 4 + sample_offset * 3
        if ofs + 2 >= len(data):
            break
        temp_raw = data[ofs] + data[ofs + 1] * 256
        humidity = data[ofs + 2]
        if temp_raw == 0xFFFF or (data[ofs] == 0xFF and data[ofs + 1] == 0xFF):
            continue
        if temp_raw > 1024 or humidity > 100 or humidity == 0xFF:
            continue
        minute_index = 5 * (packet_index - 1) + sample_offset
        readings.append(
            Reading(
                timestamp=t0 + timedelta(minutes=minute_index),
                temp_f=_raw_temp_to_fahrenheit(temp_raw),
                humidity_pct=int(humidity),
            )
        )
    return readings


def _parse_day_packets(packets: list[list[int]], *, t0: datetime | None = None) -> list[Reading]:
    anchor = t0 or _day_history_t0()
    merged: dict[datetime, Reading] = {}
    for packet in packets:
        for reading in _parse_day_packet(packet, anchor):
            merged[reading.timestamp] = reading
    return sorted(merged.values(), key=lambda item: item.timestamp)


def _make_datetime_sync_cmd(now: datetime | None = None) -> bytes:
    """TP357S/TP359 datetime handshake (required before stream history)."""
    anchor = now or datetime.now()
    payload = bytes(
        [
            DATETIME_SYNC_OPCODE,
            anchor.year % 100,
            anchor.month,
            anchor.day,
            anchor.hour,
            anchor.minute,
            anchor.second,
            anchor.weekday() + 1,
        ]
    )
    return payload + bytes([sum(payload) & 0xFF])


def _make_stream_history_cmds(
    count: int,
    now: datetime | None = None,
) -> tuple[bytes, bytes, bytes]:
    """TP357S/TP359 three-command history request sequence."""
    anchor = now or datetime.now()
    lo = count & 0xFF
    hi = (count >> 8) & 0xFF
    cmd1 = bytes.fromhex("cccc0201000001046666")
    cmd2 = bytes.fromhex("cccc04000000046666")
    body = bytes(
        [
            0x01,
            0x09,
            0x00,
            0x00,
            0x00,
            anchor.year % 100,
            anchor.month,
            anchor.day,
            anchor.hour,
            anchor.minute,
            anchor.second,
            lo,
            hi,
        ]
    )
    checksum = sum(body) & 0xFF
    cmd3 = STREAM_MAGIC_PREFIX + body + bytes([checksum]) + STREAM_MAGIC_SUFFIX
    return cmd1, cmd2, cmd3


def _decode_stream_history_chunks(chunks: list[bytes]) -> list[tuple[float, int]]:
    """Decode TP357S/TP359 stream chunks into (temp_c, humidity) pairs."""
    buffer = b"".join(chunks)
    if not buffer.startswith(STREAM_MAGIC_PREFIX):
        return []
    buffer = buffer[2:]
    if buffer.endswith(STREAM_MAGIC_SUFFIX):
        buffer = buffer[:-2]
    if len(buffer) < 6:
        return []
    pairs_raw = buffer[5:-1]
    readings: list[tuple[float, int]] = []
    for index in range(len(pairs_raw) // 3):
        raw = pairs_raw[index * 3 : index * 3 + 3]
        temp_c = int.from_bytes(raw[0:2], "little", signed=True) / 10.0
        humidity = raw[2]
        if humidity > 100:
            continue
        readings.append((temp_c, int(humidity)))
    return readings


def _stream_history_to_readings(
    pairs: list[tuple[float, int]],
    fetch_time: datetime | None = None,
) -> list[Reading]:
    """Assign minute timestamps to stream history (most-recent-first input)."""
    if not pairs:
        return []
    anchor = (fetch_time or datetime.now()).replace(second=0, microsecond=0)
    merged: dict[datetime, Reading] = {}
    for index, (temp_c, humidity) in enumerate(pairs):
        timestamp = anchor - timedelta(minutes=index)
        merged[timestamp] = Reading(
            timestamp=timestamp,
            temp_f=round(_celsius_to_fahrenheit(temp_c), 1),
            humidity_pct=humidity,
        )
    return sorted(merged.values(), key=lambda item: item.timestamp)


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
            if is_bluetooth_powered_off_error(exc):
                if await maybe_restart_bluetooth_radio(exc):
                    _device_cache.clear()
                    debug_write(f"ble: {label} retrying after Bluetooth radio restart")
                    _raise_if_past_deadline(deadline, address)
                    continue
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
    if is_bluetooth_powered_off_error(exc):
        return "Bluetooth is off — enable Bluetooth in system settings"
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
    temp_c = int.from_bytes(bytes(buffer[3:5]), "little", signed=True) / 10.0
    humidity = buffer[5]
    return NowReading(
        temp_f=round(_celsius_to_fahrenheit(temp_c), 1),
        humidity_pct=int(humidity),
    )


async def _collect_live_notify(
    client: BleakClient,
    read_char,
    *,
    trigger: Callable[[], Awaitable[None]] | None = None,
    timeout: float,
    label: str,
) -> list[int]:
    """Subscribe on the read characteristic and wait for a 0xC2 live packet."""
    loop = asyncio.get_running_loop()
    done = loop.create_future()
    buffer: list[int] = []

    def handler(_handle: int, data: bytearray) -> None:
        if not data:
            return
        debug_write(f"ble: notify ({label}) {data.hex()}")
        if data[0] == NOW_OPCODE:
            buffer.clear()
            buffer.extend(data)
            if not done.done():
                done.set_result(None)

    debug_write(f"ble: starting notify for live reading ({label})")
    await client.start_notify(read_char, handler)
    try:
        if trigger is not None:
            await trigger()
        await asyncio.wait_for(done, timeout=timeout)
    finally:
        await client.stop_notify(read_char)
    return buffer


async def _read_now_passive_on_client(
    client: BleakClient,
    read_char,
    *,
    progress: NowReadPhaseCallback | None = None,
) -> NowReading:
    """Legacy passive live read: subscribe and wait for an unsolicited 0xC2 notify."""
    await _emit_now_phase(progress, NOW_READ_PASSIVE)
    buffer = await _collect_live_notify(
        client,
        read_char,
        timeout=NOW_PASSIVE_NOTIFY_TIMEOUT,
        label="passive",
    )
    reading = _parse_now_buffer(buffer)
    debug_write(
        f"ble: reading temp={reading.temp_f}F humidity={reading.humidity_pct}% (passive)"
    )
    return reading


async def _read_now_sync_on_client(
    client: BleakClient,
    read_char,
    write_char,
    *,
    progress: NowReadPhaseCallback | None = None,
) -> NowReading:
    """Fast live read: datetime sync write prompts an immediate 0xC2 (TP357S/TP359)."""
    await _emit_now_phase(progress, NOW_READ_SYNC)

    async def trigger() -> None:
        sync_cmd = _make_datetime_sync_cmd()
        debug_write(f"ble: live datetime sync {sync_cmd.hex()}")
        await client.write_gatt_char(write_char, sync_cmd, response=False)

    buffer = await _collect_live_notify(
        client,
        read_char,
        trigger=trigger,
        timeout=NOW_SYNC_NOTIFY_TIMEOUT,
        label="sync",
    )
    reading = _parse_now_buffer(buffer)
    debug_write(
        f"ble: reading temp={reading.temp_f}F humidity={reading.humidity_pct}% (sync)"
    )
    return reading


async def _read_now_on_client(
    client: BleakClient,
    read_char,
    write_char,
    *,
    progress: NowReadPhaseCallback | None = None,
) -> NowReading:
    """Try sync-prompted live read first; fall back to passive notify for legacy sensors."""
    try:
        return await _read_now_sync_on_client(
            client,
            read_char,
            write_char,
            progress=progress,
        )
    except TimeoutError:
        debug_write("ble: sync live read timed out; falling back to passive notify")
    except Exception as exc:  # noqa: BLE001
        debug_write_exception("ble: sync live read failed; falling back to passive notify", exc)
    return await _read_now_passive_on_client(client, read_char, progress=progress)


async def _connect_and_read(
    address: str,
    device: BLEDevice | None,
    strategy: _ConnectStrategy,
    *,
    progress: NowReadPhaseCallback | None = None,
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
        write_char = await _find_characteristic(client, UUID_WRITE)
        return await _read_now_on_client(client, read_char, write_char, progress=progress)


async def _prepare_device(address: str) -> BLEDevice:
    """Resolve a BLEDevice and pause briefly before connect."""
    cached = get_cached_ble_device(address)
    if cached is not None:
        debug_write(f"ble: using cached device {cached.address}")
        if POST_RESOLVE_DELAY_CACHED:
            await asyncio.sleep(POST_RESOLVE_DELAY_CACHED)
        return cached

    try:
        device = await _resolve_device(address, timeout=DEVICE_SCAN_TIMEOUT)
    except RuntimeError:
        debug_write(
            f"ble: quick scan missed {_cache_mac(address)}; "
            f"retrying ({DEVICE_SCAN_TIMEOUT_EXTENDED:.0f}s)"
        )
        device = await _resolve_device(address, timeout=DEVICE_SCAN_TIMEOUT_EXTENDED)
    _remember_ble_device(address, device)
    debug_write(
        f"ble: resolved {device.address} name={device.name!r}; "
        f"waiting {POST_RESOLVE_DELAY}s before connect"
    )
    await asyncio.sleep(POST_RESOLVE_DELAY)
    return device


async def _run_now_connect_strategies(
    address: str,
    device: BLEDevice | None,
    *,
    deadline: float | None,
    progress: NowReadPhaseCallback | None,
    allow_reprepare: bool = True,
) -> NowReading:
    strategies = _connect_strategies_for(address)
    last_exc: Exception | None = None
    current_device = device

    for index, strategy in enumerate(strategies):
        _raise_if_past_deadline(deadline, address)
        try:
            connect_device = current_device if strategy.use_scanned_device else None
            reading = await _connect_and_read(
                address,
                connect_device,
                strategy,
                progress=progress,
            )
            if connect_device is not None:
                _remember_ble_device(address, connect_device, strategy=strategy.name)
            return reading
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001
            last_exc = exc
            debug_write_exception(f"ble: strategy {strategy.name} failed", exc)
            if not _is_transient_ble_error(exc) or index + 1 >= len(strategies):
                break
            _raise_if_past_deadline(deadline, address)
            await asyncio.sleep(INTER_STRATEGY_DELAY)
            if not allow_reprepare:
                break
            next_strategy = strategies[index + 1]
            if next_strategy.use_scanned_device:
                _invalidate_ble_device_cache(address)
                await _emit_now_phase(progress, NOW_READ_CONNECTING)
                current_device = await _prepare_device(address)
            else:
                await _emit_now_phase(progress, NOW_READ_CONNECTING)

    if last_exc is not None:
        raise last_exc
    raise RuntimeError(f"read_now({address}) failed")


async def _read_now_session(
    address: str,
    *,
    deadline: float | None = None,
    progress: NowReadPhaseCallback | None = None,
) -> NowReading:
    await _emit_now_phase(progress, NOW_READ_CONNECTING)

    cached = get_cached_ble_device(address)
    if cached is not None:
        try:
            return await _run_now_connect_strategies(
                address,
                cached,
                deadline=deadline,
                progress=progress,
                allow_reprepare=False,
            )
        except Exception as exc:  # noqa: BLE001
            if not _is_transient_ble_error(exc):
                raise
            debug_write(f"ble: cached device connect failed for {address}; re-resolving")
            _invalidate_ble_device_cache(address)

    strategies = _connect_strategies_for(address)
    device: BLEDevice | None = None
    if strategies[0].use_scanned_device:
        _raise_if_past_deadline(deadline, address)
        device = await _prepare_device(address)

    return await _run_now_connect_strategies(
        address,
        device,
        deadline=deadline,
        progress=progress,
    )


async def _read_day_stream_on_client(
    client: BleakClient,
    read_char,
    write_char,
    *,
    record_count: int = DAY_STREAM_RECORD_COUNT,
    progress: DayHistoryProgressCallback | None = None,
    deadline: float | None = None,
    address: str = "",
) -> list[Reading]:
    """Read day history via TP357S/TP359 stream protocol (pytp357s-compatible)."""
    loop = asyncio.get_running_loop()
    stream_done = loop.create_future()
    chunks: list[bytes] = []

    def maybe_finish() -> None:
        if chunks and not stream_done.done():
            stream_done.set_result(None)

    def handler(_handle: int, data: bytearray) -> None:
        if not data:
            return
        packet = bytes(data)
        debug_write(f"ble: day stream notify {packet.hex()}")
        if packet.startswith(STREAM_MAGIC_PREFIX):
            chunks.clear()
            chunks.append(packet)
        elif chunks:
            chunks.append(packet)
        if chunks and chunks[-1].endswith(STREAM_MAGIC_SUFFIX):
            maybe_finish()

    await _emit_day_progress(
        progress,
        DayHistoryProgress(phase="receiving", message="Syncing device clock…"),
    )
    sync_cmd = _make_datetime_sync_cmd()
    debug_write(f"ble: day stream datetime sync {sync_cmd.hex()}")
    await client.write_gatt_char(write_char, sync_cmd, response=False)
    await asyncio.sleep(DATETIME_SYNC_DELAY)

    await client.start_notify(read_char, handler)
    try:
        history_cmds = _make_stream_history_cmds(record_count)
        await _emit_day_progress(
            progress,
            DayHistoryProgress(
                phase="receiving",
                message=f"Requesting up to {record_count} records…",
            ),
        )
        for cmd_index, cmd in enumerate(history_cmds):
            debug_write(f"ble: day stream write cmd[{cmd_index}] {cmd.hex()}")
            await client.write_gatt_char(write_char, cmd, response=False)
            if cmd_index + 1 < len(history_cmds):
                await asyncio.sleep(DAY_STREAM_CMD_DELAY)

        while not stream_done.done():
            _raise_if_past_deadline(
                deadline,
                address,
                label="read_day_history",
                timeout=DAY_READ_TIMEOUT,
            )
            decoded = _decode_stream_history_chunks(chunks)
            await _emit_day_progress(
                progress,
                DayHistoryProgress(
                    phase="receiving",
                    packets=len(chunks),
                    samples=len(decoded),
                    message=f"Receiving history ({len(decoded)} samples)",
                ),
            )
            try:
                await asyncio.wait_for(asyncio.shield(stream_done), timeout=0.25)
                break
            except TimeoutError:
                continue
    finally:
        await client.stop_notify(read_char)

    fetch_time = datetime.now().replace(second=0, microsecond=0)
    pairs = _decode_stream_history_chunks(chunks)
    readings = _stream_history_to_readings(pairs, fetch_time)
    debug_write(
        f"ble: day stream decoded {len(readings)} sample(s) from {len(chunks)} chunk(s)"
    )
    return readings


async def _read_day_legacy_on_client(
    client: BleakClient,
    read_char,
    write_char,
    *,
    progress: DayHistoryProgressCallback | None = None,
    deadline: float | None = None,
    address: str = "",
) -> list[list[int]]:
    loop = asyncio.get_running_loop()
    stream_done = loop.create_future()
    packets: list[list[int]] = []
    last_packet_at = loop.time()

    def maybe_finish() -> None:
        if packets and not stream_done.done():
            stream_done.set_result(None)

    def handler(_handle: int, data: bytearray) -> None:
        nonlocal last_packet_at
        if not data:
            return
        last_packet_at = loop.time()
        debug_write(f"ble: day notify {data.hex()}")
        if data[0] == DAY_OPCODE:
            packets.append(list(data))
            return
        if data[0] == NOW_OPCODE:
            maybe_finish()
            return
        if packets:
            maybe_finish()

    await _emit_day_progress(
        progress,
        DayHistoryProgress(phase="receiving", message="Requesting 72H history…"),
    )
    await client.start_notify(read_char, handler)
    await asyncio.sleep(GATT_SETTLE_DELAY)
    try:
        for cmd_index, day_cmd in enumerate((DAY_CMD_PRIMARY, DAY_CMD_FALLBACK)):
            packets.clear()
            stream_done = loop.create_future()
            cmd_started_at = loop.time()
            last_packet_at = cmd_started_at
            debug_write(f"ble: day write cmd[{cmd_index}] {day_cmd.hex()}")
            await client.write_gatt_char(write_char, day_cmd, response=False)
            while not stream_done.done():
                _raise_if_past_deadline(
                    deadline,
                    address,
                    label="read_day_history",
                    timeout=DAY_READ_TIMEOUT,
                )
                if (
                    not packets
                    and loop.time() - cmd_started_at >= DAY_CMD_RESPONSE_SECONDS
                ):
                    debug_write(
                        f"ble: day cmd[{cmd_index}] no packets after "
                        f"{DAY_CMD_RESPONSE_SECONDS:.0f}s"
                    )
                    break
                if packets and loop.time() - last_packet_at >= DAY_STREAM_IDLE_SECONDS:
                    maybe_finish()
                parsed = _parse_day_packets(packets)
                await _emit_day_progress(
                    progress,
                    DayHistoryProgress(
                        phase="receiving",
                        packets=len(packets),
                        samples=len(parsed),
                        message=f"Receiving history ({len(packets)} packets)",
                    ),
                )
                try:
                    await asyncio.wait_for(asyncio.shield(stream_done), timeout=0.25)
                    break
                except TimeoutError:
                    continue
            if packets:
                break
            debug_write(f"ble: day cmd[{cmd_index}] returned no packets")
        if not packets:
            raise RuntimeError("No day-history packets received from sensor")
    finally:
        await client.stop_notify(read_char)
    return packets


async def _connect_and_read_day(
    address: str,
    device: BLEDevice | None,
    strategy: _ConnectStrategy,
    *,
    record_count: int = DAY_STREAM_RECORD_COUNT,
    progress: DayHistoryProgressCallback | None = None,
    deadline: float | None = None,
) -> list[Reading]:
    if strategy.use_scanned_device:
        if device is None:
            raise RuntimeError("Missing BLEDevice for device connect strategy")
        target: str | BLEDevice = device
    else:
        target = normalize_mac(address)
    target_label = target.address if isinstance(target, BLEDevice) else target
    debug_write(f"ble: day connect [{strategy.name}] -> {target_label}")
    async with BleakClient(target, timeout=CONNECT_TIMEOUT, **strategy.client_kwargs) as client:
        debug_write(f"ble: day connected [{strategy.name}] is_connected={client.is_connected}")
        await _wait_for_gatt_services(client)
        read_char = await _find_characteristic(client, UUID_READ)
        write_char = await _find_characteristic(client, UUID_WRITE)
        readings = await _read_day_stream_on_client(
            client,
            read_char,
            write_char,
            record_count=record_count,
            progress=progress,
            deadline=deadline,
            address=address,
        )
        if readings:
            await _emit_day_progress(
                progress,
                DayHistoryProgress(
                    phase="parsing",
                    packets=0,
                    samples=len(readings),
                    message="History stream complete",
                ),
            )
            return readings

        if record_count < DAY_STREAM_RECORD_COUNT:
            raise RuntimeError("Stream history unavailable for incremental poll")

        debug_write("ble: day stream empty; falling back to legacy 0xA7 protocol")
        packets = await _read_day_legacy_on_client(
            client,
            read_char,
            write_char,
            progress=progress,
            deadline=deadline,
            address=address,
        )
        await _emit_day_progress(
            progress,
            DayHistoryProgress(
                phase="parsing",
                packets=len(packets),
                message="Parsing legacy history packets…",
            ),
        )
        readings = _parse_day_packets(packets)
        if record_count < DAY_STREAM_RECORD_COUNT:
            cutoff = datetime.now().replace(second=0, microsecond=0) - timedelta(
                minutes=record_count - 1
            )
            readings = [reading for reading in readings if reading.timestamp >= cutoff]
        debug_write(f"ble: day parsed {len(readings)} sample(s) from {len(packets)} packet(s)")
        return readings


async def _read_day_session(
    address: str,
    *,
    record_count: int = DAY_STREAM_RECORD_COUNT,
    progress: DayHistoryProgressCallback | None = None,
    deadline: float | None = None,
    timeout_label: str = "read_day_history",
) -> list[Reading]:
    await _emit_day_progress(
        progress,
        DayHistoryProgress(phase="connecting", message="Connecting to sensor…"),
    )

    cached = get_cached_ble_device(address)
    if cached is not None:
        try:
            return await _run_day_connect_strategies(
                address,
                cached,
                record_count=record_count,
                progress=progress,
                deadline=deadline,
                allow_reprepare=False,
                timeout_label=timeout_label,
            )
        except Exception as exc:  # noqa: BLE001
            if not _is_transient_ble_error(exc):
                raise
            debug_write(f"ble: cached day connect failed for {address}; re-resolving")
            _invalidate_ble_device_cache(address)

    strategies = _connect_strategies_for(address)
    device: BLEDevice | None = None
    if strategies[0].use_scanned_device:
        _raise_if_past_deadline(
            deadline,
            address,
            label=timeout_label,
            timeout=DAY_READ_TIMEOUT,
        )
        device = await _prepare_device(address)

    return await _run_day_connect_strategies(
        address,
        device,
        record_count=record_count,
        progress=progress,
        deadline=deadline,
        timeout_label=timeout_label,
    )


async def _run_day_connect_strategies(
    address: str,
    device: BLEDevice | None,
    *,
    record_count: int = DAY_STREAM_RECORD_COUNT,
    progress: DayHistoryProgressCallback | None = None,
    deadline: float | None = None,
    allow_reprepare: bool = True,
    timeout_label: str = "read_day_history",
) -> list[Reading]:
    strategies = _connect_strategies_for(address)
    last_exc: Exception | None = None
    current_device = device

    for index, strategy in enumerate(strategies):
        _raise_if_past_deadline(
            deadline,
            address,
            label=timeout_label,
            timeout=DAY_READ_TIMEOUT,
        )
        try:
            connect_device = current_device if strategy.use_scanned_device else None
            readings = await _connect_and_read_day(
                address,
                connect_device,
                strategy,
                record_count=record_count,
                progress=progress,
                deadline=deadline,
            )
            if connect_device is not None:
                _remember_ble_device(address, connect_device, strategy=strategy.name)
            return readings
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001
            last_exc = exc
            debug_write_exception(f"ble: day strategy {strategy.name} failed", exc)
            if not _is_transient_ble_error(exc) or index + 1 >= len(strategies):
                break
            _raise_if_past_deadline(
                deadline,
                address,
                label="read_day_history",
                timeout=DAY_READ_TIMEOUT,
            )
            await asyncio.sleep(INTER_STRATEGY_DELAY)
            if not allow_reprepare:
                break
            next_strategy = strategies[index + 1]
            if next_strategy.use_scanned_device:
                _invalidate_ble_device_cache(address)
                current_device = await _prepare_device(address)

    if last_exc is not None:
        raise last_exc
    raise RuntimeError(f"read_day_history({address}) failed")


async def _read_history_locked(
    address: str,
    *,
    record_count: int,
    progress: DayHistoryProgressCallback | None = None,
    timeout: float,
    label: str,
) -> list[Reading]:
    loop = asyncio.get_running_loop()
    deadline = loop.time() + timeout
    async with _get_ble_session_lock():
        try:
            return await asyncio.wait_for(
                _retry_ble(
                    lambda: _read_day_session(
                        address,
                        record_count=record_count,
                        progress=progress,
                        deadline=deadline,
                        timeout_label=label,
                    ),
                    label=f"{label}({address})",
                    deadline=deadline,
                    address=address,
                ),
                timeout=timeout,
            )
        except TimeoutError:
            debug_write(f"ble: {label}({address}) timed out after {timeout:.0f}s")
            raise


async def read_day_history(
    address: str,
    *,
    progress: DayHistoryProgressCallback | None = None,
) -> list[Reading]:
    """Connect and read minute-resolution 72H history from the sensor."""
    return await _read_history_locked(
        address,
        record_count=DAY_STREAM_RECORD_COUNT,
        progress=progress,
        timeout=DAY_READ_TIMEOUT,
        label="read_day_history",
    )


async def read_recent_history(
    address: str,
    record_count: int,
    *,
    progress: DayHistoryProgressCallback | None = None,
) -> list[Reading]:
    """Read the most recent minute-resolution history records from the sensor."""
    bounded = max(1, min(int(record_count), INCREMENTAL_HISTORY_MAX_RECORDS))
    timeout = recent_history_timeout(bounded)
    return await _read_history_locked(
        address,
        record_count=bounded,
        progress=progress,
        timeout=timeout,
        label="read_recent_history",
    )


async def read_now(
    address: str,
    *,
    progress: NowReadPhaseCallback | None = None,
) -> NowReading:
    """Connect and read current temperature/humidity."""
    loop = asyncio.get_running_loop()
    deadline = loop.time() + DEVICE_READ_TIMEOUT
    async with _get_ble_session_lock():
        try:
            return await asyncio.wait_for(
                _retry_ble(
                    lambda: _read_now_session(
                        address,
                        deadline=deadline,
                        progress=progress,
                    ),
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
