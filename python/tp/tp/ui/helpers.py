"""Shared UI helpers."""

from __future__ import annotations

import re
from collections.abc import Callable

from tp.ble import NOW_READ_CONNECTING, NOW_READ_PASSIVE, NOW_READ_SYNC

_RICH_TAG = re.compile(r"\[[^\]]*\]")
COLUMN_PAD = 2
INFO_HOTKEYS = "1234567890"
MAX_INFO_HOTKEYS = 10


def info_hotkey_index(key: str) -> int | None:
    """Map footer digit keys 1–9,0 to zero-based device index (0–9)."""
    if len(key) != 1 or key not in INFO_HOTKEYS:
        return None
    return 9 if key == "0" else int(key) - 1


def info_hotkey_footer_label(device_count: int) -> str:
    """Rich markup for the monitoring footer info hint (1 info / 1-5 info / 1-0 info)."""
    count = min(device_count, MAX_INFO_HOTKEYS)
    if count <= 0:
        return ""
    if count == 1:
        return "[dim]1[/] info"
    high = "0" if count == 10 else str(count)
    return f"[dim]1-{high}[/] info"


def plain_markup_len(markup: str) -> int:
    return len(_RICH_TAG.sub("", markup))


def measure_block_width(lines: list[str]) -> int:
    if not lines:
        return 0
    return max(plain_markup_len(line) for line in lines)


def measure_blocks_column_width(blocks: list[list[str]], *, minimum: int = 40) -> int:
    if not blocks:
        return minimum
    return max(minimum, max(measure_block_width(block) for block in blocks))


def max_columns_for_width(
    area_width: int,
    column_width: int,
    *,
    pad: int = COLUMN_PAD,
) -> int:
    slot = column_width + 2 * pad
    if slot <= 0:
        return 1
    return max(1, area_width // slot)


def layout_device_blocks(
    blocks: list[list[str]],
    *,
    column_width: int,
    columns: int,
    pad: int = COLUMN_PAD,
) -> str:
    """Lay out device blocks row-major: 1 2 / 3 4 / …"""
    if not blocks:
        return ""
    if columns <= 1:
        return "\n\n".join("\n".join(block) for block in blocks)

    line_count = len(blocks[0])
    padded: list[list[str]] = []
    for block in blocks:
        padded.append(
            [
                line + " " * max(0, column_width - plain_markup_len(line))
                for line in block
            ]
        )

    rows = (len(padded) + columns - 1) // columns
    output: list[str] = []
    for row in range(rows):
        for line_idx in range(line_count):
            parts: list[str] = []
            for col in range(columns):
                idx = row * columns + col
                line = padded[idx][line_idx] if idx < len(padded) else ""
                parts.append((" " * pad) + line.ljust(column_width) + (" " * pad))
            output.append("".join(parts))
        if row < rows - 1:
            output.append("")
    return "\n".join(output).rstrip()


def _format_stat_field(
    label: str,
    value: float,
    unit: str,
    color_fn: Callable[[float], str] | None,
) -> str:
    if unit == "°F":
        text = f"{label} {value:.1f}"
    else:
        text = f"{label} {int(value)}"
    if color_fn is None:
        return text
    return f"[{color_fn(value)}]{text}[/]"


def format_stats_row(
    label: str,
    cur: float | None,
    min_v: float | None,
    max_v: float | None,
    unit: str,
    *,
    color_fn: Callable[[float], str] | None = None,
) -> str:
    if cur is None:
        values = "cur —   min —   max —"
    elif min_v is not None and max_v is not None:
        values = "   ".join(
            (
                _format_stat_field("cur", cur, unit, color_fn),
                _format_stat_field("min", min_v, unit, color_fn),
                _format_stat_field("max", max_v, unit, color_fn),
            )
        )
    else:
        values = _format_stat_field("cur", cur, unit, color_fn)
    return f"{label:<10}{values}"


def _fetch_arrow_color(fetch_step: str | None) -> str:
    if fetch_step == NOW_READ_SYNC:
        return "green"
    if fetch_step == NOW_READ_PASSIVE:
        return "yellow"
    return "cyan"


def format_device_label_row(
    name: str,
    *,
    stale: bool = False,
    fetching: bool = False,
    fetch_step: str | None = None,
) -> str:
    """Device name colored by freshness; fetch arrows show BLE step (connect/sync/passive)."""
    if fetching:
        arrow_color = _fetch_arrow_color(fetch_step or NOW_READ_CONNECTING)
        prefix = f"[bold {arrow_color}]▶[/] "
        suffix = f" [bold {arrow_color}]◀[/]"
    else:
        prefix = ""
        suffix = ""
    color = "yellow" if stale else "green"
    return f"{prefix}[{color}]{name}[/]{suffix}"
