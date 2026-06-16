"""Options screen."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import ModalScreen, Screen
from textual.widgets import Footer, Header, Input, Label, Static

from tp.config import probe_log_path, resolved_log_path
from tp.debug_log import log_path as debug_log_path
from tp.debug_log import set_debug_enabled


class TextInputModal(ModalScreen[str | None]):
    DEFAULT_CSS = """
    TextInputModal {
        align: center middle;
    }
    #text-dialog {
        width: 70;
        height: auto;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    def __init__(self, title: str, default: str = "") -> None:
        super().__init__()
        self.title = title
        self.default = default

    def compose(self) -> ComposeResult:
        with Vertical(id="text-dialog"):
            yield Label(self.title)
            yield Input(value=self.default, id="text-input")
            yield Static("[dim]Enter to save · Q to cancel[/]", id="text-hint")

    def on_mount(self) -> None:
        self.query_one("#text-input", Input).focus()

    def on_input_submitted(self, _event: Input.Submitted) -> None:
        value = self.query_one("#text-input", Input).value.strip()
        self.dismiss(value or None)


class OptionsScreen(Screen):
    BINDINGS = [
        ("l", "toggle_logging", "Logging"),
        ("b", "toggle_debug", "Debug"),
        ("d", "edit_directory", "Directory"),
        ("f", "edit_filename", "Filename"),
        ("m", "menu", "Menu"),
    ]

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static("", id="options-body"),
            id="options-container",
        )
        yield Footer()

    def on_mount(self) -> None:
        self.refresh_view()

    def refresh_view(self) -> None:
        settings = self.app.config.settings
        log_path = resolved_log_path(self.app.config)
        enabled = "on" if settings.logging_enabled else "off"
        debug_enabled = "on" if self.app.debug_enabled else "off"
        debug_path = debug_log_path()
        debug_lines = [
            f"  Debug log:     [cyan]{debug_enabled}[/]  [dim](B toggle, session only)[/]",
        ]
        if self.app.debug_enabled and debug_path is not None:
            debug_lines.append(f"  Debug file:    [dim]{debug_path}[/]")
        body = self.query_one("#options-body", Static)
        body.update(
            "\n".join(
                [
                    "[bold yellow]Options[/]",
                    "",
                    f"  Logging:       [cyan]{enabled}[/]  [dim](L toggle)[/]",
                    *debug_lines,
                    f"  Log directory: [white]{settings.log_directory}[/]  [dim](D edit)[/]",
                    f"  Log filename:  [white]{settings.log_file_name}[/]  [dim](F edit)[/]",
                    "",
                    f"  Resolved path: [dim]{log_path}[/]",
                    "",
                    "[dim]M menu[/]",
                ]
            )
        )

    def _sync_debug_log(self) -> None:
        set_debug_enabled(self.app.config, self.app.debug_enabled)

    def action_toggle_debug(self) -> None:
        self.app.debug_enabled = not self.app.debug_enabled
        self._sync_debug_log()
        self.refresh_view()
        self.app.refresh_monitoring()
        state = "enabled" if self.app.debug_enabled else "disabled"
        self.notify(f"Debug logging {state} for this session.", timeout=4)

    def action_toggle_logging(self) -> None:
        self.app.config.settings.logging_enabled = not self.app.config.settings.logging_enabled
        self.app.save_config()
        self.refresh_view()

    def action_edit_directory(self) -> None:
        current = self.app.config.settings.log_directory

        def finish(value: str | None) -> None:
            if value is None:
                return
            filename = self.app.config.settings.log_file_name
            _path, error = probe_log_path(value, filename)
            if error:
                self.notify(error, severity="error", timeout=8)
                return
            self.app.config.settings.log_directory = value
            self.app.save_config()
            self._sync_debug_log()
            self.refresh_view()

        self.app.push_screen(TextInputModal("Log directory:", default=current), finish)

    def action_edit_filename(self) -> None:
        current = self.app.config.settings.log_file_name

        def finish(value: str | None) -> None:
            if value is None:
                return
            if not value.strip():
                self.notify("Filename cannot be empty.", severity="error")
                return
            directory = self.app.config.settings.log_directory
            _path, error = probe_log_path(directory, value.strip())
            if error:
                self.notify(error, severity="error", timeout=8)
                return
            self.app.config.settings.log_file_name = value.strip()
            self.app.save_config()
            self.refresh_view()

        self.app.push_screen(TextInputModal("Log filename:", default=current), finish)

    def action_menu(self) -> None:
        self.app.pop_or_main_menu()
