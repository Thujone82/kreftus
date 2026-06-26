"""Main menu screen."""

from __future__ import annotations

from textual import work
from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Footer, Header, Static

from tp.log_export import can_export_log


class MainMenuScreen(Screen):
    """Top-level menu."""

    BINDINGS = [
        ("1", "monitoring", "Monitoring"),
        ("2", "devices", "Devices"),
        ("3", "options", "Options"),
        ("4", "quit", "Exit"),
        ("5", "export_log", "Export"),
    ]

    def __init__(self) -> None:
        super().__init__()
        self._export_available_cached: bool | None = None

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static("", id="menu-body"),
            id="menu-container",
        )
        yield Footer()

    def on_mount(self) -> None:
        self.refresh_menu()
        self._refresh_export_availability()

    def _export_available(self) -> bool:
        if self._export_available_cached is not None:
            return self._export_available_cached
        return False

    @work(thread=True)
    def _refresh_export_availability(self) -> None:
        available = can_export_log(self.app.config)
        self.app.call_from_thread(self._apply_export_availability, available)

    def _apply_export_availability(self, available: bool) -> None:
        if self._export_available_cached == available:
            return
        self._export_available_cached = available
        if self.is_mounted:
            self.refresh_menu()

    def refresh_menu(self) -> None:
        device_count = len(self.app.config.devices)
        body = self.query_one("#menu-body", Static)
        lines = [
            f"[bold yellow]{self.app.title}[/]",
            "",
            f"  [white]1[/]  Monitoring  [dim]({device_count} device(s))[/]",
            "  [white]2[/]  Manage Devices",
            "  [white]3[/]  Options",
            "  [white]4[/]  Exit",
        ]
        if self._export_available():
            lines.append("  [white]5[/]  Export log to web")
            lines.append("")
            lines.append("[dim]Press 1-5 · Q to quit[/]")
        else:
            lines.append("")
            lines.append("[dim]Press 1-4 · Q to quit[/]")
        body.update("\n".join(lines))

    def check_action(self, action: str, parameters: tuple[object, ...]) -> bool | None:
        if action == "export_log":
            if getattr(self.app, "log_export_in_progress", False):
                self.app.notify(
                    "Log export already in progress…",
                    severity="warning",
                    timeout=4,
                )
                return False
            if not self._export_available():
                return False
        return True

    def on_screen_resume(self) -> None:
        self._refresh_export_availability()
        self.refresh_menu()

    def action_monitoring(self) -> None:
        self.app.push_screen("monitoring")
        self.app.refresh_monitoring()

    def action_devices(self) -> None:
        self.app.push_screen("devices")

    def action_options(self) -> None:
        self.app.push_screen("options")

    def action_export_log(self) -> None:
        from tp.ui.log_export_action import export_log_to_web

        export_log_to_web(self.app)

    def action_quit(self) -> None:
        self.app.exit()
