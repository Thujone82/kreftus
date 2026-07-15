"""Tests for sparkline window helpers."""

from __future__ import annotations

import unittest
from datetime import datetime, timedelta

from tp.sparkline import (
    build_sparkline,
    dashboard_sparkline_label,
    format_window_sparkline_rows,
    next_dashboard_sparkline_window,
    sparkline_windows,
    window_value_extremes,
)


class DashboardSparklineWindowTests(unittest.TestCase):
    def test_rotates_24h_72h_4h_for_less(self) -> None:
        self.assertEqual(next_dashboard_sparkline_window(24), ("72H", 72))
        self.assertEqual(next_dashboard_sparkline_window(72), ("4H", 4))
        self.assertEqual(next_dashboard_sparkline_window(4), ("24H", 24))

    def test_rotates_more_windows(self) -> None:
        self.assertEqual(
            next_dashboard_sparkline_window(24, time_detail="more"),
            ("36H", 36),
        )
        self.assertEqual(
            next_dashboard_sparkline_window(36, time_detail="more"),
            ("72H", 72),
        )
        self.assertEqual(
            next_dashboard_sparkline_window(72, time_detail="more"),
            ("90M", 1.5),
        )
        self.assertEqual(
            next_dashboard_sparkline_window(1.5, time_detail="more"),
            ("4H", 4),
        )
        self.assertEqual(
            next_dashboard_sparkline_window(12, time_detail="more"),
            ("24H", 24),
        )

    def test_unknown_hours_reset_to_default(self) -> None:
        self.assertEqual(next_dashboard_sparkline_window(12), ("24H", 24))

    def test_dashboard_label(self) -> None:
        self.assertEqual(dashboard_sparkline_label(72), "72H")
        self.assertEqual(dashboard_sparkline_label(99), "24H")
        self.assertEqual(
            dashboard_sparkline_label(8, time_detail="more"),
            "8H",
        )
        self.assertEqual(
            dashboard_sparkline_label(1.5, time_detail="more"),
            "90M",
        )


class SparklineWindowsTests(unittest.TestCase):
    def test_less_and_more_sets(self) -> None:
        self.assertEqual(
            [label for label, _hours in sparkline_windows("less")],
            ["4H", "24H", "72H"],
        )
        self.assertEqual(
            [label for label, _hours in sparkline_windows("more")],
            ["90M", "4H", "8H", "12H", "24H", "36H", "72H"],
        )


class WindowValueExtremesTests(unittest.TestCase):
    def setUp(self) -> None:
        self.end = datetime(2026, 6, 24, 16, 0, 0)
        self.points = [
            (self.end - timedelta(hours=50), 68.0),
            (self.end - timedelta(hours=2), 74.0),
            (self.end - timedelta(minutes=30), 82.0),
        ]

    def test_extremes_follow_selected_window(self) -> None:
        short_min, short_max = window_value_extremes(
            self.points,
            end_time=self.end,
            hours=4,
        )
        long_min, long_max = window_value_extremes(
            self.points,
            end_time=self.end,
            hours=72,
        )
        self.assertEqual((short_min, short_max), (74.0, 82.0))
        self.assertEqual((long_min, long_max), (68.0, 82.0))

    def test_extremes_use_raw_samples_not_bin_averages(self) -> None:
        points = [
            (self.end - timedelta(minutes=5), 70.0),
            (self.end - timedelta(minutes=4), 90.0),
            (self.end - timedelta(minutes=3), 70.0),
        ]
        sample_min, sample_max = window_value_extremes(
            points,
            end_time=self.end,
            hours=4,
        )
        spark = build_sparkline(points, end_time=self.end, hours=4)
        self.assertEqual((sample_min, sample_max), (70.0, 90.0))
        self.assertEqual(spark.min_value, spark.max_value)
        self.assertAlmostEqual(spark.min_value or 0.0, 76.666, places=2)


class FormatWindowSparklineRowsTests(unittest.TestCase):
    def test_emits_4h_24h_72h_in_order(self) -> None:
        end = datetime(2026, 6, 24, 16, 0, 0)
        points = [
            (end - timedelta(hours=50), 68.0),
            (end - timedelta(hours=2), 74.0),
            (end - timedelta(minutes=30), 82.0),
        ]
        rows = format_window_sparkline_rows(points, lambda _value: "white")
        self.assertEqual(len(rows), 3)
        self.assertIn("4H Ago", rows[0])
        self.assertIn("24H Ago", rows[1])
        self.assertIn("72H Ago", rows[2])

    def test_more_emits_seven_windows(self) -> None:
        end = datetime(2026, 6, 24, 16, 0, 0)
        points = [(end - timedelta(hours=1), 70.0)]
        rows = format_window_sparkline_rows(
            points,
            lambda _value: "white",
            time_detail="more",
        )
        self.assertEqual(len(rows), 7)
        self.assertIn("90M Ago", rows[0])
        self.assertIn("8H Ago", rows[2])
        self.assertIn("36H Ago", rows[5])


if __name__ == "__main__":
    unittest.main()
