"""Main menu screen."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import Screen
from textual.widgets import Footer, Header, Static


class MainMenuScreen(Screen):
    """Top-level menu."""

    BINDINGS = [
        ("1", "monitoring", "Monitoring"),
        ("2", "devices", "Devices"),
        ("3", "options", "Options"),
        ("4", "quit", "Exit"),
        ("5", "export_log", "Export"),
    ]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static("", id="menu-body"),
            id="menu-container",
        )
        yield Footer()

    def on_mount(self) -> None:
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
            "  [white]5[/]  Export log to web",
            "",
            "[dim]Press 1-5 · Q to quit[/]",
        ]
        body.update("\n".join(lines))

    def on_screen_resume(self) -> None:
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
