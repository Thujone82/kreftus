"""Cell aging color helpers (ported from gol/index.html)."""

from __future__ import annotations

HUE_SHIFT_RATE = 7
START_LIGHTNESS = 70
MIN_LIGHTNESS = 40
MAX_AGE_FOR_LIGHTNESS_EFFECT = 20
SATURATION = 90


def cell_rgb(age: int, initial_hue: float) -> tuple[int, int, int]:
    """Return RGB for a live cell given age and initial hue."""
    effective_age = min(age, MAX_AGE_FOR_LIGHTNESS_EFFECT)
    hue = (initial_hue + age * HUE_SHIFT_RATE) % 360
    lightness = max(
        MIN_LIGHTNESS,
        START_LIGHTNESS - effective_age * ((START_LIGHTNESS - MIN_LIGHTNESS) / MAX_AGE_FOR_LIGHTNESS_EFFECT),
    )
    return _hsl_to_rgb(hue, SATURATION, lightness)


def _hsl_to_rgb(h: float, s: float, l: float) -> tuple[int, int, int]:
    s /= 100
    l /= 100
    c = (1 - abs(2 * l - 1)) * s
    x = c * (1 - abs((h / 60) % 2 - 1))
    m = l - c / 2
    if h < 60:
        r, g, b = c, x, 0
    elif h < 120:
        r, g, b = x, c, 0
    elif h < 180:
        r, g, b = 0, c, x
    elif h < 240:
        r, g, b = 0, x, c
    elif h < 300:
        r, g, b = x, 0, c
    else:
        r, g, b = c, 0, x
    return (
        int((r + m) * 255),
        int((g + m) * 255),
        int((b + m) * 255),
    )
