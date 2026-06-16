"""GF-style temperature and humidity color bands."""

from __future__ import annotations

# Textual/Rich color names matching ps/gf/README.md bands
TEMP_COLD_THRESHOLD = 33
TEMP_HOT_THRESHOLD = 89


def temp_color(temp_f: float) -> str:
    if temp_f < TEMP_COLD_THRESHOLD:
        return "blue"
    if temp_f > TEMP_HOT_THRESHOLD:
        return "red"
    return "white"


def humidity_color(humidity_pct: float) -> str:
    if humidity_pct < 30:
        return "cyan"
    if humidity_pct <= 60:
        return "white"
    if humidity_pct <= 70:
        return "yellow"
    return "red"
