"""Pure helpers for TUI grid rendering."""

from __future__ import annotations

import sys
from functools import lru_cache
from typing import Literal

from gol.colors import cell_rgb
from gol.engine import Cell

BASE_DELAY_MS = 16
LIVE_CHAR = "█"
HALF_CHAR = "▄"
EMPTY_COLOR = "#121212"
CURSOR_LIVE_COLOR = "#cc0000"
CURSOR_DEAD_COLOR = "#444444"
_empty_line_cache: dict[int, str] = {}
WINDOW_TITLE = "GoLPy"
FOLLOW_PAN_INTERVAL_SECONDS = 0.5  # max 2 follow viewport updates per second

Density = Literal["low", "high"]


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


def viewport_dims(
    term_w: int, term_h: int, density: Density = "low"
) -> tuple[int, int, int, int]:
    """Return term_cols, term_rows, grid_cols, grid_rows (logical cells)."""
    term_cols = max(1, term_w)
    term_rows = max(1, term_h)
    if density == "high":
        return term_cols, term_rows, term_cols, term_rows * 2
    return term_cols, term_rows, term_cols, term_rows


def terminal_grid_dims(
    width: int, height: int, density: Density = "low"
) -> tuple[int, int]:
    """Logical grid columns and rows for the terminal size."""
    _, _, grid_cols, grid_rows = viewport_dims(width, height, density)
    return grid_cols, grid_rows


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


def screen_center(cols: int, rows: int) -> tuple[int, int]:
    """Screen cell at the viewport center."""
    return cols // 2, rows // 2


def wrap_cursor(col: int, row: int, cols: int, rows: int) -> tuple[int, int]:
    """Move cursor on a toroidal grid with modulo wrap."""
    return col % cols, row % rows


def world_cell_under_cursor(
    *,
    wrapped: bool,
    view_x: int,
    view_y: int,
    cursor_col: int,
    cursor_row: int,
) -> tuple[int, int]:
    """Grid coordinates of the cell under the edit cursor."""
    if wrapped:
        return cursor_col, cursor_row
    return view_x + cursor_col, view_y + cursor_row


def logical_to_terminal(logical_col: int, logical_row: int) -> tuple[int, int, int]:
    """Map a logical cell to terminal column, terminal row, and half (0=upper, 1=lower)."""
    return logical_col, logical_row // 2, logical_row % 2


def _empty_line(cols: int) -> str:
    line = _empty_line_cache.get(cols)
    if line is None:
        line = f"[{EMPTY_COLOR} on {EMPTY_COLOR}]{HALF_CHAR * cols}[/]"
        _empty_line_cache[cols] = line
    return line


def _occupied_logical_rows(
    cells: dict[tuple[int, int], Cell],
    *,
    cols: int,
    rows: int,
    view_x: int,
    view_y: int,
    wrapped: bool,
) -> set[int]:
    """Logical row indices that contain at least one live cell in the viewport."""
    if wrapped:
        return {
            row
            for col, row in cells
            if 0 <= col < cols and 0 <= row < rows
        }
    occupied: set[int] = set()
    x_max = view_x + cols
    y_max = view_y + rows
    for x, y in cells:
        if view_x <= x < x_max and view_y <= y < y_max:
            occupied.add(y - view_y)
    return occupied


@lru_cache(maxsize=4096)
def _cached_cell_hex(age: int, hue_key: int) -> str:
    r, g, b = cell_rgb(age, hue_key / 1000.0)
    return f"#{r:02x}{g:02x}{b:02x}"


def _cell_hex(cell: Cell | None) -> str:
    if cell is None:
        return EMPTY_COLOR
    return _cached_cell_hex(cell.age, int(cell.initial_hue * 1000))


def _cursor_half_hex(is_live: bool) -> str:
    return CURSOR_LIVE_COLOR if is_live else CURSOR_DEAD_COLOR


def half_cell_markup(
    upper: Cell | None,
    lower: Cell | None,
    *,
    cursor_half: int | None = None,
) -> str:
    """Rich markup for one high-density terminal cell (▄: bg=upper, fg=lower)."""
    upper_hex = _cell_hex(upper)
    lower_hex = _cell_hex(lower)
    if cursor_half == 0:
        active = upper
        return f"[{lower_hex} on {_cursor_half_hex(active is not None)}]{HALF_CHAR}[/]"
    if cursor_half == 1:
        active = lower
        return f"[{_cursor_half_hex(active is not None)} on {upper_hex}]{HALF_CHAR}[/]"
    return f"[{lower_hex} on {upper_hex}]{HALF_CHAR}[/]"


def cursor_char_markup(is_live: bool) -> str:
    """High-contrast edit cursor visible on empty or live cells."""
    if is_live:
        return f"[bold white on #cc0000]{LIVE_CHAR}[/]"
    return "[bold white on #444444]+[/]"


def build_grid_markup(
    cells: dict[tuple[int, int], Cell],
    *,
    cols: int,
    rows: int,
    view_x: int = 0,
    view_y: int = 0,
    wrapped: bool,
    density: Density = "low",
    term_rows: int | None = None,
    cursor: tuple[int, int] | None = None,
) -> str:
    """Build a Rich markup string for the simulation grid."""
    if density == "high":
        return _build_high_density_markup(
            cells,
            cols=cols,
            rows=rows,
            view_x=view_x,
            view_y=view_y,
            wrapped=wrapped,
            term_rows=term_rows or max(1, rows // 2),
            cursor=cursor,
        )

    lines: list[str] = []
    for row in range(rows):
        parts: list[str] = []
        for col in range(cols):
            if cursor == (col, row):
                if wrapped:
                    gx, gy = col, row
                else:
                    gx = view_x + col
                    gy = view_y + row
                parts.append(cursor_char_markup((gx, gy) in cells))
                continue
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


def _build_high_density_markup(
    cells: dict[tuple[int, int], Cell],
    *,
    cols: int,
    rows: int,
    view_x: int,
    view_y: int,
    wrapped: bool,
    term_rows: int,
    cursor: tuple[int, int] | None,
) -> str:
    cursor_term: tuple[int, int, int] | None = None
    if cursor is not None:
        cursor_term = logical_to_terminal(cursor[0], cursor[1])
    cursor_term_row = cursor_term[1] if cursor_term is not None else None
    occupied_rows = _occupied_logical_rows(
        cells,
        cols=cols,
        rows=rows,
        view_x=view_x,
        view_y=view_y,
        wrapped=wrapped,
    )

    lines: list[str] = []
    for term_row in range(term_rows):
        upper_logical = term_row * 2
        lower_logical = term_row * 2 + 1
        row_has_cells = (
            upper_logical in occupied_rows
            or (lower_logical < rows and lower_logical in occupied_rows)
        )
        if term_row != cursor_term_row and not row_has_cells:
            lines.append(_empty_line(cols))
            continue

        parts: list[str] = []
        cursor_col = (
            cursor_term[0]
            if cursor_term is not None and term_row == cursor_term_row
            else None
        )
        cursor_half = cursor_term[2] if cursor_col is not None else None
        for col in range(cols):
            if wrapped:
                upper = cells.get((col, upper_logical))
                lower = (
                    cells.get((col, lower_logical)) if lower_logical < rows else None
                )
            else:
                upper = cells.get((view_x + col, view_y + upper_logical))
                lower = (
                    cells.get((view_x + col, view_y + lower_logical))
                    if lower_logical < rows
                    else None
                )
            if cursor_col == col and cursor_half is not None:
                parts.append(
                    _half_cell_markup_fast(upper, lower, cursor_half=cursor_half)
                )
            elif upper is None and lower is None:
                parts.append(" ")
            else:
                upper_hex = _cell_hex(upper)
                lower_hex = _cell_hex(lower)
                parts.append(f"[{lower_hex} on {upper_hex}]{HALF_CHAR}[/]")
        lines.append("".join(parts))
    return "\n".join(lines)


def _half_cell_markup_fast(
    upper: Cell | None,
    lower: Cell | None,
    *,
    cursor_half: int,
) -> str:
    upper_hex = _cell_hex(upper)
    lower_hex = _cell_hex(lower)
    if cursor_half == 0:
        active = upper
        return f"[{lower_hex} on {_cursor_half_hex(active is not None)}]{HALF_CHAR}[/]"
    active = lower
    return f"[{_cursor_half_hex(active is not None)} on {upper_hex}]{HALF_CHAR}[/]"


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


def hud_pop_markup(population: int) -> str:
    """Rich markup for the population HUD label."""
    return f"[bold on black]Pop: {population}[/]"


def hud_step_markup(
    generation: int,
    *,
    edit_at: tuple[int, int] | None = None,
) -> str:
    """Rich markup for the generation HUD label."""
    text = f"Step: {generation}"
    if edit_at is not None:
        text += f"  @ {edit_at[0]},{edit_at[1]}"
    return f"[bold on black]{text}[/]"


def stats_bar_markup(
    population: int,
    generation: int,
    width: int,
    *,
    edit_at: tuple[int, int] | None = None,
) -> str:
    """Legacy combined markup (tests only); simulation uses corner HUD widgets."""
    left = hud_pop_markup(population).removeprefix("[bold on black]").removesuffix("[/]")
    right = hud_step_markup(generation, edit_at=edit_at).removeprefix("[bold on black]").removesuffix("[/]")
    pad = max(1, width - len(left) - len(right))
    return f"[bold on black]{left}[/]{' ' * pad}[bold on black]{right}[/]"


def window_title(
    generation: int,
    population: int,
    *,
    running: bool,
    infinite: bool = False,
    auto_follow: bool = False,
    show_corner_stats: bool = False,
    edit_mode: bool = False,
) -> str:
    state = "▶" if running else "⏸"
    title = f"GoLPy {state}"
    if edit_mode:
        title += "  Edit"
    if not show_corner_stats:
        title += f"  Pop:{population}  Step:{generation}"
    if infinite:
        follow = "on" if auto_follow else "off"
        title += f"  Follow:{follow}"
    return title
