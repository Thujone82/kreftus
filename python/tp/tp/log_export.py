"""Export CSV log data to a self-contained interactive HTML report."""

from __future__ import annotations

import csv
import json
from dataclasses import dataclass
from datetime import datetime, timedelta
from importlib.resources import files
from pathlib import Path

from tp.config import AppConfig, application_dir, normalize_mac, resolved_log_path
from tp.history import (
    CSV_HEADER,
    LOG_TIMESTAMP_FORMAT,
    Reading,
    filter_false_sentinel_runs,
)

EXPORT_HTML_NAME = "tp_export.html"
DATA_PLACEHOLDER = "__EXPORT_DATA__"
CAN_EXPORT_TAIL_BYTES = 512 * 1024
CAN_EXPORT_SMALL_FILE_BYTES = 512 * 1024


@dataclass(frozen=True)
class ExportRow:
    timestamp: datetime
    device_name: str
    temp_f: float
    humidity_pct: int
    mac: str


def load_log_rows_for_export(config: AppConfig) -> list[ExportRow]:
    """Read all managed-device rows from the CSV log."""
    log_path = resolved_log_path(config)
    if not log_path.is_file():
        return []

    managed = {normalize_mac(mac): name for mac, name in config.devices.items()}
    if not managed:
        return []

    rows: list[ExportRow] = []
    try:
        with log_path.open("r", encoding="utf-8", newline="") as handle:
            reader = csv.DictReader(handle)
            if reader.fieldnames is None or "mac" not in reader.fieldnames:
                return []
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
                try:
                    temp_f = float(row["temp_f"])
                    humidity_pct = int(float(row["humidity_pct"]))
                except (KeyError, TypeError, ValueError):
                    continue
                rows.append(
                    ExportRow(
                        timestamp=timestamp,
                        device_name=row.get("device", managed[mac]).strip() or managed[mac],
                        temp_f=temp_f,
                        humidity_pct=humidity_pct,
                        mac=mac,
                    )
                )
    except OSError:
        return []
    return rows


def _row_dict_from_log_line(line: str) -> dict[str, str] | None:
    if not line.strip():
        return None
    try:
        values = next(csv.reader([line]))
    except csv.Error:
        return None
    if not values or values[0] == "timestamp":
        return None
    if len(values) < len(CSV_HEADER):
        return None
    return {name: values[index] for index, name in enumerate(CSV_HEADER)}


def _readings_from_log_row_dicts(
    row_dicts: list[dict[str, str]],
    *,
    managed: dict[str, str],
) -> list[ExportRow]:
    rows: list[ExportRow] = []
    for row in row_dicts:
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
        try:
            temp_f = float(row["temp_f"])
            humidity_pct = int(float(row["humidity_pct"]))
        except (KeyError, TypeError, ValueError):
            continue
        rows.append(
            ExportRow(
                timestamp=timestamp,
                device_name=row.get("device", managed[mac]).strip() or managed[mac],
                temp_f=temp_f,
                humidity_pct=humidity_pct,
                mac=mac,
            )
        )
    return rows


def _read_log_tail_row_dicts(log_path: Path, *, tail_bytes: int) -> list[dict[str, str]]:
    try:
        size = log_path.stat().st_size
    except OSError:
        return []
    if size <= 0:
        return []

    read_start = max(0, size - tail_bytes)
    try:
        with log_path.open("rb") as handle:
            handle.seek(read_start)
            if read_start > 0:
                handle.readline()
            chunk = handle.read()
    except OSError:
        return []

    rows: list[dict[str, str]] = []
    for line in chunk.decode("utf-8", errors="replace").splitlines():
        row_dict = _row_dict_from_log_line(line)
        if row_dict is not None:
            rows.append(row_dict)
    return rows


def _exportable_rows_from_export_rows(rows: list[ExportRow], config: AppConfig) -> bool:
    for mac in config.devices:
        normalized = normalize_mac(mac)
        readings = _readings_for_mac(rows, normalized)
        if not readings:
            continue
        filtered = filter_false_sentinel_runs(
            sorted(readings, key=lambda item: item.timestamp)
        )
        if filtered:
            return True
    return False


def _readings_for_mac(rows: list[ExportRow], mac: str) -> list[Reading]:
    return [
        Reading(
            timestamp=row.timestamp,
            temp_f=row.temp_f,
            humidity_pct=row.humidity_pct,
        )
        for row in rows
        if row.mac == mac
    ]


def build_export_payload(config: AppConfig, rows: list[ExportRow]) -> dict[str, object]:
    """Build JSON-serializable export payload grouped by device."""
    log_path = resolved_log_path(config)
    devices = [
        {"mac": mac, "name": name}
        for mac, name in config.devices.items()
    ]
    series: dict[str, list[dict[str, object]]] = {}
    for mac in config.devices:
        normalized = normalize_mac(mac)
        readings = _readings_for_mac(rows, normalized)
        if not readings:
            series[normalized] = []
            continue
        filtered = filter_false_sentinel_runs(
            sorted(readings, key=lambda item: item.timestamp)
        )
        series[normalized] = [
            {
                "t": reading.timestamp.strftime("%Y-%m-%dT%H:%M:%S"),
                "temp_f": round(reading.temp_f, 1),
                "humidity_pct": reading.humidity_pct,
            }
            for reading in filtered
        ]

    return {
        "generated_at": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        "log_path": str(log_path),
        "devices": [{"mac": normalize_mac(d["mac"]), "name": d["name"]} for d in devices],
        "series": series,
    }


def filter_series_for_hours(
    points: list[dict[str, object]],
    hours: float | None,
    *,
    end_time: datetime | None = None,
) -> list[dict[str, object]]:
    """Mirror browser timeframe filtering for tests."""
    if not points:
        return []
    if hours is None:
        return list(points)
    end = end_time or datetime.now()
    start = end - timedelta(hours=hours)
    filtered: list[dict[str, object]] = []
    for point in points:
        timestamp = datetime.fromisoformat(str(point["t"]))
        if start <= timestamp < end:
            filtered.append(point)
    return filtered


def _load_export_template() -> str:
    try:
        template_path = files("tp") / "assets" / "log_export.html"
        return template_path.read_text(encoding="utf-8")
    except (FileNotFoundError, OSError, TypeError):
        fallback = Path(__file__).resolve().parent / "assets" / "log_export.html"
        return fallback.read_text(encoding="utf-8")


def render_log_export_html(payload: dict[str, object]) -> str:
    """Inject payload JSON into the HTML template."""
    template = _load_export_template()
    data_json = json.dumps(payload, separators=(",", ":"))
    if DATA_PLACEHOLDER not in template:
        raise RuntimeError("log_export.html template is missing data placeholder")
    return template.replace(DATA_PLACEHOLDER, data_json)


def default_export_path() -> Path:
    return application_dir() / EXPORT_HTML_NAME


def can_export_log(config: AppConfig) -> bool:
    """True when ``export_log_to_html`` would produce a report."""
    if not config.devices:
        return False
    log_path = resolved_log_path(config)
    if not log_path.is_file():
        return False
    try:
        file_size = log_path.stat().st_size
    except OSError:
        return False
    if file_size < 40:
        return False

    managed = {normalize_mac(mac): name for mac, name in config.devices.items()}
    if file_size <= CAN_EXPORT_SMALL_FILE_BYTES:
        return _exportable_rows_from_export_rows(
            load_log_rows_for_export(config),
            config,
        )

    tail_rows = _readings_from_log_row_dicts(
        _read_log_tail_row_dicts(log_path, tail_bytes=CAN_EXPORT_TAIL_BYTES),
        managed=managed,
    )
    if _exportable_rows_from_export_rows(tail_rows, config):
        return True

    if tail_rows and any(row.mac in managed for row in tail_rows):
        return False

    return False


def export_log_to_html(config: AppConfig) -> tuple[Path | None, str | None]:
    """Build export HTML. Returns (path, error_message)."""
    if not config.devices:
        return None, "No devices configured — add devices first."
    rows = load_log_rows_for_export(config)
    if not rows:
        log_path = resolved_log_path(config)
        if not log_path.is_file():
            return None, f"Log file not found: {log_path}"
        return None, "Log file has no readings for managed devices."

    payload = build_export_payload(config, rows)
    if not any(payload["series"].values()):
        return None, "No exportable readings after filtering."

    html = render_log_export_html(payload)
    output_path = default_export_path()
    try:
        output_path.write_text(html, encoding="utf-8")
    except OSError as exc:
        return None, f"Cannot write {output_path}: {exc}"
    return output_path, None
