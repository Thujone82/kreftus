"""Built-in pattern library."""

from __future__ import annotations

import json
from importlib import resources
from pathlib import Path

WRAPPED_MIN_AXIS = 50


def wrapped_grid_layout(
    pixel_width: float, pixel_height: float
) -> tuple[int, int, float, float, float]:
    """Square cells: 50 on the shorter axis; return cols, rows, cell_size, offset_x, offset_y."""
    min_side = min(pixel_width, pixel_height)
    cell_size = min_side / WRAPPED_MIN_AXIS
    cols = max(1, int(pixel_width / cell_size))
    rows = max(1, int(pixel_height / cell_size))
    grid_w = cols * cell_size
    grid_h = rows * cell_size
    offset_x = (pixel_width - grid_w) / 2
    offset_y = (pixel_height - grid_h) / 2
    return cols, rows, cell_size, offset_x, offset_y


def load_patterns() -> dict[str, dict]:
    """Return {name: {label, cells}} from bundled patterns.json."""
    text = _read_patterns_json()
    return json.loads(text)


def _read_patterns_json() -> str:
    try:
        return resources.files("gol").joinpath("patterns.json").read_text(encoding="utf-8")
    except (FileNotFoundError, OSError, TypeError):
        pass

    import sys

    candidates: list[Path] = []
    if getattr(sys, "frozen", False) and hasattr(sys, "_MEIPASS"):
        candidates.append(Path(sys._MEIPASS) / "gol" / "patterns.json")
    candidates.append(Path(__file__).resolve().parent / "patterns.json")

    for path in candidates:
        if path.is_file():
            return path.read_text(encoding="utf-8")

    raise FileNotFoundError("patterns.json not found in package or PyInstaller bundle")


def pattern_names() -> list[tuple[str, str]]:
    """Sorted (key, label) pairs for UI."""
    patterns = load_patterns()
    return sorted(
        ((key, data["label"]) for key, data in patterns.items()),
        key=lambda item: item[1].lower(),
    )


def get_pattern_cells(name: str) -> list[tuple[int, int]]:
    patterns = load_patterns()
    if name not in patterns:
        raise KeyError(name)
    return [tuple(cell) for cell in patterns[name]["cells"]]


def center_pattern(
    cells: list[tuple[int, int]],
    *,
    wrapped: bool,
    viewport_center: tuple[float, float] | None = None,
    grid_cols: int = WRAPPED_MIN_AXIS,
    grid_rows: int = WRAPPED_MIN_AXIS,
) -> list[tuple[int, int]]:
    """Place pattern cells on the grid, centered like loadPattern() in index.html."""
    if not cells:
        return []
    xs = [x for x, _ in cells]
    ys = [y for _, y in cells]
    min_x, max_x = min(xs), max(xs)
    min_y, max_y = min(ys), max(ys)
    pat_width = max_x - min_x + 1
    pat_height = max_y - min_y + 1

    if wrapped:
        base_x = int(grid_cols / 2 - pat_width / 2)
        base_y = int(grid_rows / 2 - pat_height / 2)
    else:
        if viewport_center is None:
            viewport_center = (0.0, 0.0)
        base_x = int(viewport_center[0] - pat_width / 2)
        base_y = int(viewport_center[1] - pat_height / 2)

    return [
        (base_x + (x - min_x), base_y + (y - min_y))
        for x, y in cells
    ]
