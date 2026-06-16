"""Reading history and CSV logging."""

from __future__ import annotations

import csv
from dataclasses import dataclass
from datetime import datetime, timedelta

from tp.config import AppConfig, probe_log_path, resolved_log_path

CSV_HEADER = ["timestamp", "device", "temp_f", "humidity_pct", "mac"]
LOG_TIMESTAMP_FORMAT = "%Y-%m-%d %H:%M:%S"
LOG_HISTORY_HOURS = 24


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
