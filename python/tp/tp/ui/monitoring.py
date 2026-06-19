"""Live monitoring dashboard."""

from __future__ import annotations

import asyncio
import re
from datetime import datetime

from textual import work
from textual.binding import Binding
from textual.events import Key
from textual.app import ComposeResult
from textual.containers import Vertical, VerticalScroll
from textual.screen import Screen
from textual.widget import Widget
from textual.widgets import Static

from tp.colors import humidity_color, temp_color
from tp.debug_log import is_enabled as debug_log_enabled
from tp.ble import NOW_READ_CONNECTING, format_ble_error
from tp.fetch import START_MARKER, run_fetch_cycle
from tp.history import PollResult, load_readings_from_log
from tp.history_fetch import bootstrap_sparklines_from_ble
from tp.scheduler import (
    POLL_INTERVAL,
    chunk_end,
    floor_to_boundary,
    is_measurement_stale,
    next_retry_time,
    seconds_until,
    stale_macs_for_chunk,
)
from tp.sparkline import build_sparkline, colored_sparkline_markup, format_sparkline_row
from tp.config import filter_devices
from tp.ui.devices import DeviceStatusModal
from tp.ui.helpers import (
    format_device_label_row,
    format_stats_row,
    info_hotkey_footer_label,
    info_hotkey_index,
    layout_device_blocks,
    max_columns_for_width,
    measure_blocks_column_width,
    plain_markup_len,
)

SPINNER_FRAMES = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
PHASE_IDLE = "idle"
PHASE_FETCHING = "fetching"
PHASE_COMMIT = "commit"
PHASE_HISTORY = "history"
PHASE_WAITING = "waiting"

_HEADER_CLOCK_WIDTH = 10
_MIN_TITLE_WIDTH = 10
_RICH_TAG = re.compile(r"\[[^\]]*\]")


def _plain_text_len(markup: str) -> int:
    return plain_markup_len(markup)


class MonitoringHeader(Widget):
    """Single-row header: status (left) | title (center when room) | clock (right)."""

    DEFAULT_CSS = """
    MonitoringHeader {
        dock: top;
        width: 100%;
        height: 1;
        max-height: 1;
        overflow: hidden;
        background: $panel;
        color: $foreground;
        layout: horizontal;
        align: center middle;
    }
    MonitoringHeader > .monitor-header-status {
        width: auto;
        min-width: 0;
        height: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        text-wrap: nowrap;
        content-align: left middle;
        padding: 0 1 0 0;
    }
    MonitoringHeader > #monitor-header-title {
        width: 1fr;
        min-width: 0;
        height: 1;
        overflow: hidden;
        text-overflow: ellipsis;
        text-wrap: nowrap;
        content-align: center middle;
    }
    MonitoringHeader > .header-clock {
        width: 10;
        min-width: 10;
        max-width: 10;
        height: 1;
        content-align: center middle;
        background: $foreground-darken-1 5%;
        text-opacity: 85%;
    }
    """

    def __init__(self, *, show_clock: bool = True) -> None:
        super().__init__()
        self._show_clock = show_clock

    def compose(self) -> ComposeResult:
        yield Static("", id="monitor-header-status", classes="monitor-header-status")
        yield Static("", id="monitor-header-title")
        if self._show_clock:
            yield Static("", id="monitor-header-clock", classes="header-clock")

    def on_mount(self) -> None:
        if self._show_clock:
            self._tick_clock()
            self.set_interval(1, self._tick_clock, name="header-clock")
        self.call_after_refresh(self.sync_title)

    def _tick_clock(self) -> None:
        if not self._show_clock:
            return
        try:
            self.query_one("#monitor-header-clock", Static).update(
                datetime.now().strftime("%X")
            )
        except Exception:  # noqa: BLE001
            pass

    def sync_title(self, *, status_markup: str | None = None) -> None:
        """Show centered title only when status and clock leave enough room."""
        try:
            title_widget = self.query_one("#monitor-header-title", Static)
        except Exception:  # noqa: BLE001
            return
        if status_markup is None:
            try:
                status_widget = self.query_one("#monitor-header-status", Static)
                status_markup = str(status_widget.render())
            except Exception:  # noqa: BLE001
                status_markup = ""
        title = self.app.title or "🌡 TemPy"
        title_len = _plain_text_len(title)
        status_len = _plain_text_len(status_markup)
        reserved = _HEADER_CLOCK_WIDTH + status_len + 2
        header_width = self.size.width
        has_room = header_width <= 0 or header_width - reserved >= _MIN_TITLE_WIDTH
        title_widget.display = has_room and title_len > 0
        if title_widget.display:
            title_widget.update(title)


def _progress_bar(current: int, total: int, width: int = 24) -> str:
    if total <= 0:
        return ""
    current = min(current, total)
    filled = int(width * current / total) if total else 0
    if filled >= width:
        bar = "=" * width
    else:
        bar = ("=" * filled) + ">" + ("." * (width - filled - 1))
    return f"[cyan]{bar}[/] [white]{current}/{total}[/]"


class MonitoringScreen(Screen):
    BINDINGS = [
        Binding("m", "menu", "Menu"),
        Binding("g", "fetch_now", "Fetch now"),
        Binding("c", "toggle_columns", "Columns", show=False),
    ]

    def __init__(self) -> None:
        super().__init__()
        self._display_columns = 1
        self._cached_max_columns = 1
        self._worker_task: asyncio.Task | None = None
        self._stop_event = asyncio.Event()
        self._errors: list[str] = []
        self._phase = PHASE_IDLE
        self._fetch_index = 0
        self._fetch_total = 0
        self._active_mac: str | None = None
        self._active_macs: set[str] = set()
        self._active_name: str | None = None
        self._spinner_index = 0
        self._wait_seconds: float | None = None
        self._worker_macs: frozenset[str] = frozenset()
        self._next_poll_at: datetime | None = None
        self._needs_worker_restart = False
        self._chunk_start: datetime | None = None
        self._last_retry_at: datetime | None = None
        self._wait_mode = "poll"
        self._stale_count = 0
        self._is_retry_cycle = False
        self._fetch_macs: frozenset[str] = frozenset()
        self._last_fetch_at: datetime | None = None
        self._active_fetch_step: str | None = None
        self._history_bootstrap_done = False
        self._had_fetch_success = False
        self._fetch_lock: asyncio.Lock | None = None

    def _get_fetch_lock(self) -> asyncio.Lock:
        if self._fetch_lock is None:
            self._fetch_lock = asyncio.Lock()
        return self._fetch_lock

    def _fetch_in_progress(self) -> bool:
        """True while a fetch cycle holds the lock or the UI phase is active."""
        if self._phase in {PHASE_FETCHING, PHASE_COMMIT, PHASE_HISTORY}:
            return True
        lock = self._fetch_lock
        return lock is not None and lock.locked()

    def _device_activity_in_progress(self) -> bool:
        """True while a device is actively shown as busy on the dashboard."""
        return self._fetch_in_progress()

    def _seed_fetch_success_from_history(self) -> None:
        """Treat recent successful fetches as proof polling worked before an outage."""
        if not self.app.config.devices:
            self._had_fetch_success = False
            return
        self._had_fetch_success = any(
            self.app.history.fetch_status(mac).ok for mac in self.app.config.devices
        )

    def compose(self) -> ComposeResult:
        yield MonitoringHeader(show_clock=True)
        yield VerticalScroll(
            Static("", id="monitor-body"),
            id="monitor-scroll",
            can_focus=False,
        )
        yield Static("", id="monitor-footer")

    def _footer_text(self) -> str:
        parts = ["[dim]m[/] Menu"]
        if not self._fetch_in_progress():
            parts.append("[dim]g[/] Fetch now")
        info_hint = info_hotkey_footer_label(len(self._visible_devices()))
        if info_hint:
            parts.append(info_hint)
        if self._cached_max_columns >= 2:
            parts.append("[dim]c[/] Columns")
        return "  ".join(parts)

    def on_key(self, event: Key) -> None:
        idx = info_hotkey_index(event.key)
        if idx is None:
            return
        visible = self._visible_devices()
        if idx >= min(len(visible), 10):
            return
        mac, name = visible[idx]
        event.prevent_default()
        event.stop()
        self.app.push_screen(DeviceStatusModal(mac, name))

    def _refresh_footer(self) -> None:
        try:
            self.query_one("#monitor-footer", Static).update(self._footer_text())
        except Exception:  # noqa: BLE001
            pass

    def _visible_devices(self) -> list[tuple[str, str]]:
        return filter_devices(
            self.app.config.devices,
            getattr(self.app, "device_filter", None),
        )

    def _poll_scheduling_enabled(self) -> bool:
        return getattr(self.app, "poll_enabled", True)

    def _reload_log_from_disk(self) -> None:
        """Re-import CSV log to pick up rows written by another TemPy instance."""
        if not self.is_mounted or self._fetch_in_progress():
            return
        load_readings_from_log(self.app.history, self.app.config)
        self._note_last_fetch_time()
        self.refresh_display()

    async def on_mount(self) -> None:
        self.sync_from_config()
        load_readings_from_log(self.app.history, self.app.config)
        self._seed_fetch_success_from_history()
        self._chunk_start = floor_to_boundary(datetime.now())
        self._note_last_fetch_time()
        if self._poll_scheduling_enabled():
            self._set_phase(PHASE_WAITING, wait_seconds=0)
        else:
            self._set_phase(PHASE_IDLE)
        self.refresh_display()
        self._stop_event.clear()
        if self._poll_scheduling_enabled():
            self._worker_task = asyncio.create_task(self._poll_worker())
        else:
            self.set_interval(
                POLL_INTERVAL.total_seconds(),
                self._reload_log_from_disk,
                name="nopoll-log-refresh",
            )
            self.run_worker(
                self._run_history_bootstrap_if_needed,
                name="history-bootstrap",
                exclusive=True,
            )
        self.set_interval(0.12, self._animate_spinner, name="fetch-spinner")
        self.set_interval(1.0, self._tick_header, name="header-status")
        self.call_after_refresh(self.refresh_display)

    def on_screen_resume(self) -> None:
        """Re-entering monitoring after Manage Devices — reload names and refresh."""
        self.sync_from_config()
        load_readings_from_log(self.app.history, self.app.config)
        self._note_last_fetch_time()
        if not self._poll_scheduling_enabled():
            self.refresh_display()
            return
        if self._needs_worker_restart or self.devices_changed_since_worker_start():
            self._needs_worker_restart = False
            self.request_worker_restart()
        else:
            self.refresh_display()
            self._ensure_worker()

    def sync_from_config(self) -> None:
        from tp.config import load_config

        self.app.config = load_config(self.app.config.ini_path)

    def devices_changed_since_worker_start(self) -> bool:
        return frozenset(self.app.config.devices) != self._worker_macs

    def request_worker_restart(self) -> None:
        """Cancel in-flight fetch and restart from current config."""
        if not self._poll_scheduling_enabled():
            if not self.is_mounted:
                return
            self.sync_from_config()
            load_readings_from_log(self.app.history, self.app.config)
            self._note_last_fetch_time()
            self.refresh_display()
            return
        if not self.is_mounted:
            self._needs_worker_restart = True
            return
        self.run_worker(self._restart_worker, name="monitor-restart", exclusive=True)

    async def _restart_worker(self) -> None:
        self._stop_event.set()
        task = self._worker_task
        if task is not None and not task.done():
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass
        self._stop_event.clear()
        self._active_mac = None
        self._active_macs = set()
        self._active_name = None
        self._next_poll_at = None
        self._chunk_start = None
        self._last_retry_at = None
        self._wait_mode = "poll"
        self._stale_count = 0
        self._is_retry_cycle = False
        self._fetch_macs: frozenset[str] = frozenset()
        self._last_fetch_at: datetime | None = None
        if self.is_mounted:
            self.refresh_display()
        self._worker_task = asyncio.create_task(self._poll_worker())

    def _ensure_worker(self) -> None:
        if not self._poll_scheduling_enabled():
            return
        if self._worker_task is None or self._worker_task.done():
            self._stop_event.clear()
            self._worker_task = asyncio.create_task(self._poll_worker())

    async def on_unmount(self) -> None:
        self._stop_event.set()
        if self._worker_task is not None:
            self._worker_task.cancel()
            try:
                await self._worker_task
            except asyncio.CancelledError:
                pass

    def _set_phase(
        self,
        phase: str,
        *,
        index: int = 0,
        total: int = 0,
        mac: str | None = None,
        name: str | None = None,
        wait_seconds: float | None = None,
    ) -> None:
        self._phase = phase
        self._fetch_index = index
        self._fetch_total = total
        self._active_mac = mac
        self._active_name = name
        self._wait_seconds = wait_seconds

    def _animate_spinner(self) -> None:
        if not self._fetch_in_progress():
            return
        self._spinner_index = (self._spinner_index + 1) % len(SPINNER_FRAMES)
        self._refresh_header()

    def _tick_header(self) -> None:
        if not self.is_mounted:
            return
        self._refresh_header()

    def _next_event_info(self) -> tuple[datetime, datetime, bool, int]:
        """Return boundary, next wake time, whether it is a retry, and stale count."""
        now = datetime.now()
        chunk_start = self._chunk_start or floor_to_boundary(now)
        boundary = chunk_end(chunk_start)
        stale = self._stale_macs()
        stale_count = len(stale)
        if stale_count:
            retry_at = next_retry_time(
                chunk_start=chunk_start,
                last_retry_at=self._last_retry_at,
                from_time=now,
            )
            if retry_at is not None and retry_at < boundary:
                return boundary, retry_at, True, stale_count
        return boundary, boundary, False, stale_count

    def action_menu(self) -> None:
        self.app.pop_or_main_menu()

    def check_action(self, action: str, parameters: tuple[object, ...]) -> bool | None:
        if action == "toggle_columns":
            return True
        return super().check_action(action, parameters)

    def _content_area_width(self) -> int | None:
        """Usable body width; None until the scroll area has been laid out."""
        try:
            scroll = self.query_one("#monitor-scroll")
            if scroll.size.width > 0:
                return scroll.size.width
            body = self.query_one("#monitor-body", Static)
            if body.size.width > 0:
                return body.size.width
        except Exception:  # noqa: BLE001
            pass
        if self.size.width > 0:
            return max(40, self.size.width - 4)
        return None

    def _clamp_display_columns(self) -> None:
        if self._cached_max_columns < 2:
            self._display_columns = 1
        elif self._display_columns > self._cached_max_columns:
            self._display_columns = self._cached_max_columns

    def action_toggle_columns(self) -> None:
        if self._cached_max_columns < 2:
            self._display_columns = 1
            return
        self._display_columns = (self._display_columns % self._cached_max_columns) + 1
        self.refresh_display()

    def _begin_fetch_ui(self, macs: frozenset[str]) -> None:
        """Show fetching state immediately (before BLE work starts)."""
        self._fetch_macs = macs
        self._active_macs = set()
        self._active_mac = None
        self._active_name = None
        self._active_fetch_step = None
        self._fetch_total = len(macs)
        self._fetch_index = 0
        self._is_retry_cycle = False
        self._set_phase(PHASE_FETCHING, index=0, total=self._fetch_total)
        self.refresh_display()

    @work
    async def action_fetch_now(self) -> None:
        if self._fetch_in_progress():
            self.notify("Fetch already in progress.", severity="warning")
            return
        stale = self._stale_macs()
        if stale:
            await self._run_cycle(only_macs=stale)
            return
        self._chunk_start = floor_to_boundary(datetime.now())
        self._last_retry_at = None
        await self._run_cycle()

    async def _run_history_bootstrap_if_needed(self) -> None:
        """When logging is off, import 24H BLE history so dashboard sparklines populate."""
        if self._history_bootstrap_done or self.app.config.settings.logging_enabled:
            return
        if not self.app.config.devices:
            return

        self._history_bootstrap_done = True

        async def on_device_start(mac: str, name: str, index: int, total: int) -> None:
            self._active_macs = {mac}
            self._active_mac = mac
            self._active_name = name
            self._active_fetch_step = NOW_READ_CONNECTING
            self._set_phase(
                PHASE_HISTORY,
                index=index,
                total=total,
                mac=mac,
                name=name,
            )
            self.refresh_display()

        errors = await bootstrap_sparklines_from_ble(
            self.app.config,
            self.app.history,
            on_device_start=on_device_start,
            stop_requested=self._stop_event.is_set,
        )
        self._active_macs = set()
        self._active_mac = None
        self._active_name = None
        self._active_fetch_step = None
        if errors:
            self._errors = [*self._errors, *errors]
        if self.is_mounted and self._phase == PHASE_HISTORY:
            if self._poll_scheduling_enabled():
                self._set_phase(PHASE_WAITING, wait_seconds=0)
            else:
                self._set_phase(PHASE_IDLE)
            self.refresh_display()

    async def _poll_worker(self) -> None:
        self._worker_macs = frozenset(self.app.config.devices)
        if not self.app.config.devices:
            self._set_phase(PHASE_IDLE)
            self._errors = ["No devices configured — add devices first."]
            self.refresh_display()
            return

        self._chunk_start = floor_to_boundary(datetime.now())

        try:
            if not self._stop_event.is_set():
                await self._run_history_bootstrap_if_needed()
            if not self._stop_event.is_set():
                await self._run_startup_cycle()

            while not self._stop_event.is_set():
                await self._wait_for_next_event()
                if self._stop_event.is_set():
                    break
                if self._phase in {PHASE_FETCHING, PHASE_COMMIT}:
                    continue

                now = datetime.now()
                chunk_start = self._chunk_start or floor_to_boundary(now)
                if now >= chunk_end(chunk_start):
                    self._chunk_start = floor_to_boundary(now)
                    self._last_retry_at = None
                    await self._run_cycle()
                    continue

                stale = self._stale_macs()
                if stale:
                    await self._run_cycle(only_macs=stale)
        except asyncio.CancelledError:
            raise
        except Exception as exc:  # noqa: BLE001
            self._errors = [format_ble_error(exc)]
            self._set_phase(PHASE_IDLE)
            self.refresh_display()

    def _stale_macs(self) -> frozenset[str]:
        chunk_start = self._chunk_start or floor_to_boundary(datetime.now())
        return stale_macs_for_chunk(
            self.app.config.devices,
            self.app.history.last_updated,
            chunk_start,
        )

    def _note_last_fetch_time(self) -> None:
        self._last_fetch_at = self.app.history.latest_updated(self.app.config.devices)

    async def _run_startup_cycle(self) -> None:
        """Fetch only devices missing a fresh reading for the current chunk."""
        self._note_last_fetch_time()
        stale = self._stale_macs()
        if not stale:
            self._next_poll_at = chunk_end(self._chunk_start or floor_to_boundary(datetime.now()))
            self._set_phase(PHASE_WAITING, wait_seconds=0)
            self.refresh_display()
            return
        if len(stale) == len(self.app.config.devices):
            await self._run_cycle()
        else:
            await self._run_cycle(only_macs=stale)

    async def _wait_for_next_event(self) -> None:
        while not self._stop_event.is_set():
            now = datetime.now()
            chunk_start = self._chunk_start or floor_to_boundary(now)
            boundary = chunk_end(chunk_start)
            stale = self._stale_macs()
            self._stale_count = len(stale)

            if stale:
                retry_at = next_retry_time(
                    chunk_start=chunk_start,
                    last_retry_at=self._last_retry_at,
                    from_time=now,
                )
                if retry_at is not None and retry_at < boundary:
                    target = retry_at
                    self._wait_mode = "retry"
                else:
                    target = boundary
                    self._wait_mode = "poll"
            else:
                self._last_retry_at = None
                target = boundary
                self._wait_mode = "poll"

            self._next_poll_at = boundary
            remaining = seconds_until(now, target)
            if remaining <= 0:
                break

            if not self._fetch_in_progress():
                self._set_phase(PHASE_WAITING, wait_seconds=remaining)
                self.refresh_display()
            await asyncio.sleep(min(1.0, remaining))

    async def _set_now_read_phase(self, mac: str, name: str, phase: str) -> None:
        self._active_macs = {mac}
        self._active_mac = mac
        self._active_name = name
        self._active_fetch_step = phase
        self.refresh_display()

    async def _set_fetch_status(
        self, index: int, total: int, name: str, mac: str
    ) -> None:
        if name == START_MARKER:
            self._active_macs = set()
            self._active_fetch_step = None
            self._set_phase(PHASE_FETCHING, index=0, total=total)
        elif name == "Saving results":
            self._active_macs = set()
            self._active_mac = None
            self._active_name = None
            self._active_fetch_step = None
            self._set_phase(PHASE_COMMIT, index=index, total=total)
        else:
            self._active_macs = {mac} if mac else set()
            self._active_fetch_step = NOW_READ_CONNECTING
            self._set_phase(
                PHASE_FETCHING,
                index=index,
                total=total,
                mac=mac,
                name=name,
            )
        self.refresh_display()

    async def _on_device_result(self, _result: PollResult) -> None:
        self.refresh_display()

    async def _run_cycle(self, *, only_macs: frozenset[str] | None = None) -> None:
        async with self._get_fetch_lock():
            if self._stop_event.is_set():
                return
            macs = only_macs or frozenset(self.app.config.devices)
            if not self._fetch_in_progress():
                self._begin_fetch_ui(macs)
            self._is_retry_cycle = only_macs is not None
            self._fetch_macs = macs
            batch, fetch_errors = await run_fetch_cycle(
                self.app.config,
                self.app.history,
                only_macs=only_macs,
                progress=self._set_fetch_status,
                on_now_phase=self._set_now_read_phase,
                on_result=self._on_device_result,
                had_prior_success=self._had_fetch_success,
            )
            if any(result.reading is not None for result in batch):
                self._had_fetch_success = True
            self._errors = fetch_errors
            self._active_macs = set()
            self._active_fetch_step = None
            now = datetime.now()
            if only_macs is None:
                self._chunk_start = floor_to_boundary(now)
                self._last_retry_at = None
            else:
                self._last_retry_at = now
            self._next_poll_at = chunk_end(self._chunk_start or floor_to_boundary(now))
            self._is_retry_cycle = False
            self._note_last_fetch_time()
            self._set_phase(PHASE_WAITING, wait_seconds=0)
            self.refresh_display()

    def _header_status_text(self) -> str:
        parts: list[str] = []
        if getattr(self.app, "debug_enabled", False) or debug_log_enabled():
            parts.append("[magenta]DEBUG[/]")

        device_filter = getattr(self.app, "device_filter", None)
        if device_filter:
            parts.append(f"[dim]Filter: {device_filter}[/]")

        if self._fetch_in_progress():
            spinner = SPINNER_FRAMES[self._spinner_index]
            bar = _progress_bar(self._fetch_index, self._fetch_total, width=12)
            if self._phase == PHASE_COMMIT:
                label = "Saving"
            elif self._phase == PHASE_HISTORY:
                label = "24H"
            elif self._is_retry_cycle:
                label = "Retry"
            else:
                label = "Fetch"
            device = self._status_device_label()
            parts.append(f"[bold]{spinner}[/] {label} {device} {bar}")
        elif self.app.config.devices:
            if self._poll_scheduling_enabled():
                boundary, _, _, stale_count = self._next_event_info()
                self._next_poll_at = boundary
                self._stale_count = stale_count
                poll_at = boundary.strftime("%H:%M")
                parts.append(f"[dim]Next poll: {poll_at}[/]")
            else:
                parts.append("[dim]Polling off[/]")

        return "  ".join(parts)

    def _status_device_label(self) -> str:
        if self._active_name:
            return self._active_name
        if self._active_mac:
            return self.app.config.devices.get(self._active_mac, self._active_mac)
        return "…"

    def _refresh_header(self) -> None:
        status_text = self._header_status_text()
        try:
            status = self.query_one("#monitor-header-status", Static)
        except Exception:  # noqa: BLE001
            return
        status.update(status_text)
        try:
            self.query_one(MonitoringHeader).sync_title(status_markup=status_text)
        except Exception:  # noqa: BLE001
            pass

    def refresh_display(self) -> None:
        if not self.is_mounted:
            return
        body = self.query_one("#monitor-body", Static)

        if not self.app.config.devices:
            body.update("[dim]No devices configured.[/]")
        else:
            visible = self._visible_devices()
            if not visible:
                filt = getattr(self.app, "device_filter", None) or ""
                body.update(f"[dim]No devices match filter '{filt}'.[/]")
            else:
                blocks = [self._device_block(mac, name) for mac, name in visible]
                column_width = measure_blocks_column_width(blocks)
                area_width = self._content_area_width()
                if area_width is not None:
                    self._cached_max_columns = max_columns_for_width(
                        area_width,
                        column_width,
                    )
                self._clamp_display_columns()
                body.update(
                    layout_device_blocks(
                        blocks,
                        column_width=column_width,
                        columns=self._display_columns,
                    )
                )

        self._refresh_header()
        self._refresh_footer()

    def _device_block(self, mac: str, name: str) -> list[str]:
        is_active = mac in self._active_macs and self._device_activity_in_progress()
        updated_dt = self.app.history.last_updated(mac)
        stale = is_measurement_stale(updated_dt)

        label = format_device_label_row(
            name,
            stale=stale,
            fetching=is_active,
            fetch_step=self._active_fetch_step if is_active else None,
        )

        temp_points = self.app.history.temp_points(mac)
        humid_points = self.app.history.humidity_points(mac)
        temp_spark = build_sparkline(temp_points)
        humid_spark = build_sparkline(humid_points)

        readings = self.app.history.get_readings(mac)
        latest = readings[-1] if readings else None
        temp_cur = latest.temp_f if latest else None
        humid_cur = float(latest.humidity_pct) if latest else None

        temp_stats = format_stats_row(
            "Temp °F",
            temp_cur,
            temp_spark.min_value,
            temp_spark.max_value,
            "°F",
            color_fn=temp_color,
        )
        humid_stats = format_stats_row(
            "Humid %",
            humid_cur,
            humid_spark.min_value,
            humid_spark.max_value,
            "%",
            color_fn=humidity_color,
        )

        if is_active and not self.app.history.has_data(mac):
            temp_stats = f"[dim]{temp_stats}[/]"
            humid_stats = f"[dim]{humid_stats}[/]"
        elif stale:
            temp_stats = f"[dim]{temp_stats}[/]"
            humid_stats = f"[dim]{humid_stats}[/]"

        temp_core = colored_sparkline_markup(temp_spark, temp_color)
        humid_core = colored_sparkline_markup(humid_spark, humidity_color)
        if is_active and not self.app.history.has_data(mac):
            temp_core = f"[dim]{temp_core}[/]"
            humid_core = f"[dim]{humid_core}[/]"
        elif stale:
            temp_core = f"[dim]{temp_core}[/]"
            humid_core = f"[dim]{humid_core}[/]"
        temp_line = format_sparkline_row(temp_core)
        humid_line = format_sparkline_row(humid_core)

        return [label, temp_stats, temp_line, humid_stats, humid_line]

    def on_resize(self) -> None:
        self.refresh_display()
