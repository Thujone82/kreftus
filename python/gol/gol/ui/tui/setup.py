"""TUI setup screen — mode, pattern, speed."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.binding import Binding
from textual.containers import Vertical, VerticalScroll
from textual.screen import Screen
from textual.widgets import OptionList, Static
from textual.widgets.option_list import Option

from gol.engine import Mode
from gol.patterns import pattern_names
from gol.ui.tui.controls_modal import ControlsModal
from gol.ui.tui.render import WINDOW_TITLE, Density, set_terminal_window_title


class SetupScreen(Screen):
    """Pick mode, pattern, and speed before simulation."""

    BINDINGS = [
        Binding("w", "set_wrapped", "Wrapped", show=False),
        Binding("i", "set_infinite", "Infinite", show=False),
        Binding("s", "start", "Start", show=False),
        Binding("up", "pattern_up", "Up", show=False),
        Binding("down", "pattern_down", "Down", show=False),
        Binding("left", "speed_down", "Slower", show=False),
        Binding("right", "speed_up", "Faster", show=False),
        Binding("d", "toggle_density", "Density", show=False),
        Binding("c", "show_controls", "Controls", show=False),
    ]

    DEFAULT_CSS = """
    SetupScreen {
        align: center middle;
    }
    #setup-panel {
        width: 72;
        height: auto;
        max-height: 100%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    #pattern-list {
        height: 14;
        margin: 1 0;
    }
    #setup-hint {
        color: $text-muted;
    }
    """

    def __init__(
        self,
        *,
        initial_mode: Mode = "wrapped",
        initial_pattern: str | None = None,
        initial_speed: int = 100,
        initial_density: Density = "low",
    ) -> None:
        super().__init__()
        self.patterns = pattern_names()
        self.mode: Mode = initial_mode
        self.speed = max(10, min(200, initial_speed))
        self.density: Density = initial_density
        self._pattern_index = 0
        if initial_pattern:
            for idx, (key, _) in enumerate(self.patterns):
                if key == initial_pattern:
                    self._pattern_index = idx
                    break

    @property
    def selected_pattern_key(self) -> str:
        return self.patterns[self._pattern_index][0]

    @property
    def selected_pattern_label(self) -> str:
        return self.patterns[self._pattern_index][1]

    def compose(self) -> ComposeResult:
        with Vertical(id="setup-panel"):
            yield Static("", id="setup-header")
            yield OptionList(id="pattern-list")
            yield Static("", id="setup-status")
            yield Static(
                "[dim]W/I mode · ↑↓ pattern · ←→ speed · D density · Enter/S start · C controls · Q quit[/]",
                id="setup-hint",
            )

    def on_mount(self) -> None:
        set_terminal_window_title()
        self.app.console.set_window_title(WINDOW_TITLE)
        options = [Option(label, id=key) for key, label in self.patterns]
        option_list = self.query_one("#pattern-list", OptionList)
        option_list.add_options(options)
        if self.patterns:
            option_list.highlighted = self._pattern_index
        self._refresh()

    def on_screen_resume(self) -> None:
        set_terminal_window_title()
        self.app.console.set_window_title(WINDOW_TITLE)

    def _refresh(self) -> None:
        mode_label = "Wrapped" if self.mode == "wrapped" else "Infinite"
        density_label = "High" if self.density == "high" else "Low"
        self.query_one("#setup-header", Static).update(
            f"[bold]GoLPy — Setup[/]\n\n"
            f"Mode: [cyan]{mode_label}[/]  ·  Speed: [cyan]{self.speed}[/]  ·  "
            f"Density: [cyan]{density_label}[/]"
        )
        self.query_one("#setup-status", Static).update(
            f"Pattern: [yellow]{self.selected_pattern_label}[/] [dim]({self.selected_pattern_key})[/]"
        )
        option_list = self.query_one("#pattern-list", OptionList)
        if option_list.highlighted != self._pattern_index:
            option_list.highlighted = self._pattern_index

    def _sync_pattern_index(self) -> None:
        option_list = self.query_one("#pattern-list", OptionList)
        if option_list.highlighted is not None:
            self._pattern_index = option_list.highlighted
        self._refresh()

    def on_option_list_option_highlighted(self, event: OptionList.OptionHighlighted) -> None:
        if event.option_index is not None:
            self._pattern_index = event.option_index
            self._refresh()

    def action_set_wrapped(self) -> None:
        self.mode = "wrapped"
        self._refresh()

    def action_set_infinite(self) -> None:
        self.mode = "infinite"
        self._refresh()

    def action_pattern_up(self) -> None:
        if not self.patterns:
            return
        self._pattern_index = (self._pattern_index - 1) % len(self.patterns)
        self.query_one("#pattern-list", OptionList).highlighted = self._pattern_index
        self._refresh()

    def action_pattern_down(self) -> None:
        if not self.patterns:
            return
        self._pattern_index = (self._pattern_index + 1) % len(self.patterns)
        self.query_one("#pattern-list", OptionList).highlighted = self._pattern_index
        self._refresh()

    def action_speed_up(self) -> None:
        self.speed = min(200, self.speed + 10)
        self._refresh()

    def action_speed_down(self) -> None:
        self.speed = max(10, self.speed - 10)
        self._refresh()

    def action_toggle_density(self) -> None:
        self.density = "high" if self.density == "low" else "low"
        self._refresh()

    def action_start(self) -> None:
        self._sync_pattern_index()
        self.app.push_simulation(
            mode=self.mode,
            pattern_key=self.selected_pattern_key,
            speed=self.speed,
            density=self.density,
        )

    def action_show_controls(self) -> None:
        self.app.push_screen(ControlsModal())

    def on_key(self, event) -> None:
        if event.key in ("enter", "return"):
            self.action_start()
            event.prevent_default()
            event.stop()
