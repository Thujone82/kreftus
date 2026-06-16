"""Indoor temperature and humidity color bands."""

from __future__ import annotations


def temp_color(temp_f: float) -> str:
    """Indoor °F bands for current values and sparkline glyphs."""
    if temp_f < 55:
        return "cyan"
    if temp_f < 65:
        return "green"
    if temp_f < 72:
        return "white"
    if temp_f < 78:
        return "yellow"
    if temp_f < 82:
        return "red"
    return "magenta"


def humidity_color(humidity_pct: float) -> str:
    if humidity_pct < 30:
        return "cyan"
    if humidity_pct <= 60:
        return "white"
    if humidity_pct <= 70:
        return "yellow"
    return "red"
