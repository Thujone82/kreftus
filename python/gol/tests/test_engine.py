"""Engine unit tests."""

from __future__ import annotations

import unittest

from gol.engine import Cell, GameOfLife


class TestEngine(unittest.TestCase):
    def test_block_stable(self) -> None:
        game = GameOfLife("infinite")
        game.cells = {
            (0, 0): Cell(0, 0.0),
            (1, 0): Cell(0, 0.0),
            (0, 1): Cell(0, 0.0),
            (1, 1): Cell(0, 0.0),
        }
        pop = game.population
        game.step()
        self.assertEqual(game.population, pop)
        self.assertEqual(set(game.cells.keys()), {(0, 0), (1, 0), (0, 1), (1, 1)})

    def test_blinker_period_two(self) -> None:
        game = GameOfLife("infinite")
        game.cells = {
            (1, 0): Cell(0, 0.0),
            (1, 1): Cell(0, 0.0),
            (1, 2): Cell(0, 0.0),
        }
        game.step()
        self.assertEqual(set(game.cells), {(0, 1), (1, 1), (2, 1)})
        game.step()
        self.assertEqual(set(game.cells), {(1, 0), (1, 1), (1, 2)})

    def test_glider_moves_in_infinite_mode(self) -> None:
        game = GameOfLife("infinite")
        game.load_coords([(1, 0), (2, 1), (0, 2), (1, 2), (2, 2)])
        start = set(game.cells)
        for _ in range(4):
            game.step()
        self.assertNotEqual(set(game.cells), start)
        self.assertEqual(game.population, 5)

    def test_empty_grid_stable(self) -> None:
        game = GameOfLife("wrapped")
        game.step()
        self.assertEqual(game.population, 0)
        self.assertEqual(game.generation, 1)

    def test_wrapped_toroidal_wrap(self) -> None:
        game = GameOfLife("wrapped")
        # Place blinker on top row; in wrapped mode vertical blinker at y=0 wraps.
        game.load_coords([(0, 0), (0, 1), (0, 2)])
        before = game.population
        game.step()
        self.assertEqual(game.population, before)

    def test_set_wrapped_dimensions(self) -> None:
        game = GameOfLife("wrapped")
        self.assertFalse(game.set_wrapped_dimensions(50, 50))
        self.assertTrue(game.set_wrapped_dimensions(100, 50))
        self.assertEqual(game.grid_cols, 100)
        self.assertEqual(game.grid_rows, 50)

    def test_snapshot_restore(self) -> None:
        game = GameOfLife("infinite")
        game.cells = {
            (0, 0): Cell(0, 0.0),
            (1, 0): Cell(0, 0.0),
            (2, 0): Cell(0, 0.0),
        }
        snap = game.snapshot()
        gen_before = game.generation
        game.step()
        self.assertEqual(game.generation, gen_before + 1)
        game.restore(snap)
        self.assertEqual(game.generation, snap.generation)
        self.assertEqual(len(game.cells), len(snap.cells))


if __name__ == "__main__":
    unittest.main()
