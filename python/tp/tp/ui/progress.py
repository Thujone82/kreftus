"""Shared Rich progress bar markup for the TUI."""

from __future__ import annotations


def format_progress_bar(current: int, total: int, *, width: int = 32) -> str:
    """Render a simple ASCII progress bar with current/total counts."""
    if total <= 0:
        return ""
    current = max(0, min(current, total))
    filled = int(width * current / total)
    if filled >= width:
        bar = "=" * width
    else:
        bar = ("=" * filled) + ">" + ("." * (width - filled - 1))
    return f"[cyan]{bar}[/] [white]{current}/{total}[/]"
