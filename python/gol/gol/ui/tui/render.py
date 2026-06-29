"""Pure helpers for TUI grid rendering."""

from __future__ import annotations

import sys

from gol.colors import cell_rgb
from gol.engine import Cell

BASE_DELAY_MS = 16
LIVE_CHAR = "█"
WINDOW_TITLE = "GoLPy"
FOLLOW_PAN_INTERVAL_SECONDS = 0.5  # max 2 follow viewport updates per second


def set_terminal_window_title(title: str = WINDOW_TITLE) -> None:
    """Set the host terminal tab/window title."""
    if sys.platform == "win32":
        try:
            import ctypes

            ctypes.windll.kernel32.SetConsoleTitleW(title)
        except (AttributeError, OSError):
            pass
    try:
        from rich.console import Console

        Console().set_window_title(title)
    except Exception:  # noqa: BLE001
        pass


def terminal_grid_dims(width: int, height: int) -> tuple[int, int]:
    """One terminal character per cell."""
    return max(1, width), max(1, height)


def sim_delay_seconds(speed: int) -> float:
    """Match pygame app delay: BASE_DELAY_MS * (200 / speed)."""
    clamped = max(10, min(200, speed))
    return BASE_DELAY_MS * (200 / clamped) / 1000.0


def population_centroid(cells: dict[tuple[int, int], Cell]) -> tuple[float, float] | None:
    if not cells:
        return None
    xs = [x for x, _ in cells]
    ys = [y for _, y in cells]
    return sum(xs) / len(xs), sum(ys) / len(ys)


def viewport_topleft(center_x: float, center_y: float, cols: int, rows: int) -> tuple[int, int]:
    """Top-left cell so (center_x, center_y) sits near the viewport center."""
    return int(center_x - cols / 2), int(center_y - rows / 2)


def pattern_bounds(cells: dict[tuple[int, int], Cell]) -> tuple[float, float] | None:
    if not cells:
        return None
    xs = [x for x, _ in cells]
    ys = [y for _, y in cells]
    return (min(xs) + max(xs)) / 2, (min(ys) + max(ys)) / 2


def _rgb_markup(r: int, g: int, b: int, char: str) -> str:
    return f"[#{r:02x}{g:02x}{b:02x}]{char}[/]"


def build_grid_markup(
    cells: dict[tuple[int, int], Cell],
    *,
    cols: int,
    rows: int,
    view_x: int = 0,
    view_y: int = 0,
    wrapped: bool,
) -> str:
    """Build a Rich markup string: one char per grid cell, rows separated by newlines."""
    lines: list[str] = []
    for row in range(rows):
        parts: list[str] = []
        for col in range(cols):
            if wrapped:
                gx, gy = col, row
            else:
                gx = view_x + col
                gy = view_y + row
            cell = cells.get((gx, gy))
            if cell:
                r, g, b = cell_rgb(cell.age, cell.initial_hue)
                parts.append(_rgb_markup(r, g, b, LIVE_CHAR))
            else:
                parts.append(" ")
        lines.append("".join(parts))
    return "\n".join(lines)


def centroid_on_screen(
    centroid: tuple[float, float],
    *,
    view_x: int,
    view_y: int,
    cols: int,
    rows: int,
) -> bool:
    """True when the population centroid lies inside the current viewport."""
    cx, cy = centroid
    return view_x <= cx < view_x + cols and view_y <= cy < view_y + rows


def follow_nudge_delta(
    centroid: tuple[float, float],
    *,
    view_x: int,
    view_y: int,
    cols: int,
    rows: int,
) -> tuple[int, int]:
    """One orthogonal viewport step toward centering the centroid."""
    cx, cy = centroid
    ex = cx - (view_x + cols / 2)
    ey = cy - (view_y + rows / 2)
    if abs(ex) < 0.5 and abs(ey) < 0.5:
        return 0, 0
    if abs(ex) >= abs(ey):
        if abs(ex) >= 0.5:
            return (1 if ex > 0 else -1), 0
        return 0, (1 if ey > 0 else -1)
    if abs(ey) >= 0.5:
        return 0, (1 if ey > 0 else -1)
    return (1 if ex > 0 else -1), 0


def apply_follow_viewport(
    centroid: tuple[float, float],
    *,
    view_x: int,
    view_y: int,
    cols: int,
    rows: int,
    now: float,
    last_at: float | None,
    force: bool = False,
    interval: float = FOLLOW_PAN_INTERVAL_SECONDS,
) -> tuple[int, int, float | None]:
    """Track centroid: snap if off-screen, else ≤1 orthogonal cell per interval."""
    if force:
        vx, vy = viewport_topleft(centroid[0], centroid[1], cols, rows)
        return vx, vy, now

    if not centroid_on_screen(
        centroid, view_x=view_x, view_y=view_y, cols=cols, rows=rows
    ):
        vx, vy = viewport_topleft(centroid[0], centroid[1], cols, rows)
        return vx, vy, now

    if not should_recenter_follow(now, last_at, interval=interval):
        return view_x, view_y, last_at

    dx, dy = follow_nudge_delta(
        centroid, view_x=view_x, view_y=view_y, cols=cols, rows=rows
    )
    if dx == 0 and dy == 0:
        return view_x, view_y, now
    return view_x + dx, view_y + dy, now


def should_recenter_follow(
    now: float,
    last_at: float | None,
    *,
    interval: float = FOLLOW_PAN_INTERVAL_SECONDS,
) -> bool:
    """True when auto-follow may move the viewport (immediate if never centered)."""
    if last_at is None:
        return True
    return now - last_at >= interval


def corner_counter_markup(label: str, value: int) -> str:
    """Rich markup for a single corner stat label."""
    return f"[bold]{label}: {value}[/]"


def stats_bar_markup(population: int, generation: int, width: int) -> str:
    """One-line overlay with Pop left and Step right; only text blocks the grid."""
    left = f"Pop: {population}"
    right = f"Step: {generation}"
    pad = max(1, width - len(left) - len(right))
    styled = "[bold on black]"
    return f"{styled}{left}[/]{' ' * pad}{styled}{right}[/]"


def window_title(
    generation: int,
    population: int,
    *,
    running: bool,
    infinite: bool = False,
    auto_follow: bool = False,
    show_corner_stats: bool = False,
) -> str:
    state = "▶" if running else "⏸"
    title = f"GoLPy {state}"
    if not show_corner_stats:
        title += f"  Pop:{population}  Step:{generation}"
    if infinite:
        follow = "on" if auto_follow else "off"
        title += f"  Follow:{follow}"
    return title
