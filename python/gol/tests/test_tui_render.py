"""TUI render helper tests."""

from __future__ import annotations

import unittest

from gol.engine import Cell
from gol.ui.tui.render import (
    build_grid_markup,
    population_centroid,
    sim_delay_seconds,
    terminal_grid_dims,
    viewport_topleft,
    window_title,
)


class TestTuiRender(unittest.TestCase):
    def test_terminal_grid_dims(self) -> None:
        self.assertEqual(terminal_grid_dims(120, 40), (120, 40))
        self.assertEqual(terminal_grid_dims(0, 0), (1, 1))

    def test_sim_delay_matches_pygame_scale(self) -> None:
        self.assertAlmostEqual(sim_delay_seconds(100), 0.032)
        self.assertAlmostEqual(sim_delay_seconds(200), 0.016)

    def test_population_centroid(self) -> None:
        cells = {
            (0, 0): Cell(0, 0.0),
            (2, 0): Cell(0, 0.0),
            (0, 2): Cell(0, 0.0),
        }
        self.assertEqual(population_centroid(cells), (2 / 3, 2 / 3))
        self.assertIsNone(population_centroid({}))

    def test_viewport_topleft(self) -> None:
        self.assertEqual(viewport_topleft(50.0, 25.0, 100, 50), (0, 0))

    def test_build_grid_markup_wrapped(self) -> None:
        cells = {(1, 0): Cell(0, 0.0)}
        markup = build_grid_markup(cells, cols=3, rows=2, wrapped=True)
        lines = markup.split("\n")
        self.assertEqual(len(lines), 2)
        self.assertTrue("█" in lines[0])
        self.assertEqual(lines[0].count("█"), 1)

    def test_build_grid_markup_infinite_viewport(self) -> None:
        cells = {(10, 5): Cell(0, 0.0)}
        markup = build_grid_markup(
            cells, cols=5, rows=3, view_x=9, view_y=4, wrapped=False
        )
        lines = markup.split("\n")
        self.assertEqual(lines[1].count("█"), 1)

    def test_window_title(self) -> None:
        self.assertIn("Pop:5", window_title(10, 5, running=True))
        self.assertIn("⏸", window_title(0, 0, running=False))
        self.assertIn("Follow:on", window_title(0, 0, running=False, infinite=True))
        self.assertIn("Follow:off", window_title(0, 0, running=False, infinite=True, auto_follow=False))


if __name__ == "__main__":
    unittest.main()
