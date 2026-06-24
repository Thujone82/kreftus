"""Tests for CSV log preload and in-memory retention."""

from __future__ import annotations

import csv
import tempfile
import unittest
from datetime import datetime, timedelta
from pathlib import Path
from unittest.mock import patch

from tp.config import AppConfig, Settings
from tp.history import (
    MEMORY_HISTORY_HOURS,
    DeviceHistory,
    Reading,
    load_readings_from_log,
)
from tp.sparkline import build_sparkline, populated_bin_count


class LogPreloadTests(unittest.TestCase):
    def setUp(self) -> None:
        self.mac = "F7:94:C0:18:DD:7D"
        self.temp_dir = tempfile.TemporaryDirectory()
        self.config = AppConfig(
            devices={self.mac: "Guest Room"},
            settings=Settings(
                logging_enabled=True,
                log_directory=str(self.temp_dir.name),
                log_file_name="tp.log",
            ),
        )
        self.history = DeviceHistory()
        self.log_path = Path(self.temp_dir.name) / "tp.log"

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

    def test_loads_rows_older_than_24h_for_72h_sparklines(self) -> None:
        now = datetime.now().replace(second=0, microsecond=0)
        rows = [
            (now - timedelta(hours=70), 70.0, 50),
            (now - timedelta(hours=48), 72.0, 52),
            (now - timedelta(hours=12), 74.0, 54),
            (now - timedelta(hours=1), 76.0, 56),
        ]
        self._write_log(rows)

        with patch("tp.history.resolved_log_path", return_value=self.log_path):
            loaded = load_readings_from_log(self.history, self.config)

        self.assertEqual(loaded, 4)
        points = self.history.temp_points(self.mac)
        result_72h = build_sparkline(points, hours=MEMORY_HISTORY_HOURS)
        populated = sum(1 for value in result_72h.binned_values if value is not None)
        self.assertGreaterEqual(populated, 3)
        self.assertEqual(
            populated_bin_count(points, hours=MEMORY_HISTORY_HOURS),
            self.history.log_load_status(self.mac).last_load_hour_bins,
        )

    def test_prune_keeps_72h_of_samples(self) -> None:
        now = datetime.now().replace(second=0, microsecond=0)
        for hours_ago in (80, 60, 30, 1):
            self.history.add_reading(
                self.mac,
                Reading(
                    timestamp=now - timedelta(hours=hours_ago),
                    temp_f=70.0 + hours_ago / 10,
                    humidity_pct=50,
                ),
            )

        timestamps = [r.timestamp for r in self.history.get_readings(self.mac)]
        self.assertEqual(len(timestamps), 3)
        self.assertTrue(all(ts >= now - timedelta(hours=MEMORY_HISTORY_HOURS) for ts in timestamps))


if __name__ == "__main__":
    unittest.main()
