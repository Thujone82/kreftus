"""Unit tests for shared progress bar markup."""

from __future__ import annotations

import unittest

from tp.ui.progress import format_progress_bar


class FormatProgressBarTests(unittest.TestCase):
    def test_empty_total_returns_empty(self) -> None:
        self.assertEqual(format_progress_bar(1, 0), "")

    def test_zero_progress_is_all_shade_track(self) -> None:
        bar = format_progress_bar(0, 6, width=12)
        self.assertIn("░", bar)
        self.assertIn("0%", bar)
        self.assertNotIn("█", bar)
        self.assertNotIn("·", bar)

    def test_full_progress_is_solid_fill(self) -> None:
        bar = format_progress_bar(6, 6, width=12)
        self.assertIn("█", bar)
        self.assertIn("100%", bar)
        self.assertNotIn("░", bar)

    def test_mid_progress_has_solid_fill_and_shade_track(self) -> None:
        bar = format_progress_bar(3, 6, width=12)
        self.assertIn("█", bar)
        self.assertIn("░", bar)
        self.assertIn("50%", bar)
        self.assertNotIn("·", bar)
        self.assertNotIn("green", bar)
        self.assertNotIn("[white]", bar)

    def test_clamps_current_to_total(self) -> None:
        bar = format_progress_bar(99, 4, width=8)
        self.assertIn("100%", bar)


if __name__ == "__main__":
    unittest.main()
