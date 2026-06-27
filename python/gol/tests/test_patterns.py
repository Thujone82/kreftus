"""Pattern library tests."""

from __future__ import annotations

import unittest

from gol.patterns import (
    WRAPPED_MIN_AXIS,
    center_pattern,
    get_pattern_cells,
    load_patterns,
    pattern_names,
    wrapped_grid_layout,
)


class TestPatterns(unittest.TestCase):
    def test_all_patterns_have_cells(self) -> None:
        patterns = load_patterns()
        self.assertGreater(len(patterns), 50)
        for key, data in patterns.items():
            self.assertIn("label", data)
            self.assertIn("cells", data)
            self.assertGreater(len(data["cells"]), 0, msg=key)

    def test_pattern_names_sorted(self) -> None:
        names = pattern_names()
        labels = [label for _, label in names]
        self.assertEqual(labels, sorted(labels, key=str.lower))

    def test_glider_coords(self) -> None:
        cells = get_pattern_cells("glider")
        self.assertEqual(len(cells), 5)

    def test_center_wrapped_within_bounds(self) -> None:
        cells = get_pattern_cells("gosper")
        cols, rows = 100, 50
        placed = center_pattern(cells, wrapped=True, grid_cols=cols, grid_rows=rows)
        for x, y in placed:
            self.assertGreaterEqual(x, 0)
            self.assertGreaterEqual(y, 0)
            self.assertLess(x, cols)
            self.assertLess(y, rows)

    def test_wrapped_grid_layout_square_cells(self) -> None:
        cols, rows, cell_size, off_x, off_y = wrapped_grid_layout(400, 200)
        self.assertEqual(rows, WRAPPED_MIN_AXIS)
        self.assertEqual(cols, 100)
        self.assertAlmostEqual(cell_size, 4.0)
        self.assertAlmostEqual(cols * cell_size, 400.0)
        self.assertAlmostEqual(rows * cell_size, 200.0)
        self.assertAlmostEqual(off_x, 0.0)
        self.assertAlmostEqual(off_y, 0.0)

    def test_wrapped_grid_layout_letterbox(self) -> None:
        cols, rows, cell_size, off_x, off_y = wrapped_grid_layout(300, 300)
        self.assertEqual(cols, WRAPPED_MIN_AXIS)
        self.assertEqual(rows, WRAPPED_MIN_AXIS)
        self.assertAlmostEqual(off_x, 0.0)
        self.assertAlmostEqual(off_y, 0.0)

    def test_center_infinite_at_viewport(self) -> None:
        cells = get_pattern_cells("glider")
        placed = center_pattern(cells, wrapped=False, viewport_center=(10.0, 20.0))
        xs = [x for x, _ in placed]
        ys = [y for _, y in placed]
        cx = (min(xs) + max(xs)) / 2
        cy = (min(ys) + max(ys)) / 2
        self.assertAlmostEqual(cx, 10.0, delta=1.0)
        self.assertAlmostEqual(cy, 20.0, delta=1.0)


if __name__ == "__main__":
    unittest.main()
