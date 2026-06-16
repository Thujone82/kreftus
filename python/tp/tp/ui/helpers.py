"""Shared UI helpers."""

from __future__ import annotations

from collections.abc import Callable

UPDATED_COL_WIDTH = len("updated HH:MM")
FETCHING_SUFFIX = "  ◀"
STATUS_COL_WIDTH = UPDATED_COL_WIDTH + len(FETCHING_SUFFIX)


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
    width: int,
    *,
    updated: str | None = None,
    stale: bool = False,
    fetching: bool = False,
) -> str:
    """Device name left; updated time right-aligned in a fixed column."""
    prefix = "[bold cyan]▶[/] " if fetching else ""
    prefix_len = len("▶ ") if fetching else 0

    updated_core = (f"updated {updated}" if updated else "updated —").rjust(UPDATED_COL_WIDTH)
    if not updated:
        updated_markup = f"[dim]{updated_core}[/]"
    elif stale:
        updated_markup = f"[yellow]{updated_core}[/]"
    else:
        updated_markup = f"[green]{updated_core}[/]"

    fetching_markup = "  [bold cyan]◀[/]" if fetching else ""
    status_plain_len = UPDATED_COL_WIDTH + (len(FETCHING_SUFFIX) if fetching else 0)
    status_left_pad = STATUS_COL_WIDTH - status_plain_len
    status_markup = (" " * status_left_pad) + updated_markup + fetching_markup

    used = prefix_len + len(name) + status_left_pad + status_plain_len
    middle_pad = max(1, width - used)
    return f"{prefix}[yellow]{name}[/]{' ' * middle_pad}{status_markup}"


def format_label_row(name: str, width: int, updated: str | None) -> str:
    """Legacy helper — prefer format_device_label_row for monitoring headers."""
    return format_device_label_row(name, width, updated=updated)
