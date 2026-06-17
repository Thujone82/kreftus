"""Shared UI helpers."""

from __future__ import annotations

from collections.abc import Callable


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
