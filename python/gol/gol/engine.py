"""Conway's Game of Life simulation engine."""

from __future__ import annotations

import random
from copy import deepcopy
from dataclasses import dataclass
from typing import Literal

from gol.patterns import WRAPPED_MIN_AXIS, center_pattern, get_pattern_cells

Mode = Literal["wrapped", "infinite"]

NEIGHBOR_OFFSETS = [
    (dx, dy)
    for dy in (-1, 0, 1)
    for dx in (-1, 0, 1)
    if not (dx == 0 and dy == 0)
]


@dataclass
class Cell:
    age: int
    initial_hue: float


@dataclass
class Snapshot:
    cells: dict[tuple[int, int], Cell]
    generation: int


class GameOfLife:
    def __init__(self, mode: Mode = "wrapped") -> None:
        self.mode = mode
        self.cells: dict[tuple[int, int], Cell] = {}
        self.generation = 0
        self.grid_cols = WRAPPED_MIN_AXIS
        self.grid_rows = WRAPPED_MIN_AXIS

    def set_wrapped_dimensions(self, cols: int, rows: int) -> bool:
        """Update toroidal size. Returns True if dimensions changed."""
        cols = max(1, cols)
        rows = max(1, rows)
        if cols == self.grid_cols and rows == self.grid_rows:
            return False
        self.grid_cols = cols
        self.grid_rows = rows
        return True

    @property
    def population(self) -> int:
        return len(self.cells)

    def clear(self) -> None:
        self.cells.clear()
        self.generation = 0

    def set_mode(self, mode: Mode) -> None:
        self.mode = mode
        self.clear()

    def toggle_cell(self, x: int, y: int) -> None:
        key = (x, y)
        if key in self.cells:
            del self.cells[key]
        else:
            self.cells[key] = Cell(age=0, initial_hue=random.random() * 360)

    def load_coords(
        self,
        coords: list[tuple[int, int]],
        *,
        viewport_center: tuple[float, float] | None = None,
    ) -> None:
        self.clear()
        placed = center_pattern(
            coords,
            wrapped=self.mode == "wrapped",
            viewport_center=viewport_center,
            grid_cols=self.grid_cols,
            grid_rows=self.grid_rows,
        )
        for x, y in placed:
            self.cells[(x, y)] = Cell(age=0, initial_hue=random.random() * 360)

    def load_pattern(
        self,
        name: str,
        *,
        viewport_center: tuple[float, float] | None = None,
    ) -> None:
        self.load_coords(get_pattern_cells(name), viewport_center=viewport_center)

    def snapshot(self) -> Snapshot:
        return Snapshot(cells=deepcopy(self.cells), generation=self.generation)

    def restore(self, snap: Snapshot) -> None:
        self.cells = deepcopy(snap.cells)
        self.generation = snap.generation

    def step(self) -> tuple[int, int]:
        """Advance one generation. Returns (born, died) counts."""
        old = self.cells
        if self.mode == "wrapped":
            new_cells = self._step_wrapped(old)
        else:
            new_cells = self._step_infinite(old)
        born = sum(1 for key in new_cells if key not in old)
        died = sum(1 for key in old if key not in new_cells)
        self.cells = new_cells
        self.generation += 1
        return born, died

    def scope(self) -> tuple[int, tuple[int, int] | None]:
        """Manhattan scope of farthest cell from origin (for debug logging)."""
        if not self.cells:
            return 0, None
        best = 0
        best_coord: tuple[int, int] | None = None
        for x, y in self.cells:
            value = abs(x) + abs(y)
            if value > best:
                best = value
                best_coord = (x, y)
        return best, best_coord

    def _live_neighbors(self, cells: dict[tuple[int, int], Cell], x: int, y: int) -> int:
        count = 0
        for dx, dy in NEIGHBOR_OFFSETS:
            if (x + dx, y + dy) in cells:
                count += 1
        return count

    def _step_wrapped(self, old: dict[tuple[int, int], Cell]) -> dict[tuple[int, int], Cell]:
        new_cells: dict[tuple[int, int], Cell] = {}
        for row in range(self.grid_rows):
            for col in range(self.grid_cols):
                live = 0
                for dx, dy in NEIGHBOR_OFFSETS:
                    nx = (col + dx + self.grid_cols) % self.grid_cols
                    ny = (row + dy + self.grid_rows) % self.grid_rows
                    if (nx, ny) in old:
                        live += 1
                key = (col, row)
                current = old.get(key)
                if current:
                    if live in (2, 3):
                        new_cells[key] = Cell(
                            age=current.age + 1,
                            initial_hue=current.initial_hue,
                        )
                elif live == 3:
                    new_cells[key] = Cell(age=0, initial_hue=random.random() * 360)
        return new_cells

    def _step_infinite(self, old: dict[tuple[int, int], Cell]) -> dict[tuple[int, int], Cell]:
        candidates: set[tuple[int, int]] = set()
        for x, y in old:
            for dx in (-1, 0, 1):
                for dy in (-1, 0, 1):
                    candidates.add((x + dx, y + dy))

        new_cells: dict[tuple[int, int], Cell] = {}
        for cx, cy in candidates:
            live = self._live_neighbors(old, cx, cy)
            current = old.get((cx, cy))
            if current:
                if live in (2, 3):
                    new_cells[(cx, cy)] = Cell(
                        age=current.age + 1,
                        initial_hue=current.initial_hue,
                    )
            elif live == 3:
                new_cells[(cx, cy)] = Cell(
                    age=0, initial_hue=random.random() * 360
                )
        return new_cells
