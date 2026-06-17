"""Reading history and CSV logging."""

from __future__ import annotations

import csv
import os
import tempfile
from dataclasses import dataclass
from datetime import datetime, timedelta
from pathlib import Path

from tp.config import AppConfig, normalize_mac, probe_log_path, resolved_log_path

CSV_HEADER = ["timestamp", "device", "temp_f", "humidity_pct", "mac"]
LOG_TIMESTAMP_FORMAT = "%Y-%m-%d %H:%M:%S"
LOG_HISTORY_HOURS = 24
MIN_DAY_HISTORY_SAMPLES = 100


@dataclass
class Reading:
    timestamp: datetime
    temp_f: float
    humidity_pct: int


@dataclass
class PollResult:
    mac: str
    device_name: str
    reading: Reading | None
    error: str | None = None


@dataclass
class FetchStatus:
    at: datetime | None = None
    ok: bool = False
    error: str | None = None
    temp_f: float | None = None
    humidity_pct: int | None = None


@dataclass
class LogLoadStatus:
    at: datetime | None = None
    log_path: str | None = None
    total_samples: int = 0
    last_load_samples: int = 0
    last_load_hour_bins: int = 0


class DeviceHistory:
    """In-memory 24h+ reading store per device."""

    def __init__(self) -> None:
        self._readings: dict[str, list[Reading]] = {}
        self._last_updated: dict[str, datetime] = {}
        self._fetch_status: dict[str, FetchStatus] = {}
        self._log_load_status: dict[str, LogLoadStatus] = {}

    def get_readings(self, mac: str) -> list[Reading]:
        return list(self._readings.get(mac, []))

    def last_updated(self, mac: str) -> datetime | None:
        return self._last_updated.get(mac)

    def has_data(self, mac: str) -> bool:
        return bool(self._readings.get(mac))

    def fetch_status(self, mac: str) -> FetchStatus:
        return self._fetch_status.get(mac, FetchStatus())

    def log_load_status(self, mac: str) -> LogLoadStatus:
        return self._log_load_status.get(mac, LogLoadStatus())

    def record_log_load(
        self,
        mac: str,
        *,
        added: int,
        hour_bins: int,
        log_path: str,
    ) -> None:
        previous = self._log_load_status.get(mac, LogLoadStatus())
        self._log_load_status[mac] = LogLoadStatus(
            at=datetime.now(),
            log_path=log_path,
            total_samples=previous.total_samples + added,
            last_load_samples=added,
            last_load_hour_bins=hour_bins,
        )

    def record_fetch_result(self, result: PollResult) -> None:
        mac = result.mac
        if result.reading is not None:
            self._fetch_status[mac] = FetchStatus(
                at=result.reading.timestamp,
                ok=True,
                temp_f=result.reading.temp_f,
                humidity_pct=result.reading.humidity_pct,
            )
            return
        self._fetch_status[mac] = FetchStatus(
            at=datetime.now(),
            ok=False,
            error=result.error or "Unknown fetch error",
        )

    def prune_old(self, mac: str, *, keep_hours: int = 25) -> None:
        cutoff = datetime.now() - timedelta(hours=keep_hours)
        readings = self._readings.get(mac, [])
        self._readings[mac] = [r for r in readings if r.timestamp >= cutoff]

    def add_reading(self, mac: str, reading: Reading) -> None:
        if mac not in self._readings:
            self._readings[mac] = []
        self._readings[mac].append(reading)
        self._last_updated[mac] = reading.timestamp
        self.prune_old(mac)

    def import_readings(self, mac: str, readings: list[Reading]) -> int:
        """Merge readings for a device, skipping duplicate timestamps."""
        if not readings:
            return 0
        existing = {r.timestamp for r in self.get_readings(mac)}
        added = 0
        for reading in sorted(readings, key=lambda item: item.timestamp):
            if reading.timestamp in existing:
                continue
            self.add_reading(mac, reading)
            existing.add(reading.timestamp)
            added += 1
        return added

    def clear_device(self, mac: str) -> None:
        self._readings.pop(mac, None)
        self._last_updated.pop(mac, None)
        self._fetch_status.pop(mac, None)
        self._log_load_status.pop(mac, None)

    def temp_points(self, mac: str) -> list[tuple[datetime, float]]:
        return [(r.timestamp, r.temp_f) for r in self.get_readings(mac)]

    def humidity_points(self, mac: str) -> list[tuple[datetime, float]]:
        return [(r.timestamp, float(r.humidity_pct)) for r in self.get_readings(mac)]

    def latest_updated(self, macs) -> datetime | None:
        """Most recent successful reading across the given devices."""
        times = [self.last_updated(mac) for mac in macs]
        valid = [t for t in times if t is not None]
        return max(valid) if valid else None

    def seed_fetch_status_from_readings(self, mac: str) -> None:
        """Restore last-fetch metadata from the newest in-memory reading."""
        readings = self.get_readings(mac)
        if not readings:
            return
        latest = readings[-1]
        self._fetch_status[mac] = FetchStatus(
            at=latest.timestamp,
            ok=True,
            temp_f=latest.temp_f,
            humidity_pct=latest.humidity_pct,
        )


def load_readings_from_log(history: DeviceHistory, config: AppConfig) -> int:
    """Import CSV log rows from the last 24 hours into in-memory history."""
    from tp.config import normalize_mac
    from tp.sparkline import populated_hour_bin_count

    log_path = resolved_log_path(config)
    if not log_path.is_file():
        return 0

    managed = {normalize_mac(mac) for mac in config.devices}
    if not managed:
        return 0

    cutoff = datetime.now() - timedelta(hours=LOG_HISTORY_HOURS)
    pending: dict[str, list[Reading]] = {mac: [] for mac in managed}
    loaded = 0
    log_path_text = str(log_path)

    try:
        with log_path.open("r", encoding="utf-8", newline="") as handle:
            reader = csv.DictReader(handle)
            if reader.fieldnames is None or "mac" not in reader.fieldnames:
                return 0
            for row in reader:
                mac_raw = row.get("mac", "")
                if not mac_raw:
                    continue
                mac = normalize_mac(mac_raw)
                if mac not in managed:
                    continue
                ts_raw = row.get("timestamp", "").strip()
                if not ts_raw:
                    continue
                try:
                    timestamp = datetime.strptime(ts_raw, LOG_TIMESTAMP_FORMAT)
                except ValueError:
                    continue
                if timestamp < cutoff:
                    continue
                try:
                    temp_f = float(row["temp_f"])
                    humidity_pct = int(float(row["humidity_pct"]))
                except (KeyError, TypeError, ValueError):
                    continue
                pending[mac].append(
                    Reading(
                        timestamp=timestamp,
                        temp_f=temp_f,
                        humidity_pct=humidity_pct,
                    )
                )
    except OSError:
        return loaded

    for mac, readings in pending.items():
        if not readings:
            continue
        existing = {r.timestamp for r in history.get_readings(mac)}
        new_readings = [
            reading
            for reading in sorted(readings, key=lambda item: item.timestamp)
            if reading.timestamp not in existing
        ]
        if not new_readings:
            continue
        points = [(r.timestamp, r.temp_f) for r in new_readings]
        hour_bins = populated_hour_bin_count(points)
        added = history.import_readings(mac, new_readings)
        if added:
            history.record_log_load(
                mac,
                added=added,
                hour_bins=hour_bins,
                log_path=log_path_text,
            )
            loaded += added
    for mac in managed:
        history.seed_fetch_status_from_readings(mac)
    return loaded


def append_poll_results_to_log(
    config: AppConfig, results: list[PollResult]
) -> str | None:
    """Append successful poll rows after a fetch cycle (Phase 2).

    Returns an error message on failure; readings are still committed in memory.
    """
    if not config.settings.logging_enabled:
        return None

    log_path = resolved_log_path(config)
    _, probe_error = probe_log_path(
        config.settings.log_directory,
        config.settings.log_file_name,
    )
    if probe_error:
        return probe_error

    try:
        write_header = not log_path.exists() or log_path.stat().st_size == 0
        with log_path.open("a", encoding="utf-8", newline="\n") as handle:
            writer = csv.writer(handle, lineterminator="\n")
            if write_header:
                writer.writerow(CSV_HEADER)
            for result in results:
                if result.reading is None:
                    continue
                r = result.reading
                writer.writerow(
                    [
                        r.timestamp.strftime("%Y-%m-%d %H:%M:%S"),
                        result.device_name,
                        f"{r.temp_f:.1f}",
                        r.humidity_pct,
                        result.mac,
                    ]
                )
    except OSError as exc:
        return f"Cannot write to {log_path}: {exc}"
    return None


def _history_window_cutoff(*, hours: float = LOG_HISTORY_HOURS) -> datetime:
    return datetime.now() - timedelta(hours=hours)


def _received_history_window(
    readings: list[Reading],
    *,
    hours: float = LOG_HISTORY_HOURS,
) -> tuple[list[Reading], datetime, datetime] | None:
    """Return in-window readings and the inclusive replace span [start, end]."""
    cutoff = _history_window_cutoff(hours=hours)
    in_window = [r for r in readings if r.timestamp >= cutoff]
    if not in_window:
        return None
    replace_start = min(r.timestamp for r in in_window)
    replace_end = max(r.timestamp for r in in_window)
    return in_window, replace_start, replace_end


def replace_device_memory_window(
    history: DeviceHistory,
    mac: str,
    readings: list[Reading],
    *,
    hours: float = LOG_HISTORY_HOURS,
) -> int:
    """Replace in-memory readings for mac only within the received data span."""
    window = _received_history_window(readings, hours=hours)
    if window is None:
        return 0
    in_window, replace_start, replace_end = window

    if mac not in history._readings:
        history._readings[mac] = []
    cutoff = _history_window_cutoff(hours=hours)
    kept = [
        r
        for r in history.get_readings(mac)
        if r.timestamp < cutoff
        or r.timestamp < replace_start
        or r.timestamp > replace_end
    ]
    merged = {r.timestamp: r for r in kept}
    for reading in in_window:
        merged[reading.timestamp] = reading
    history._readings[mac] = sorted(merged.values(), key=lambda item: item.timestamp)
    if history._readings[mac]:
        history._last_updated[mac] = history._readings[mac][-1].timestamp
    else:
        history._last_updated.pop(mac, None)
    history.prune_old(mac)
    history.seed_fetch_status_from_readings(mac)
    return len(in_window)


def replace_device_log_window(
    config: AppConfig,
    mac: str,
    device_name: str,
    readings: list[Reading],
    *,
    hours: float = LOG_HISTORY_HOURS,
) -> str | None:
    """Rewrite CSV log, replacing this device's rows only in the received span."""
    window = _received_history_window(readings, hours=hours)
    if window is None:
        return None
    in_window, replace_start, replace_end = window

    log_path = resolved_log_path(config)
    _, probe_error = probe_log_path(
        config.settings.log_directory,
        config.settings.log_file_name,
    )
    if probe_error:
        return probe_error

    cutoff = _history_window_cutoff(hours=hours)
    target_mac = normalize_mac(mac)
    kept_rows: list[list[str]] = []
    had_header = False

    if log_path.is_file():
        try:
            with log_path.open("r", encoding="utf-8", newline="") as handle:
                reader = csv.DictReader(handle)
                if reader.fieldnames:
                    had_header = True
                for row in reader:
                    mac_raw = row.get("mac", "")
                    if not mac_raw:
                        continue
                    row_mac = normalize_mac(mac_raw)
                    ts_raw = row.get("timestamp", "").strip()
                    try:
                        timestamp = datetime.strptime(ts_raw, LOG_TIMESTAMP_FORMAT)
                    except ValueError:
                        kept_rows.append(
                            [
                                ts_raw,
                                row.get("device", ""),
                                row.get("temp_f", ""),
                                row.get("humidity_pct", ""),
                                mac_raw,
                            ]
                        )
                        continue
                    if (
                        row_mac == target_mac
                        and replace_start <= timestamp <= replace_end
                    ):
                        continue
                    kept_rows.append(
                        [
                            timestamp.strftime(LOG_TIMESTAMP_FORMAT),
                            row.get("device", ""),
                            row.get("temp_f", ""),
                            row.get("humidity_pct", ""),
                            mac_raw,
                        ]
                    )
        except OSError as exc:
            return f"Cannot read {log_path}: {exc}"

    new_rows = [
        [
            reading.timestamp.strftime(LOG_TIMESTAMP_FORMAT),
            device_name,
            f"{reading.temp_f:.1f}",
            reading.humidity_pct,
            target_mac,
        ]
        for reading in sorted(in_window, key=lambda item: item.timestamp)
    ]
    combined = kept_rows + new_rows
    combined.sort(key=lambda row: row[0])

    try:
        log_path.parent.mkdir(parents=True, exist_ok=True)
        fd, temp_name = tempfile.mkstemp(
            prefix=f"{log_path.name}.",
            suffix=".tmp",
            dir=log_path.parent,
        )
        os.close(fd)
        temp_path = Path(temp_name)
        try:
            with temp_path.open("w", encoding="utf-8", newline="\n") as handle:
                writer = csv.writer(handle, lineterminator="\n")
                writer.writerow(CSV_HEADER)
                for row in combined:
                    writer.writerow(row)
            os.replace(temp_path, log_path)
        finally:
            if temp_path.exists():
                temp_path.unlink(missing_ok=True)
    except OSError as exc:
        return f"Cannot write to {log_path}: {exc}"

    if not had_header and not combined:
        return None
    return None


def apply_day_history(
    history: DeviceHistory,
    config: AppConfig,
    mac: str,
    device_name: str,
    readings: list[Reading],
) -> tuple[int, str | None]:
    """Merge day-history readings into memory and optionally replace log window."""
    if len(readings) < MIN_DAY_HISTORY_SAMPLES:
        return (
            0,
            f"Too few samples ({len(readings)}); need at least {MIN_DAY_HISTORY_SAMPLES}.",
        )
    imported = replace_device_memory_window(history, mac, readings)
    log_error: str | None = None
    if config.settings.logging_enabled:
        log_error = replace_device_log_window(config, mac, device_name, readings)
    return imported, log_error
