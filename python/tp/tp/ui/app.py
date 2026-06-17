"""Textual application root."""

from __future__ import annotations

import asyncio
import sys
from collections.abc import Awaitable, Callable
from typing import Any, TypeVar

from textual.app import App
from textual.binding import Binding
from textual.screen import ModalScreen

from tp.ble import scan_devices
from tp.config import AppConfig, load_config, save_config
from tp.debug_log import set_debug_enabled
from tp.history import DeviceHistory, load_readings_from_log
from tp.ui.devices import DevicesScreen
from tp.ui.menus import MainMenuScreen
from tp.ui.monitoring import MonitoringScreen
from tp.ui.options import OptionsScreen

T = TypeVar("T")

WINDOW_TITLE = "TemPy"


def set_terminal_window_title(title: str = WINDOW_TITLE) -> None:
    """Set the host terminal tab/window title (not the in-app header)."""
    if sys.platform == "win32":
        try:
            import ctypes

            ctypes.windll.kernel32.SetConsoleTitleW(title)
        except (AttributeError, OSError):
            pass
    try:
        from rich.console import Console

        Console().set_window_title(title)
    except Exception:  # noqa: BLE001
        pass


def reload_app_config(app: "TPApp") -> None:
    """Reload tp.ini into the running app (e.g. after device renames)."""
    ini_path = app.config.ini_path
    app.config = load_config(ini_path)


def refresh_monitoring_screen(app: "TPApp", *, restart_worker: bool = False) -> None:
    """Refresh monitoring labels if that screen is mounted."""
    try:
        screen = app.get_screen("monitoring")
    except KeyError:
        return
    if not isinstance(screen, MonitoringScreen):
        return
    screen.sync_from_config()
    if not screen.is_mounted:
        if restart_worker or screen.devices_changed_since_worker_start():
            screen._needs_worker_restart = True
        return
    if restart_worker or screen.devices_changed_since_worker_start():
        screen.request_worker_restart()
    else:
        screen.refresh_display()


class TPApp(App):
    """TemPy — ThermoPro TP35x monitor."""

    TITLE = "🌡 TemPy"
    BINDINGS = [
        Binding("q", "quit_or_back", "Quit", priority=True, show=False),
    ]
    CSS = """
    Screen {
        background: $surface;
    }
    #menu-container, #devices-container, #options-container {
        padding: 1 2;
    }
    #monitor-scroll {
        padding: 1 2;
    }
    #monitor-footer, #devices-footer {
        dock: bottom;
        padding: 0 2;
        height: 1;
        background: $boost;
        color: $text-muted;
    }
    """

    SCREENS = {
        "main": MainMenuScreen,
        "monitoring": MonitoringScreen,
        "devices": DevicesScreen,
        "options": OptionsScreen,
    }

    def __init__(
        self,
        config: AppConfig | None = None,
        *,
        debug_enabled: bool = False,
        poll_enabled: bool = True,
        device_filter: str | None = None,
    ) -> None:
        super().__init__()
        self.config = config or load_config()
        self.history = DeviceHistory()
        self.debug_enabled = debug_enabled
        self.poll_enabled = poll_enabled
        self.device_filter = device_filter
        if debug_enabled:
            set_debug_enabled(self.config, True)
        load_readings_from_log(self.history, self.config)

    def on_mount(self) -> None:
        self.console.set_window_title(WINDOW_TITLE)
        self.push_screen("main")
        if self.config.devices:
            self.push_screen("monitoring")
            self.refresh_monitoring()
        else:
            self.push_screen("devices")

    def pop_or_main_menu(self) -> None:
        """Pop a sub-screen, or open main menu if it is the only screen."""
        if len(self.screen_stack) > 1:
            self.pop_screen()
        else:
            self.switch_screen("main")

    def action_quit_or_back(self) -> None:
        """Q: dismiss modal, pop sub-screen, or exit from main menu."""
        screen = self.screen
        if isinstance(screen, ModalScreen):
            screen.dismiss(None)
            return
        if isinstance(screen, MainMenuScreen):
            self.exit()
            return
        self.pop_or_main_menu()

    def save_config(self) -> None:
        save_config(self.config)

    def reload_config(self) -> None:
        reload_app_config(self)

    def refresh_monitoring(self, *, restart_worker: bool = False) -> None:
        refresh_monitoring_screen(self, restart_worker=restart_worker)

    async def run_ble(self, func: Callable[..., Awaitable[T]], *args: Any, **kwargs: Any) -> T:
        """Run BLE coroutine on the app's asyncio loop."""
        return await func(*args, **kwargs)

    async def ble_scan(self) -> list:
        return await scan_devices()


def run_app(
    config: AppConfig | None = None,
    *,
    debug_enabled: bool = False,
    poll_enabled: bool = True,
    device_filter: str | None = None,
) -> None:
    set_terminal_window_title()
    app = TPApp(
        config=config,
        debug_enabled=debug_enabled,
        poll_enabled=poll_enabled,
        device_filter=device_filter,
    )
    app.run()
