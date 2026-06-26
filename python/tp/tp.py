#!/usr/bin/env python3
"""TemPy — ThermoPro TP35x Monitor — entry point."""

from __future__ import annotations

import argparse
import asyncio
import shutil
import sys
from datetime import datetime
from pathlib import Path

from rich.console import Console

# Allow running as `python tp.py` from python/tp/
_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from tp.colors import humidity_color, temp_color
from tp.config import default_ini_path, filter_devices, load_config, save_config
from tp.history import DeviceHistory, load_readings_from_log
from tp.scheduler import is_measurement_stale
from tp.sparkline import build_sparkline, colored_sparkline_markup, format_sparkline_row
from tp.ui.helpers import format_device_label_row, format_stats_row
from tp.ui.app import run_app

def _render_snapshot(config, *, device_filter: str | None = None) -> str:
    history = DeviceHistory()
    load_readings_from_log(history, config)
    width = max(60, shutil.get_terminal_size(fallback=(100, 24)).columns - 2)
    visible = filter_devices(config.devices, device_filter)
    lines: list[str] = []
    for mac, name in visible:
        updated_dt = history.last_updated(mac)
        stale = is_measurement_stale(updated_dt)
        lines.append(
            format_device_label_row(
                name,
                stale=stale,
                fetching=False,
            )
        )
        temp_points = history.temp_points(mac)
        humid_points = history.humidity_points(mac)
        temp_spark = build_sparkline(temp_points)
        humid_spark = build_sparkline(humid_points)
        readings = history.get_readings(mac)
        latest = readings[-1] if readings else None
        temp_cur = latest.temp_f if latest else None
        humid_cur = float(latest.humidity_pct) if latest else None
        temp_stats = format_stats_row(
            "Temp °F", temp_cur, temp_spark.min_value, temp_spark.max_value, "°F", color_fn=temp_color
        )
        humid_stats = format_stats_row(
            "Humid %",
            humid_cur,
            humid_spark.min_value,
            humid_spark.max_value,
            "%",
            color_fn=humidity_color,
        )
        temp_line = format_sparkline_row(colored_sparkline_markup(temp_spark, temp_color))
        humid_line = format_sparkline_row(colored_sparkline_markup(humid_spark, humidity_color))
        if stale:
            temp_stats = f"[dim]{temp_stats}[/]"
            humid_stats = f"[dim]{humid_stats}[/]"
            temp_line = f"[dim]{temp_line}[/]"
            humid_line = f"[dim]{humid_line}[/]"
        lines.append(temp_stats)
        lines.append(temp_line)
        lines.append(humid_stats)
        lines.append(humid_line)
        lines.append("")
    stamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    header = f"[bold]TemPy snapshot[/] ({stamp})"
    if device_filter:
        header += f"  [dim]filter: {device_filter}[/]"
    return "\n".join([header, "-" * len(header), *lines]).rstrip()


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="TemPy — ThermoPro TP35x Monitor")
    parser.add_argument(
        "-debug",
        "--debug",
        action="store_true",
        help="Enable session debug.log in the configured log directory",
    )
    parser.add_argument(
        "-x",
        action="store_true",
        help="Display one snapshot from saved log data and exit (no BLE polling)",
    )
    parser.add_argument(
        "-nopoll",
        "-np",
        action="store_true",
        dest="nopoll",
        help="Interactive mode without automatic poll scheduling (G still fetches manually)",
    )
    parser.add_argument(
        "-f",
        "-filter",
        dest="device_filter",
        metavar="TEXT",
        default=None,
        help="View filter: only show devices whose name contains TEXT (case-insensitive)",
    )
    parser.add_argument(
        "--history-day",
        metavar="MAC",
        default=None,
        help="Fetch up to 1 year of BLE history for MAC and exit (dev/test)",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv)
    ini_path = default_ini_path()
    config = load_config(ini_path)
    if not ini_path.exists():
        save_config(config)
    if args.history_day:
        from tp.config import normalize_mac
        from tp.debug_log import set_debug_enabled
        from tp.history import DeviceHistory
        from tp.history_fetch import DayHistoryResult, fetch_day_history_for_device

        if args.debug:
            set_debug_enabled(config, True)

        mac = normalize_mac(args.history_day)
        name = config.devices.get(mac, mac)

        async def progress(update) -> None:
            print(
                f"{update.phase}: {update.message} "
                f"(packets={update.packets}, samples={update.samples})"
            )

        async def run_fetch() -> DayHistoryResult:
            history = DeviceHistory()
            return await fetch_day_history_for_device(
                config,
                history,
                mac,
                name,
                progress,
            )

        result = asyncio.run(run_fetch())
        if result.ok:
            print(
                f"Imported {result.imported} sample(s) from {result.sample_count} received "
                f"({'memory only' if result.memory_only else 'log replaced'})."
            )
        else:
            print(f"Failed: {result.error}")
            raise SystemExit(1)
        return
    if args.x:
        if not config.devices:
            print("No managed devices configured.")
            return
        visible = filter_devices(config.devices, args.device_filter)
        if args.device_filter and not visible:
            print(f"No devices match filter {args.device_filter!r}.")
            return
        Console().print(_render_snapshot(config, device_filter=args.device_filter))
        return
    run_app(
        config,
        debug_enabled=args.debug,
        poll_enabled=not args.nopoll,
        device_filter=args.device_filter,
    )


if __name__ == "__main__":
    main()
