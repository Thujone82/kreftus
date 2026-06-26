"""Startup screen while recent CSV log rows are loaded."""

from __future__ import annotations

import asyncio

from textual import work
from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Footer, Header, Static

from tp.config import resolved_log_path
from tp.history import load_readings_from_log
from tp.ui.progress import format_progress_bar

LOG_PRELOAD_UI_MIN_BYTES = 32 * 1024


def should_show_log_preload(config) -> bool:  # noqa: ANN001
    """True when the log is large enough to show a loading screen."""
    if not config.devices:
        return False
    log_path = resolved_log_path(config)
    try:
        return log_path.is_file() and log_path.stat().st_size >= LOG_PRELOAD_UI_MIN_BYTES
    except OSError:
        return False


class LogLoadScreen(Screen):
    """Load recent log rows on a worker thread with progress feedback."""

    BINDINGS = []

    def __init__(self) -> None:
        super().__init__()
        self._message = "Starting…"
        self._progress_current = 0
        self._progress_total = 100

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static("", id="log-load-body"),
            id="log-load-container",
        )
        yield Footer()

    def on_mount(self) -> None:
        self._refresh_body()
        self._run_load()

    def check_action(self, action: str, parameters: tuple[object, ...]) -> bool | None:
        if action == "quit_or_back":
            self.app.notify("Loading log — please wait…", severity="warning", timeout=3)
            return False
        return True

    def _refresh_body(self) -> None:
        if not self.is_mounted:
            return
        lines = [
            "[bold yellow]TemPy[/]",
            "",
            "[bold]Loading recent log data[/]",
            f"  {self._message}",
            "",
            "  "
            + format_progress_bar(
                self._progress_current,
                self._progress_total,
                width=28,
            ),
            "",
            "[dim]Reading the last 72 hours from the CSV log…[/]",
        ]
        self.query_one("#log-load-body", Static).update("\n".join(lines))

    def _set_progress(self, message: str, current: int, total: int) -> None:
        self._message = message
        self._progress_current = max(0, current)
        self._progress_total = max(1, total)
        self._refresh_body()

    def _finish_startup(self) -> None:
        self.app.log_preloaded = True
        while self.app.screen_stack and isinstance(self.app.screen, LogLoadScreen):
            self.app.pop_screen()
        self.app.enter_main_screens()

    @work
    async def _run_load(self) -> None:
        loop = asyncio.get_running_loop()

        def progress(message: str, current: int, total: int) -> None:
            loop.call_soon_threadsafe(self._set_progress, message, current, total)

        try:
            await asyncio.to_thread(
                load_readings_from_log,
                self.app.history,
                self.app.config,
                progress_cb=progress,
            )
        finally:
            loop.call_soon_threadsafe(self._finish_startup)
