"""TUI render helper tests."""

from __future__ import annotations

import unittest

from gol.engine import Cell
from gol.ui.tui.render import (
    HALF_CHAR,
    apply_follow_viewport,
    build_grid_markup,
    centroid_on_screen,
    corner_counter_markup,
    cursor_char_markup,
    follow_nudge_delta,
    half_cell_markup,
    logical_to_terminal,
    population_centroid,
    screen_center,
    should_recenter_follow,
    sim_delay_seconds,
    stats_bar_markup,
    terminal_grid_dims,
    viewport_dims,
    viewport_topleft,
    window_title,
    world_cell_under_cursor,
    wrap_cursor,
    FOLLOW_PAN_INTERVAL_SECONDS,
)


class TestTuiRender(unittest.TestCase):
    def test_terminal_grid_dims(self) -> None:
        self.assertEqual(terminal_grid_dims(120, 40), (120, 40))
        self.assertEqual(terminal_grid_dims(0, 0), (1, 1))
        self.assertEqual(terminal_grid_dims(80, 40, "high"), (80, 80))

    def test_viewport_dims(self) -> None:
        self.assertEqual(viewport_dims(80, 40, "low"), (80, 40, 80, 40))
        self.assertEqual(viewport_dims(80, 40, "high"), (80, 40, 80, 80))

    def test_logical_to_terminal(self) -> None:
        self.assertEqual(logical_to_terminal(5, 0), (5, 0, 0))
        self.assertEqual(logical_to_terminal(5, 1), (5, 0, 1))
        self.assertEqual(logical_to_terminal(3, 7), (3, 3, 1))

    def test_half_cell_markup_cursor_halves(self) -> None:
        plain = half_cell_markup(None, None)
        self.assertIn(HALF_CHAR, plain)
        upper_cursor = half_cell_markup(None, None, cursor_half=0)
        lower_cursor = half_cell_markup(None, None, cursor_half=1)
        self.assertNotEqual(upper_cursor, lower_cursor)
        self.assertIn("#444444", upper_cursor)
        self.assertIn("#444444", lower_cursor)

    def test_build_grid_markup_high_density(self) -> None:
        cells = {(1, 0): Cell(0, 0.0), (1, 1): Cell(0, 0.0)}
        markup = build_grid_markup(
            cells,
            cols=3,
            rows=4,
            wrapped=True,
            density="high",
            term_rows=2,
        )
        lines = markup.split("\n")
        self.assertEqual(len(lines), 2)
        self.assertIn(HALF_CHAR, lines[0])
        self.assertIn(" on ", lines[0])

    def test_build_grid_markup_high_density_edit_cursor(self) -> None:
        markup_upper = build_grid_markup(
            {},
            cols=5,
            rows=4,
            wrapped=True,
            density="high",
            term_rows=2,
            cursor=(2, 0),
        )
        markup_lower = build_grid_markup(
            {},
            cols=5,
            rows=4,
            wrapped=True,
            density="high",
            term_rows=2,
            cursor=(2, 1),
        )
        self.assertNotEqual(markup_upper, markup_lower)
        self.assertIn("#444444", markup_upper.split("\n")[0])
        self.assertIn("#444444", markup_lower.split("\n")[0])

    def test_stats_bar_markup_edit_at(self) -> None:
        markup = stats_bar_markup(1, 2, 60, edit_at=(10, 20))
        self.assertIn("@ 10,20", markup)

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
        self.assertIn("Follow:off", window_title(0, 0, running=False, infinite=True))
        self.assertIn("Follow:on", window_title(0, 0, running=False, infinite=True, auto_follow=True))
        self.assertNotIn("Pop:", window_title(10, 5, running=True, show_corner_stats=True))
        self.assertIn("Step:", window_title(10, 5, running=False, show_corner_stats=False))
        self.assertIn("Edit", window_title(0, 0, running=False, edit_mode=True))

    def test_screen_center(self) -> None:
        self.assertEqual(screen_center(10, 8), (5, 4))

    def test_wrap_cursor(self) -> None:
        self.assertEqual(wrap_cursor(-1, 0, 10, 8), (9, 0))
        self.assertEqual(wrap_cursor(10, 8, 10, 8), (0, 0))

    def test_world_cell_under_cursor(self) -> None:
        self.assertEqual(
            world_cell_under_cursor(
                wrapped=True, view_x=0, view_y=0, cursor_col=3, cursor_row=4
            ),
            (3, 4),
        )
        self.assertEqual(
            world_cell_under_cursor(
                wrapped=False, view_x=10, view_y=20, cursor_col=5, cursor_row=4
            ),
            (15, 24),
        )

    def test_cursor_char_markup(self) -> None:
        self.assertIn("+", cursor_char_markup(False))
        self.assertIn("█", cursor_char_markup(True))

    def test_build_grid_markup_with_cursor(self) -> None:
        markup = build_grid_markup(
            {}, cols=5, rows=3, wrapped=True, cursor=(2, 1)
        )
        lines = markup.split("\n")
        self.assertIn("+", lines[1])
        self.assertEqual(lines[1].count("+"), 1)

    def test_corner_counter_markup(self) -> None:
        self.assertEqual(corner_counter_markup("Pop", 42), "[bold]Pop: 42[/]")

    def test_stats_bar_markup(self) -> None:
        markup = stats_bar_markup(12, 505, 80)
        self.assertIn("[bold on black]Pop: 12[/]", markup)
        self.assertIn("[bold on black]Step: 505[/]", markup)
        plain = markup.replace("[bold on black]", "").replace("[/]", "")
        self.assertGreaterEqual(len(plain), 80)

    def test_should_recenter_follow(self) -> None:
        self.assertTrue(should_recenter_follow(10.0, None))
        self.assertFalse(should_recenter_follow(10.0, 9.8))
        self.assertTrue(should_recenter_follow(10.0, 9.5))
        self.assertEqual(FOLLOW_PAN_INTERVAL_SECONDS, 0.5)

    def test_centroid_on_screen(self) -> None:
        self.assertTrue(
            centroid_on_screen((5.0, 5.0), view_x=0, view_y=0, cols=10, rows=10)
        )
        self.assertFalse(
            centroid_on_screen((15.0, 5.0), view_x=0, view_y=0, cols=10, rows=10)
        )

    def test_follow_nudge_delta(self) -> None:
        # Centroid right of center -> pan view right (+1 view_x)
        self.assertEqual(follow_nudge_delta((6.0, 5.0), view_x=0, view_y=0, cols=10, rows=10), (1, 0))
        # Centroid above center -> pan view up (-1 view_y)
        self.assertEqual(follow_nudge_delta((5.0, 3.0), view_x=0, view_y=0, cols=10, rows=10), (0, -1))
        # Centered -> no move
        self.assertEqual(follow_nudge_delta((5.0, 5.0), view_x=0, view_y=0, cols=10, rows=10), (0, 0))

    def test_apply_follow_viewport_off_screen_snaps(self) -> None:
        vx, vy, last = apply_follow_viewport(
            (50.0, 50.0),
            view_x=0,
            view_y=0,
            cols=10,
            rows=10,
            now=1.0,
            last_at=None,
        )
        self.assertEqual((vx, vy), viewport_topleft(50.0, 50.0, 10, 10))
        self.assertEqual(last, 1.0)

    def test_apply_follow_viewport_on_screen_nudges(self) -> None:
        vx, vy, last = apply_follow_viewport(
            (6.0, 5.0),
            view_x=0,
            view_y=0,
            cols=10,
            rows=10,
            now=1.0,
            last_at=None,
        )
        self.assertEqual((vx, vy), (1, 0))
        self.assertEqual(last, 1.0)
        vx2, vy2, last2 = apply_follow_viewport(
            (6.0, 5.0),
            view_x=vx,
            view_y=vy,
            cols=10,
            rows=10,
            now=1.2,
            last_at=last,
        )
        self.assertEqual((vx2, vy2), (1, 0))
        self.assertEqual(last2, 1.0)

    def test_apply_follow_viewport_force_recenters(self) -> None:
        vx, vy, _ = apply_follow_viewport(
            (6.0, 5.0),
            view_x=0,
            view_y=0,
            cols=10,
            rows=10,
            now=1.0,
            last_at=None,
            force=True,
        )
        self.assertEqual((vx, vy), viewport_topleft(6.0, 5.0, 10, 10))


if __name__ == "__main__":
    unittest.main()
