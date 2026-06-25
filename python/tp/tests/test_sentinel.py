"""Tests for false 32 °F / 10% sentinel filtering."""

from __future__ import annotations

import unittest
from datetime import datetime, timedelta

from tp.history import (
    DeviceHistory,
    PollResult,
    Reading,
    filter_false_sentinel_runs,
    is_false_sentinel_reading,
    sanitize_poll_result,
)


class SentinelFilterTests(unittest.TestCase):
    def setUp(self) -> None:
        self.base = datetime.now().replace(second=0, microsecond=0)

    def _reading(self, minutes: int, temp: float = 70.0, humidity: int = 40) -> Reading:
        return Reading(
            timestamp=self.base + timedelta(minutes=minutes),
            temp_f=temp,
            humidity_pct=humidity,
        )

    def _sentinel(self, minutes: int) -> Reading:
        return self._reading(minutes, temp=32.0, humidity=10)

    def test_identifies_sentinel(self) -> None:
        self.assertTrue(is_false_sentinel_reading(self._sentinel(0)))
        self.assertFalse(is_false_sentinel_reading(self._reading(0)))

    def test_keeps_isolated_sentinel(self) -> None:
        readings = [self._reading(0), self._sentinel(1), self._reading(2)]
        filtered = filter_false_sentinel_runs(readings)
        self.assertEqual(len(filtered), 3)

    def test_drops_consecutive_sentinel_run(self) -> None:
        readings = [
            self._reading(0),
            self._sentinel(1),
            self._sentinel(2),
            self._sentinel(3),
            self._reading(4),
        ]
        filtered = filter_false_sentinel_runs(readings)
        self.assertEqual([r.timestamp for r in filtered], [readings[0].timestamp, readings[4].timestamp])

    def test_extends_run_across_prior_reading(self) -> None:
        prior = self._sentinel(0)
        readings = [self._sentinel(1), self._sentinel(2), self._reading(3)]
        filtered = filter_false_sentinel_runs(readings, prior=prior)
        self.assertEqual(len(filtered), 1)
        self.assertEqual(filtered[0].timestamp, readings[2].timestamp)

    def test_import_readings_skips_sentinel_run(self) -> None:
        history = DeviceHistory()
        mac = "E0:A4:4B:A4:53:0D"
        history.import_readings(
            mac,
            [
                self._reading(0),
                self._sentinel(1),
                self._sentinel(2),
                self._reading(3),
            ],
        )
        self.assertEqual(len(history.get_readings(mac)), 2)

    def test_sanitize_poll_result_drops_sentinel_batch(self) -> None:
        history = DeviceHistory()
        mac = "E0:A4:4B:A4:53:0D"
        history.add_reading(mac, self._reading(0))
        result = PollResult(
            mac=mac,
            device_name="Office",
            reading=self._sentinel(3),
            readings=[self._sentinel(1), self._sentinel(2), self._sentinel(3)],
        )
        sanitized = sanitize_poll_result(history, result)
        self.assertIsNone(sanitized.reading)
        self.assertIsNone(sanitized.readings)


if __name__ == "__main__":
    unittest.main()
