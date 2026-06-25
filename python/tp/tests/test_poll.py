"""Tests for poll mode and incremental history sizing."""

from __future__ import annotations

import unittest
from datetime import datetime, timedelta
from pathlib import Path

from tp.config import (
    DEFAULT_POLL_MODE,
    POLL_MODE_INCREMENTAL,
    POLL_MODE_LIVE,
    AppConfig,
    Settings,
    load_config,
    poll_mode_label,
    probe_log_directory,
    rename_log_file,
    save_config,
)
from tp.history import DeviceHistory, PollResult, Reading, append_poll_results_to_log
from tp.poll import incremental_history_record_count, uses_incremental_history


class PollModeTests(unittest.TestCase):
    def test_defaults_to_incremental(self) -> None:
        self.assertEqual(Settings().poll_mode, POLL_MODE_INCREMENTAL)
        self.assertEqual(Settings().log_file_name, "tp_log.csv")

    def test_uses_incremental_history(self) -> None:
        self.assertTrue(uses_incremental_history(POLL_MODE_INCREMENTAL))
        self.assertFalse(uses_incremental_history(POLL_MODE_LIVE))

    def test_poll_mode_label(self) -> None:
        self.assertIn("incremental", poll_mode_label(POLL_MODE_INCREMENTAL))
        self.assertIn("live", poll_mode_label(POLL_MODE_LIVE))


class IncrementalCountTests(unittest.TestCase):
    def setUp(self) -> None:
        self.history = DeviceHistory()
        self.mac = "E0:A4:4B:A4:53:0D"
        self.now = datetime.now().replace(second=0, microsecond=0)

    def _reading(self, minutes_ago: int, temp: float = 70.0) -> Reading:
        return Reading(
            timestamp=self.now - timedelta(minutes=minutes_ago),
            temp_f=temp,
            humidity_pct=40,
        )

    def test_defaults_when_no_history(self) -> None:
        self.assertEqual(incremental_history_record_count(self.history, self.mac), 5)

    def test_requests_gap_since_last_minute(self) -> None:
        self.history.add_reading(self.mac, self._reading(5))
        self.assertEqual(incremental_history_record_count(self.history, self.mac), 5)

    def test_requests_at_least_one_for_same_minute(self) -> None:
        self.history.add_reading(self.mac, self._reading(0))
        self.assertEqual(incremental_history_record_count(self.history, self.mac), 1)


class LogRenameTests(unittest.TestCase):
    def setUp(self) -> None:
        import tempfile

        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.config = AppConfig(
            ini_path=self.root / "tp.ini",
            settings=Settings(
                log_directory=str(self.root),
                log_file_name="old.csv",
            ),
        )
        self.old_path = self.root / "old.csv"
        self.old_path.write_text(
            "timestamp,device,temp_f,humidity_pct,mac\n",
            encoding="utf-8",
        )

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def test_renames_existing_log(self) -> None:
        error = rename_log_file(self.config, "old.csv", "new.csv")
        self.assertIsNone(error)
        self.assertFalse(self.old_path.exists())
        self.assertTrue((self.root / "new.csv").is_file())

    def test_signals_when_destination_exists(self) -> None:
        (self.root / "new.csv").write_text("existing\n", encoding="utf-8")
        error = rename_log_file(self.config, "old.csv", "new.csv", overwrite=False)
        self.assertEqual(error, "exists")
        self.assertTrue(self.old_path.is_file())

    def test_overwrite_replaces_destination(self) -> None:
        new_path = self.root / "new.csv"
        new_path.write_text("existing\n", encoding="utf-8")
        error = rename_log_file(self.config, "old.csv", "new.csv", overwrite=True)
        self.assertIsNone(error)
        self.assertFalse(self.old_path.exists())
        self.assertIn("timestamp", new_path.read_text(encoding="utf-8"))

    def test_probe_log_directory_does_not_create_log_file(self) -> None:
        target = self.root / "would_be_log.csv"
        self.assertFalse(target.exists())
        dir_path, error = probe_log_directory(str(self.root))
        self.assertIsNone(error)
        self.assertEqual(dir_path, self.root.resolve())
        self.assertFalse(target.exists())


class MultiRowLogAppendTests(unittest.TestCase):
    def setUp(self) -> None:
        import tempfile

        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.config = AppConfig(
            ini_path=self.root / "tp.ini",
            settings=Settings(
                logging_enabled=True,
                log_directory=str(self.root),
                log_file_name="tp_log.csv",
            ),
        )
        self.now = datetime.now().replace(second=0, microsecond=0)

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def test_appends_all_incremental_readings(self) -> None:
        readings = [
            Reading(self.now - timedelta(minutes=2), 70.0, 40),
            Reading(self.now - timedelta(minutes=1), 71.0, 41),
            Reading(self.now, 72.0, 42),
        ]
        result = PollResult(
            mac="E0:A4:4B:A4:53:0D",
            device_name="Office",
            reading=readings[-1],
            readings=readings,
        )
        error = append_poll_results_to_log(self.config, [result])
        self.assertIsNone(error)
        lines = (self.root / "tp_log.csv").read_text(encoding="utf-8").strip().splitlines()
        self.assertEqual(len(lines), 4)
        self.assertIn("70.0", lines[1])
        self.assertIn("72.0", lines[3])


class ConfigRoundTripTests(unittest.TestCase):
    def test_save_and_load_poll_mode(self) -> None:
        import tempfile

        with tempfile.TemporaryDirectory() as tmp:
            ini_path = Path(tmp) / "tp.ini"
            config = AppConfig(
                ini_path=ini_path,
                settings=Settings(
                    poll_mode=POLL_MODE_LIVE,
                    log_file_name="tp_log.csv",
                ),
            )
            save_config(config)
            loaded = load_config(ini_path)
            self.assertEqual(loaded.settings.poll_mode, POLL_MODE_LIVE)
            self.assertEqual(loaded.settings.log_file_name, "tp_log.csv")
            self.assertEqual(DEFAULT_POLL_MODE, POLL_MODE_INCREMENTAL)


if __name__ == "__main__":
    unittest.main()
