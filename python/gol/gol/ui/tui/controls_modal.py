"""TUI controls help modal."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import ModalScreen
from textual.widgets import Static

from gol.help_text import TUI_CONTROLS_MODAL_MARKUP


class ControlsModal(ModalScreen[None]):
    """Simulation controls reference."""

    BINDINGS = [
        ("q", "dismiss", "Close"),
        ("escape", "dismiss", "Close"),
    ]

    DEFAULT_CSS = """
    ControlsModal {
        align: center middle;
    }
    #controls-dialog {
        width: 58;
        height: auto;
        max-height: 90%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    def compose(self) -> ComposeResult:
        with Vertical(id="controls-dialog"):
            yield Static(TUI_CONTROLS_MODAL_MARKUP, markup=True)

    def action_dismiss(self) -> None:
        self.dismiss(None)
