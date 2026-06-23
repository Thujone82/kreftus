"""Bluetooth enable permission modal."""

from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Vertical
from textual.screen import ModalScreen
from textual.widgets import Label, Static


class BluetoothPermissionModal(ModalScreen[bool]):
    """Ask the user before TemPy enables Bluetooth."""

    DEFAULT_CSS = """
    BluetoothPermissionModal {
        align: center middle;
    }
    #bluetooth-prompt-dialog {
        width: 72;
        height: auto;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    BINDINGS = [
        ("y", "choose_yes", "Yes"),
        ("n", "choose_no", "No"),
    ]

    def __init__(self, title: str, body: str) -> None:
        super().__init__()
        self._title = title
        self._body = body

    def compose(self) -> ComposeResult:
        with Vertical(id="bluetooth-prompt-dialog"):
            yield Label(self._title)
            yield Static(self._body, id="bluetooth-prompt-body")
            yield Static("[dim]Y yes · N no · Q decline[/]", id="bluetooth-prompt-hint")

    def action_choose_yes(self) -> None:
        self.dismiss(True)

    def action_choose_no(self) -> None:
        self.dismiss(False)

    def action_quit_or_back(self) -> None:
        self.dismiss(False)
