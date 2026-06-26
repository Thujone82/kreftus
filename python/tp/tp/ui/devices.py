"""Manage devices screen."""

from __future__ import annotations

import asyncio
from datetime import datetime

from textual import work
from textual.app import ComposeResult
from textual.containers import Vertical, VerticalScroll
from textual.css.query import NoMatches
from textual.screen import ModalScreen, Screen
from textual.widgets import Header, Input, Label, Static

from tp.ble import DayHistoryProgress
from tp.ble import DayHistoryProgress
from tp.ble_radio import ensure_bluetooth_enabled_for_polling
from tp.config import default_device_name
from tp.history_fetch import DayHistoryResult, fetch_day_history_for_device
from tp.ui.device_status import format_device_status
from tp.ui.history_fetch_status import format_history_fetch_status

_IDLE_FOOTER = (
    "[dim]D discover · A add · I status · H history fetch · E edit · R remove · "
    "W up · S down · M menu[/]"
)
_BUSY_FOOTER = "[dim]Please wait — scan in progress[/]"
_BUSY_HISTORY_FOOTER = "[dim]Please wait — history fetch in progress[/]"


class NameInputModal(ModalScreen[str | None]):
    """Prompt for a device display name."""

    DEFAULT_CSS = """
    NameInputModal {
        align: center middle;
    }
    #name-dialog {
        width: 60;
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
        with Vertical(id="name-dialog"):
            yield Label(self.title)
            yield Input(value=self.default, id="name-input")
            yield Static("[dim]Enter to save · Q to cancel[/]", id="name-hint")

    def on_mount(self) -> None:
        self.query_one("#name-input", Input).focus()

    def on_input_submitted(self, _event: Input.Submitted) -> None:
        value = self.query_one("#name-input", Input).value.strip()
        self.dismiss(value or None)


class HistoryLoadPromptModal(ModalScreen[bool]):
    """Ask whether to load BLE history when adding a device."""

    DEFAULT_CSS = """
    HistoryLoadPromptModal {
        align: center middle;
    }
    #history-prompt-dialog {
        width: 70;
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

    def __init__(self, device_name: str) -> None:
        super().__init__()
        self.device_name = device_name

    def compose(self) -> ComposeResult:
        with Vertical(id="history-prompt-dialog"):
            yield Label(f"Load sensor history for {self.device_name}?")
            yield Static(
                "[dim]Backfills sparklines from the sensor (up to 1 year). "
                "May take a while.[/]",
                id="history-prompt-body",
            )
            yield Static("[dim]Y yes · N no · Q skip[/]", id="history-prompt-hint")

    def action_choose_yes(self) -> None:
        self.dismiss(True)

    def action_choose_no(self) -> None:
        self.dismiss(False)

    def action_quit_or_back(self) -> None:
        self.dismiss(False)


class DeviceStatusModal(ModalScreen[None]):
    """Show last fetch and log preload results for a managed device."""

    DEFAULT_CSS = """
    DeviceStatusModal {
        align: center middle;
    }
    #status-dialog {
        width: 80;
        height: auto;
        max-height: 90%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    def __init__(self, mac: str, device_name: str) -> None:
        super().__init__()
        self.device_mac = mac
        self.device_name = device_name

    def compose(self) -> ComposeResult:
        with Vertical(id="status-dialog"):
            yield VerticalScroll(Static("", id="status-body"), id="status-scroll")
            yield Static("[dim]Q to close[/]", id="status-hint")

    def on_mount(self) -> None:
        body = self.query_one("#status-body", Static)
        body.update(
            format_device_status(
                self.app.config,
                self.app.history,
                self.device_mac,
                self.device_name,
            )
        )


class DeviceHistoryFetchModal(ModalScreen[None]):
    """Fetch BLE history for a managed device."""

    DEFAULT_CSS = """
    DeviceHistoryFetchModal {
        align: center middle;
    }
    #history-fetch-dialog {
        width: 80;
        height: auto;
        max-height: 90%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    def __init__(self, mac: str, device_name: str) -> None:
        super().__init__()
        self.device_mac = mac
        self.device_name = device_name
        self._started_at = datetime.now()
        self._progress: DayHistoryProgress | None = DayHistoryProgress(
            phase="preparing",
            message="Preparing history fetch…",
        )
        self._result: DayHistoryResult | None = None
        self._active = True
        self._elapsed_timer = None

    def compose(self) -> ComposeResult:
        body_text = format_history_fetch_status(
            self.device_name,
            self.device_mac,
            progress=self._progress,
            started_at=self._started_at,
        )
        with Vertical(id="history-fetch-dialog"):
            yield VerticalScroll(
                Static(body_text, id="history-fetch-body"),
                id="history-fetch-scroll",
            )
            hint = "[dim]Q cancel[/]" if self._active else "[dim]Q to close[/]"
            yield Static(hint, id="history-fetch-hint")

    def on_mount(self) -> None:
        self._refresh_body()
        self._elapsed_timer = self.set_interval(
            1.0,
            self._refresh_body,
            name="history-fetch-elapsed",
        )
        self._run_fetch()

    def on_unmount(self) -> None:
        if self._elapsed_timer is not None:
            self._elapsed_timer.stop()

    def _refresh_body(self) -> None:
        if not self.is_mounted:
            return
        try:
            body = self.query_one("#history-fetch-body", Static)
        except NoMatches:
            return
        body.update(
            format_history_fetch_status(
                self.device_name,
                self.device_mac,
                progress=self._progress,
                result=self._result,
                started_at=self._started_at,
            )
        )
        try:
            hint = self.query_one("#history-fetch-hint", Static)
        except NoMatches:
            return
        if self._active:
            hint.update("[dim]Q cancel[/]")
        else:
            hint.update("[dim]Q to close[/]")

    @work
    async def _run_fetch(self) -> None:
        async def progress(update: DayHistoryProgress) -> None:
            self._progress = update
            if self.is_mounted:
                self._refresh_body()

        try:
            result = await fetch_day_history_for_device(
                self.app.config,
                self.app.history,
                self.device_mac,
                self.device_name,
                progress,
                ble_wait_detail=self.app.describe_ble_wait,
            )
        except asyncio.CancelledError:
            return
        except Exception as exc:  # noqa: BLE001
            self._progress = None
            self._result = DayHistoryResult(ok=False, error=str(exc) or exc.__class__.__name__)
        else:
            self._progress = None
            self._result = result
            if result.ok:
                self.app.refresh_monitoring()
        finally:
            self._active = False
            if self._elapsed_timer is not None:
                self._elapsed_timer.stop()
            if self.is_mounted:
                self._refresh_body()

    def action_quit_or_back(self) -> None:
        self.dismiss(None)


class DevicesScreen(Screen):
    """Discover and manage TP35x devices."""

    BINDINGS = [
        ("d", "discover", "Discover"),
        ("a", "add", "Add"),
        ("e", "edit", "Edit"),
        ("r", "remove", "Remove"),
        ("i", "status", "Status"),
        ("h", "history_fetch", "History Fetch"),
        ("w", "move_up", "Up"),
        ("s", "move_down", "Down"),
        ("m", "menu", "Menu"),
    ]

    def __init__(self) -> None:
        super().__init__()
        self.discovered: list[tuple[str, str]] = []
        self.selected_index = 0
        self._scanning = False
        self._history_fetching = False

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Vertical(
            Static("", id="devices-body"),
            id="devices-container",
        )
        yield Static(_IDLE_FOOTER, id="devices-footer")

    def on_mount(self) -> None:
        self.refresh_view()
        self._auto_discover_if_empty()

    def on_screen_resume(self) -> None:
        self.app.reload_config()
        self.refresh_view()
        self._auto_discover_if_empty()

    def _auto_discover_if_empty(self) -> None:
        if self.app.config.devices or self._scanning:
            return
        self.action_discover()

    def check_action(self, action: str, parameters: tuple[object, ...]) -> bool | None:
        if self._history_fetching and action in {"history_fetch", "add"}:
            self.notify("Busy — history fetch in progress", severity="warning", timeout=4)
            return False
        if not self._scanning:
            return True
        self.notify("Busy — scanning for devices", severity="warning", timeout=4)
        return False

    def _refresh_footer(self) -> None:
        footer = self.query_one("#devices-footer", Static)
        if self._scanning:
            footer.update(_BUSY_FOOTER)
        elif self._history_fetching:
            footer.update(_BUSY_HISTORY_FOOTER)
        else:
            footer.update(_IDLE_FOOTER)

    def refresh_view(self) -> None:
        body = self.query_one("#devices-body", Static)
        lines = ["[bold yellow]Manage Devices[/]", ""]

        lines.append("[bold]Managed[/]")
        if not self.app.config.devices:
            lines.append("  [dim](none)[/]")
        else:
            for index, (mac, name) in enumerate(self.app.config.devices.items()):
                marker = ">" if index == self.selected_index else " "
                lines.append(f"  {marker} [white]{name}[/]  [dim]{mac}[/]")

        lines.append("")
        lines.append("[bold]Discovered[/]  [dim](D discover · A add selected)[/]")
        if self._scanning:
            lines.append("  [dim](scanning…)[/]")
        elif not self.discovered:
            lines.append("  [dim](none — press D to scan)[/]")
        else:
            base = len(self.app.config.devices)
            for offset, (mac, name) in enumerate(self.discovered):
                index = base + offset
                marker = ">" if index == self.selected_index else " "
                lines.append(f"  {marker} [cyan]{name}[/]  [dim]{mac}[/]")

        body.update("\n".join(lines))
        self._refresh_footer()

    def _all_entries(self) -> list[tuple[str, str, str]]:
        managed = [(mac, name, "managed") for mac, name in self.app.config.devices.items()]
        discovered = [(mac, name, "discovered") for mac, name in self.discovered]
        return managed + discovered

    @work
    async def action_discover(self) -> None:
        if self._scanning:
            return
        self._scanning = True
        body = self.query_one("#devices-body", Static)
        body.update("[yellow]Scanning for TP35x devices…[/]")
        self._refresh_footer()
        try:
            scanned = await self.app.run_ble(self.app.ble_scan)
        except Exception as exc:  # noqa: BLE001
            body.update(f"[red]Scan failed: {exc}[/]")
            return
        finally:
            self._scanning = False

        existing = set(self.app.config.devices)
        self.discovered = [
            (device.address, device.name)
            for device in scanned
            if device.address not in existing
        ]
        self.selected_index = len(self.app.config.devices)
        self.refresh_view()

    def _open_history_fetch(self, mac: str, name: str) -> None:
        def on_close(_result: None) -> None:
            self._history_fetching = False
            self._refresh_footer()

        self._history_fetching = True
        self._refresh_footer()
        self.app.push_screen(DeviceHistoryFetchModal(mac, name), on_close)

    def _commit_added_device(self, mac: str, name: str, *, load_history: bool) -> None:
        self.app.config.devices[mac] = name
        self.app.save_config()
        self.app.refresh_monitoring(restart_worker=True)
        self.discovered = [entry for entry in self.discovered if entry[0] != mac]
        self.selected_index = min(
            self.selected_index, max(0, len(self._all_entries()) - 1)
        )
        self.refresh_view()
        if load_history:
            self._open_history_fetch(mac, name)

    def action_add(self) -> None:
        if self._scanning or self._history_fetching or not self.discovered:
            return
        base = len(self.app.config.devices)
        offset = self.selected_index - base
        if offset < 0 or offset >= len(self.discovered):
            return
        mac, discovered_name = self.discovered[offset]

        def finish_name(name: str | None) -> None:
            if not name:
                return

            def finish_history_prompt(load_history: bool) -> None:
                self._commit_added_device(mac, name, load_history=load_history)

            self.app.push_screen(HistoryLoadPromptModal(name), finish_history_prompt)

        self.app.push_screen(
            NameInputModal(
                "Device name:",
                default=default_device_name(mac, bluetooth_name=discovered_name),
            ),
            finish_name,
        )

    def _managed_device_count(self) -> int:
        return len(self.app.config.devices)

    def _reorder_managed_device(self, direction: int) -> None:
        """Move selected managed device up (-1) or down (+1) in display order."""
        count = self._managed_device_count()
        if count < 2 or self.selected_index >= count:
            return
        items = list(self.app.config.devices.items())
        index = self.selected_index
        target = index + direction
        if target < 0 or target >= count:
            return
        items[index], items[target] = items[target], items[index]
        self.app.config.devices = dict(items)
        self.selected_index = target
        self.app.save_config()
        self.app.refresh_monitoring()
        self.refresh_view()

    def action_move_up(self) -> None:
        if self._scanning:
            return
        self._reorder_managed_device(-1)

    def action_move_down(self) -> None:
        if self._scanning:
            return
        self._reorder_managed_device(1)

    def action_edit(self) -> None:
        if self._scanning:
            return
        entries = list(self.app.config.devices.items())
        if self.selected_index >= len(entries):
            return
        mac, current = entries[self.selected_index]

        def finish(name: str | None) -> None:
            if not name:
                return
            self.app.config.devices[mac] = name
            self.app.save_config()
            self.app.refresh_monitoring()
            self.refresh_view()

        self.app.push_screen(
            NameInputModal("New device name:", default=current),
            finish,
        )

    def action_status(self) -> None:
        if self._scanning:
            return
        entries = list(self.app.config.devices.items())
        if self.selected_index >= len(entries):
            self.notify("Select a managed device for status.", severity="warning")
            return
        mac, name = entries[self.selected_index]
        self.app.push_screen(DeviceStatusModal(mac, name))

    @work
    async def action_history_fetch(self) -> None:
        if self._scanning or self._history_fetching:
            return
        entries = list(self.app.config.devices.items())
        if self.selected_index >= len(entries):
            self.notify("Select a managed device for history fetch.", severity="warning")
            return
        mac, name = entries[self.selected_index]
        if not await ensure_bluetooth_enabled_for_polling():
            self.notify(
                "Bluetooth is off — enable Bluetooth to fetch history.",
                severity="warning",
                timeout=5,
            )
            return
        self._open_history_fetch(mac, name)

    def action_remove(self) -> None:
        if self._scanning:
            return
        entries = list(self.app.config.devices.items())
        if self.selected_index >= len(entries):
            return
        mac, _ = entries[self.selected_index]
        del self.app.config.devices[mac]
        self.app.history.clear_device(mac)
        self.app.save_config()
        self.app.refresh_monitoring(restart_worker=True)
        self.selected_index = max(0, self.selected_index - 1)
        self.refresh_view()

    def action_menu(self) -> None:
        if self._scanning:
            self.notify("Busy — scanning for devices", severity="warning", timeout=4)
            return
        self.app.pop_or_main_menu()

    def key_up(self) -> None:
        if self._scanning:
            return
        entries = self._all_entries()
        if not entries:
            return
        self.selected_index = (self.selected_index - 1) % len(entries)
        self.refresh_view()

    def key_down(self) -> None:
        if self._scanning:
            return
        entries = self._all_entries()
        if not entries:
            return
        self.selected_index = (self.selected_index + 1) % len(entries)
        self.refresh_view()
