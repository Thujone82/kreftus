"""Device fetch and log status formatting."""

from __future__ import annotations

from collections.abc import Callable

from tp.colors import humidity_color, temp_color
from tp.config import AppConfig
from tp.history import DeviceHistory, FetchStatus, LogLoadStatus
from tp.sparkline import BIN_COUNT, format_window_sparkline_rows


def _format_time(at) -> str:
    if at is None:
        return "—"
    return at.strftime("%Y-%m-%d %H:%M:%S")


def _format_fetch_section(status: FetchStatus) -> list[str]:
    lines = ["[bold]Last fetch[/]"]
    if status.at is None:
        lines.append("  [dim]Never run[/]")
        return lines
    lines.append(f"  [dim]At:[/] {_format_time(status.at)}")
    if status.ok:
        lines.append("  [green]Status: OK[/]")
        lines.append(
            f"  Temp: [white]{status.temp_f:.1f} °F[/]  "
            f"Humidity: [white]{status.humidity_pct}%[/]"
        )
    else:
        lines.append("  [red]Status: Failed[/]")
        lines.append(f"  [red]{status.error or 'Unknown error'}[/]")
    return lines


def _format_log_load_section(status: LogLoadStatus) -> list[str]:
    lines = ["[bold]Log preload[/]"]
    if status.at is None or status.total_samples == 0:
        lines.append("  [dim]No samples loaded from log[/]")
        return lines
    lines.append(f"  [dim]Last load:[/] {_format_time(status.at)}")
    if status.log_path:
        lines.append(f"  [dim]Log file:[/] {status.log_path}")
    lines.append(
        f"  Samples from log: [white]{status.total_samples}[/]"
        + (
            f"  [dim](+{status.last_load_samples} last load)[/]"
            if status.last_load_samples
            and status.last_load_samples != status.total_samples
            else ""
        )
    )
    lines.append(
        f"  72H bins from log: [white]{status.last_load_hour_bins}/{BIN_COUNT}[/]"
    )
    return lines


_STATUS_EMPTY_BIN = "[dim] [/]"


def _format_metric_sparklines(
    points: list[tuple],
    color_fn: Callable[[float], str],
    *,
    time_detail: str | None = None,
) -> list[str]:
    return format_window_sparkline_rows(
        points,
        color_fn,
        indent="  ",
        empty_bin=_STATUS_EMPTY_BIN,
        time_detail=time_detail,
    )


def format_device_status(
    config: AppConfig,
    history: DeviceHistory,
    mac: str,
    name: str,
) -> str:
    """Build Rich markup for the device status modal."""
    fetch = history.fetch_status(mac)
    log_load = history.log_load_status(mac)
    reading_count = len(history.get_readings(mac))
    temp_points = history.temp_points(mac)
    humid_points = history.humidity_points(mac)
    time_detail = getattr(config.settings, "time_detail", "less")

    lines = [
        f"[bold yellow]{name}[/]",
        f"[dim]{mac}[/]",
        "",
        *_format_log_load_section(log_load),
        "",
        *_format_fetch_section(fetch),
        "",
        "[bold]Temperature[/]",
        *_format_metric_sparklines(temp_points, temp_color, time_detail=time_detail),
        "",
        "[bold]Humidity[/]",
        *_format_metric_sparklines(humid_points, humidity_color, time_detail=time_detail),
        "",
        "[bold]Memory[/]",
        f"  Readings stored: [white]{reading_count}[/]",
        "  [dim]Sparklines use log preload + live polls[/]",
    ]
    return "\n".join(lines)
