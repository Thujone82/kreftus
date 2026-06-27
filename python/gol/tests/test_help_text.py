"""CLI help text tests."""

from __future__ import annotations

import unittest

from gol.help_text import build_parser


class TestHelpText(unittest.TestCase):
    def test_main_parser_includes_tui_flag_and_section(self) -> None:
        parser = build_parser(tui_only=False)
        help_text = parser.format_help()
        self.assertIn("-tui", help_text)
        self.assertIn("Terminal UI (-tui", help_text)
        self.assertIn("pygame GUI", help_text)

    def test_tui_only_parser_excludes_gui_and_tui_flag(self) -> None:
        parser = build_parser(tui_only=True)
        help_text = parser.format_help()
        self.assertNotIn("  -tui", help_text)
        self.assertIn("gol-tui.exe", help_text)
        self.assertNotIn("pygame GUI (default)", help_text)
        self.assertIn("C              Simulation controls", help_text)


if __name__ == "__main__":
    unittest.main()
