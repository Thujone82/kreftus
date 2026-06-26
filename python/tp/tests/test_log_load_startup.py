"""Tests for startup log preload screen and navigation."""

from __future__ import annotations

import asyncio
import csv
import tempfile
import unittest
from datetime import datetime, timedelta
from pathlib import Path
from unittest.mock import patch

from tp.config import AppConfig, Settings, save_config
from tp.ui.app import TPApp
from tp.ui.log_load import LogLoadScreen, should_show_log_preload
from tp.ui.menus import MainMenuScreen
from tp.ui.monitoring import MonitoringScreen


class LogLoadStartupTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.mac = "F7:94:C0:18:DD:7D"
        self.log_path = self.root / "tp_log.csv"
        self.config = AppConfig(
            ini_path=self.root / "tp.ini",
            settings=Settings(
                logging_enabled=True,
                log_directory=str(self.root),
                log_file_name="tp_log.csv",
            ),
            devices={self.mac: "Guest Room"},
        )
        self._write_log([(datetime.now() - timedelta(hours=1), 70.0, 50)])

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def _write_log(self, rows: list[tuple[datetime, float, int]]) -> None:
        with self.log_path.open("w", encoding="utf-8", newline="") as handle:
            writer = csv.writer(handle, lineterminator="\n")
            writer.writerow(
                ["timestamp", "device", "temp_f", "humidity_pct", "mac"]
            )
            for timestamp, temp_f, humidity in rows:
                writer.writerow(
                    [
                        timestamp.strftime("%Y-%m-%d %H:%M:%S"),
                        "Guest Room",
                        f"{temp_f:.1f}",
                        humidity,
                        self.mac,
                    ]
                )

    def test_should_show_log_preload_for_large_log(self) -> None:
        padding = b"x" * (33 * 1024)
        with self.log_path.open("ab") as handle:
            handle.write(padding)
        self.assertTrue(should_show_log_preload(self.config))

    async def test_finish_startup_opens_monitoring(self) -> None:
        save_config(self.config)
        app = TPApp(config=self.config, poll_enabled=False)
        with patch("tp.ui.log_load.load_readings_from_log", return_value=0):
            async with app.run_test(size=(80, 24)) as pilot:
                while not isinstance(app.screen, MainMenuScreen):
                    app.pop_screen()
                load_screen = LogLoadScreen()
                app.push_screen(load_screen)
                await pilot.pause()
                load_screen._finish_startup()
                await pilot.pause()
                self.assertIsInstance(app.screen, MonitoringScreen)
                self.assertTrue(
                    any(isinstance(screen, MainMenuScreen) for screen in app.screen_stack)
                )
                self.assertTrue(app.log_preloaded)


if __name__ == "__main__":
    unittest.main()
