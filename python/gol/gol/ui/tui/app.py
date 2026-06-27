"""Textual application root for GoLPy TUI."""

from __future__ import annotations

from textual.app import App
from textual.binding import Binding
from textual.screen import ModalScreen

from gol.engine import Mode
from gol.ui.tui.render import WINDOW_TITLE, set_terminal_window_title
from gol.ui.tui.setup import SetupScreen
from gol.ui.tui.sim import SimulationScreen


class GolTuiApp(App):
    """GoLPy terminal UI."""

    TITLE = "GoLPy"
    BINDINGS = [
        Binding("q", "quit_or_back", "Quit", priority=True, show=False),
    ]

    CSS = """
    Screen {
        background: #121212;
    }
    """

    def __init__(
        self,
        *,
        initial_mode: Mode = "wrapped",
        initial_pattern: str | None = None,
        initial_speed: int = 100,
        debug: bool = False,
    ) -> None:
        super().__init__()
        self.initial_mode = initial_mode
        self.initial_pattern = initial_pattern
        self.initial_speed = initial_speed
        self.debug_enabled = debug

    def on_mount(self) -> None:
        set_terminal_window_title()
        self.console.set_window_title(WINDOW_TITLE)
        self.push_screen(
            SetupScreen(
                initial_mode=self.initial_mode,
                initial_pattern=self.initial_pattern,
                initial_speed=self.initial_speed,
            )
        )

    def push_simulation(self, *, mode: Mode, pattern_key: str, speed: int) -> None:
        self.push_screen(
            SimulationScreen(mode=mode, pattern_key=pattern_key, speed=speed)
        )

    def action_quit_or_back(self) -> None:
        """Q: close modal, leave simulation for setup, or exit from setup."""
        screen = self.screen
        if isinstance(screen, ModalScreen):
            if hasattr(screen, "action_dismiss"):
                screen.action_dismiss()
            else:
                screen.dismiss(None)
            return
        if isinstance(screen, SetupScreen):
            self.exit()
            return
        self.pop_screen()


def run_tui(
    *,
    mode: Mode = "wrapped",
    pattern: str | None = None,
    speed: int = 100,
    debug: bool = False,
) -> None:
    set_terminal_window_title()
    app = GolTuiApp(
        initial_mode=mode,
        initial_pattern=pattern,
        initial_speed=speed,
        debug=debug,
    )
    app.run()
