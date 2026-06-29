"""TUI simulation screen — full-terminal cell grid."""

from __future__ import annotations

import time

from textual.binding import Binding
from textual.app import ComposeResult
from textual.events import Resize
from textual.screen import Screen
from textual.widgets import Static

from gol.engine import GameOfLife, Mode
from gol.ui.tui.controls_modal import ControlsModal
from gol.ui.tui.render import (
    apply_follow_viewport,
    build_grid_markup,
    pattern_bounds,
    population_centroid,
    sim_delay_seconds,
    stats_bar_markup,
    terminal_grid_dims,
    viewport_topleft,
    window_title,
)


class SimulationScreen(Screen):
    """Run Conway's Game of Life using the entire terminal as the grid."""

    BINDINGS = [
        Binding("space", "toggle_play", "Play/Pause", show=False),
        Binding("n", "step", "Step", show=False),
        Binding("r", "reset", "Reset", show=False),
        Binding("up", "pan_up", "Up", show=False),
        Binding("down", "pan_down", "Down", show=False),
        Binding("left", "pan_left", "Left", show=False),
        Binding("right", "pan_right", "Right", show=False),
        Binding("w", "pan_up", "Up", show=False),
        Binding("s", "pan_down", "Down", show=False),
        Binding("a", "pan_left", "Left", show=False),
        Binding("d", "pan_right", "Right", show=False),
        Binding("plus", "speed_up", "Faster", show=False),
        Binding("minus", "speed_down", "Slower", show=False),
        Binding("c", "show_controls", "Controls", show=False),
        Binding("f", "toggle_follow", "Follow", show=False),
        Binding("p", "toggle_stats", "Stats", show=False),
    ]

    DEFAULT_CSS = """
    SimulationScreen {
        overflow: hidden;
        layers: grid overlay;
    }
    #grid {
        layer: grid;
        width: 100%;
        height: 100%;
        padding: 0;
        margin: 0;
    }
    #stats-bar {
        layer: overlay;
        dock: top;
        width: 100%;
        height: 1;
        background: transparent;
        padding: 0;
    }
    """

    def __init__(
        self,
        *,
        mode: Mode,
        pattern_key: str,
        speed: int,
    ) -> None:
        super().__init__()
        self.mode = mode
        self.pattern_key = pattern_key
        self.speed = speed
        self.game = GameOfLife(mode)
        self.running = False
        self.auto_follow = False
        self.show_stats = True
        self.view_x = 0
        self.view_y = 0
        self._grid_cols = 1
        self._grid_rows = 1
        self._sim_timer = None
        self._last_follow_at: float | None = None

    def compose(self) -> ComposeResult:
        yield Static("", id="grid", markup=True)
        yield Static("", id="stats-bar", markup=True)

    def on_mount(self) -> None:
        self._apply_terminal_size()
        self._load_pattern()
        self._refresh_display()
        self._restart_timer()

    def on_unmount(self) -> None:
        if self._sim_timer is not None:
            self._sim_timer.stop()
            self._sim_timer = None

    def on_resize(self, event: Resize) -> None:
        if self.mode == "wrapped":
            old_cols, old_rows = self._grid_cols, self._grid_rows
            self._apply_terminal_size()
            if (self._grid_cols, self._grid_rows) != (old_cols, old_rows):
                if self.game.set_wrapped_dimensions(self._grid_cols, self._grid_rows):
                    self.game.clear()
                    self.running = False
                self._load_pattern()
        else:
            self._apply_terminal_size()
        self._refresh_display()

    def _apply_terminal_size(self) -> None:
        self._grid_cols, self._grid_rows = terminal_grid_dims(self.app.size.width, self.app.size.height)
        if self.mode == "wrapped":
            self.game.set_wrapped_dimensions(self._grid_cols, self._grid_rows)

    def _load_pattern(self) -> None:
        if self.mode == "wrapped":
            center = (self._grid_cols / 2, self._grid_rows / 2)
        else:
            center = (self.view_x + self._grid_cols / 2, self.view_y + self._grid_rows / 2)
        self.game.load_pattern(self.pattern_key, viewport_center=center)
        if self.mode == "infinite":
            bounds = pattern_bounds(self.game.cells)
            if bounds:
                self.view_x, self.view_y = viewport_topleft(
                    bounds[0], bounds[1], self._grid_cols, self._grid_rows
                )

    def _center_on_population(self) -> None:
        centroid = population_centroid(self.game.cells)
        if centroid is None:
            return
        self.view_x, self.view_y = viewport_topleft(
            centroid[0], centroid[1], self._grid_cols, self._grid_rows
        )

    def _maybe_follow_population(self, *, force: bool = False) -> None:
        """Track population centroid: snap off-screen, else slow orthogonal pan."""
        if self.mode != "infinite" or not self.auto_follow:
            return
        centroid = population_centroid(self.game.cells)
        if centroid is None:
            return
        now = time.monotonic()
        self.view_x, self.view_y, self._last_follow_at = apply_follow_viewport(
            centroid,
            view_x=self.view_x,
            view_y=self.view_y,
            cols=self._grid_cols,
            rows=self._grid_rows,
            now=now,
            last_at=self._last_follow_at,
            force=force,
        )

    def _refresh_display(self) -> None:
        markup = build_grid_markup(
            self.game.cells,
            cols=self._grid_cols,
            rows=self._grid_rows,
            view_x=self.view_x,
            view_y=self.view_y,
            wrapped=self.mode == "wrapped",
        )
        self.query_one("#grid", Static).update(markup)
        stats_bar = self.query_one("#stats-bar", Static)
        if self.show_stats:
            stats_bar.display = True
            stats_bar.update(
                stats_bar_markup(
                    self.game.population,
                    self.game.generation,
                    self.app.size.width,
                )
            )
        else:
            stats_bar.display = False
        self.app.console.set_window_title(
            window_title(
                self.game.generation,
                self.game.population,
                running=self.running,
                infinite=self.mode == "infinite",
                auto_follow=self.auto_follow,
                show_corner_stats=self.show_stats,
            )
        )

    def _restart_timer(self) -> None:
        if self._sim_timer is not None:
            self._sim_timer.stop()
        if self.running:
            self._sim_timer = self.set_interval(
                sim_delay_seconds(self.speed),
                self._sim_tick,
                name="sim",
            )
        else:
            self._sim_timer = None

    def _sim_tick(self) -> None:
        self.game.step()
        self._maybe_follow_population()
        self._refresh_display()

    def action_toggle_follow(self) -> None:
        if self.mode != "infinite":
            return
        self.auto_follow = not self.auto_follow
        if self.auto_follow:
            self._maybe_follow_population(force=True)
        else:
            self._last_follow_at = None
        self._refresh_display()

    def action_toggle_stats(self) -> None:
        self.show_stats = not self.show_stats
        self._refresh_display()

    def action_toggle_play(self) -> None:
        self.running = not self.running
        self._restart_timer()
        self._refresh_display()

    def action_step(self) -> None:
        self.running = False
        self._restart_timer()
        self.game.step()
        self._maybe_follow_population(force=True)
        self._refresh_display()

    def action_reset(self) -> None:
        self.running = False
        self.auto_follow = False
        self._last_follow_at = None
        self._restart_timer()
        self._load_pattern()
        self._refresh_display()

    def action_show_controls(self) -> None:
        self.app.push_screen(ControlsModal())

    def _pan_if_allowed(self, dx: int, dy: int) -> None:
        if self.mode != "infinite":
            return
        if self.running and self.auto_follow:
            return
        self.view_x += dx
        self.view_y += dy
        self._refresh_display()

    def action_pan_up(self) -> None:
        self._pan_if_allowed(0, -1)

    def action_pan_down(self) -> None:
        self._pan_if_allowed(0, 1)

    def action_pan_left(self) -> None:
        self._pan_if_allowed(-1, 0)

    def action_pan_right(self) -> None:
        self._pan_if_allowed(1, 0)

    def action_speed_up(self) -> None:
        self.speed = min(200, self.speed + 10)
        self._restart_timer()

    def action_speed_down(self) -> None:
        self.speed = max(10, self.speed - 10)
        self._restart_timer()
