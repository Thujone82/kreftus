"""Shared Rich progress bar markup for the TUI."""

from __future__ import annotations

# Solid fill over a light-shade track (HUD-style meter).
_FILL = "█"
_TRACK = "░"


def format_progress_bar(current: int, total: int, *, width: int = 32) -> str:
    """Render a solid-fill / light-shade progress bar with a percent label.

    Empty track is ``░`` across the full remainder; filled progress uses solid
    ``█`` in the same muted foreground gray as header text (not a bright accent).
    """
    if total <= 0:
        return ""
    current = max(0, min(int(current), int(total)))
    width = max(1, int(width))
    ratio = current / total
    filled = int(round(width * ratio))
    filled = max(0, min(width, filled))
    empty = width - filled

    # No color tags on the fill so it inherits the same gray as "Fetch …" / TemPy.
    if filled >= width:
        bar = _FILL * width
    elif filled <= 0:
        bar = f"[dim]{_TRACK * width}[/]"
    else:
        bar = f"{_FILL * filled}[dim]{_TRACK * empty}[/]"

    percent = int(round(100 * ratio))
    return f"{bar} {percent}%"
