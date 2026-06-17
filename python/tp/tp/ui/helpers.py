"""Shared UI helpers."""

from __future__ import annotations

import re
from collections.abc import Callable

_RICH_TAG = re.compile(r"\[[^\]]*\]")
COLUMN_PAD = 2


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


def format_device_label_row(
    name: str,
    *,
    stale: bool = False,
    fetching: bool = False,
) -> str:
    """Device name colored by freshness; cyan ▶/◀ when actively fetching."""
    prefix = "[bold cyan]▶[/] " if fetching else ""
    suffix = " [bold cyan]◀[/]" if fetching else ""
    color = "yellow" if stale else "green"
    return f"{prefix}[{color}]{name}[/]{suffix}"
