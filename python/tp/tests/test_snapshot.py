"""Tests for CLI snapshot output."""

from __future__ import annotations

import unittest
from datetime import datetime, timedelta

from tp.history import DeviceHistory, Reading
from tp.snapshot import snapshot_device_lines
from tp.sparkline import window_value_extremes


class SnapshotDeviceLinesTests(unittest.TestCase):
    def test_each_window_gets_temp_and_humid_stats(self) -> None:
        history = DeviceHistory()
        mac = "F7:94:C0:18:DD:7D"
        now = datetime.now().replace(second=0, microsecond=0)
        history.add_reading(
            mac,
            Reading(
                timestamp=now - timedelta(hours=50),
                temp_f=60.0,
                humidity_pct=40,
            ),
        )
        history.add_reading(
            mac,
            Reading(
                timestamp=now - timedelta(hours=2),
                temp_f=74.0,
                humidity_pct=52,
            ),
        )
        history.add_reading(
            mac,
            Reading(
                timestamp=now - timedelta(minutes=5),
                temp_f=82.0,
                humidity_pct=58,
            ),
        )

        lines = snapshot_device_lines(history, mac, "Office")
        temp_lines = [line for line in lines if "Temp °F" in line]
        humid_lines = [line for line in lines if "Humid %" in line]
        sparklines = [line for line in lines if "Ago |" in line]

        self.assertEqual(len(temp_lines), 3)
        self.assertEqual(len(humid_lines), 3)
        self.assertEqual(len(sparklines), 6)
        self.assertIn("4H Ago", sparklines[0])
        self.assertIn("24H Ago", sparklines[2])
        self.assertIn("72H Ago", sparklines[4])

        temp_points = history.temp_points(mac)
        short_min, short_max = window_value_extremes(temp_points, hours=4)
        long_min, long_max = window_value_extremes(temp_points, hours=72)
        self.assertIn(f"min {short_min:.1f}", temp_lines[0])
        self.assertIn(f"max {short_max:.1f}", temp_lines[0])
        self.assertIn(f"min {long_min:.1f}", temp_lines[2])
        self.assertIn(f"max {long_max:.1f}", temp_lines[2])
        self.assertNotEqual(temp_lines[0], temp_lines[2])

    def test_more_time_detail_emits_six_windows(self) -> None:
        history = DeviceHistory()
        mac = "F7:94:C0:18:DD:7D"
        now = datetime.now().replace(second=0, microsecond=0)
        history.add_reading(
            mac,
            Reading(
                timestamp=now - timedelta(minutes=5),
                temp_f=82.0,
                humidity_pct=58,
            ),
        )
        lines = snapshot_device_lines(
            history,
            mac,
            "Office",
            time_detail="more",
        )
        temp_lines = [line for line in lines if "Temp °F" in line]
        sparklines = [line for line in lines if "Ago |" in line]
        self.assertEqual(len(temp_lines), 6)
        self.assertEqual(len(sparklines), 12)
        self.assertIn("8H Ago", sparklines[2])
        self.assertIn("36H Ago", sparklines[8])


if __name__ == "__main__":
    unittest.main()
