"""CLI snapshot rendering (-x)."""

from __future__ import annotations

from datetime import datetime

from tp.colors import humidity_color, temp_color
from tp.config import AppConfig, filter_devices
from tp.history import DeviceHistory, load_readings_from_log
from tp.scheduler import is_measurement_stale
from tp.sparkline import (
    SPARKLINE_WINDOWS,
    build_sparkline,
    colored_sparkline_markup,
    format_sparkline_row,
    window_value_extremes,
)
from tp.ui.helpers import format_device_label_row, format_stats_row


def snapshot_device_lines(
    history: DeviceHistory,
    mac: str,
    name: str,
) -> list[str]:
    updated_dt = history.last_updated(mac)
    stale = is_measurement_stale(updated_dt)
    lines: list[str] = [
        format_device_label_row(
            name,
            stale=stale,
            fetching=False,
        )
    ]
    temp_points = history.temp_points(mac)
    humid_points = history.humidity_points(mac)
    readings = history.get_readings(mac)
    latest = readings[-1] if readings else None
    temp_cur = latest.temp_f if latest else None
    humid_cur = float(latest.humidity_pct) if latest else None

    for label, hours in SPARKLINE_WINDOWS:
        temp_min, temp_max = window_value_extremes(temp_points, hours=hours)
        humid_min, humid_max = window_value_extremes(humid_points, hours=hours)
        temp_spark = build_sparkline(temp_points, hours=hours)
        humid_spark = build_sparkline(humid_points, hours=hours)
        temp_stats = format_stats_row(
            "Temp °F",
            temp_cur,
            temp_min,
            temp_max,
            "°F",
            color_fn=temp_color,
        )
        humid_stats = format_stats_row(
            "Humid %",
            humid_cur,
            humid_min,
            humid_max,
            "%",
            color_fn=humidity_color,
        )
        temp_line = format_sparkline_row(
            colored_sparkline_markup(temp_spark, temp_color),
            hours_label=label,
            fixed_label=True,
        )
        humid_line = format_sparkline_row(
            colored_sparkline_markup(humid_spark, humidity_color),
            hours_label=label,
            fixed_label=True,
        )
        if stale:
            temp_stats = f"[dim]{temp_stats}[/]"
            humid_stats = f"[dim]{humid_stats}[/]"
            temp_line = f"[dim]{temp_line}[/]"
            humid_line = f"[dim]{humid_line}[/]"
        lines.extend([temp_stats, temp_line, humid_stats, humid_line])

    lines.append("")
    return lines


def render_snapshot(config: AppConfig, *, device_filter: str | None = None) -> str:
    history = DeviceHistory()
    load_readings_from_log(history, config)
    visible = filter_devices(config.devices, device_filter)
    lines: list[str] = []
    for mac, name in visible:
        lines.extend(snapshot_device_lines(history, mac, name))
    stamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    header = f"[bold]TemPy snapshot[/] ({stamp})"
    if device_filter:
        header += f"  [dim]filter: {device_filter}[/]"
    return "\n".join([header, "-" * len(header), *lines]).rstrip()
