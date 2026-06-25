"""Unit tests for day-history parsing and log merge."""

from __future__ import annotations

import csv
import asyncio
import tempfile
import unittest
from datetime import datetime, timedelta
from pathlib import Path

from tp.ble import (
    _decode_stream_history_chunks,
    _make_datetime_sync_cmd,
    _make_stream_history_cmds,
    _parse_day_packet,
    _parse_day_packets,
    _stream_history_to_readings,
)
from tp.config import AppConfig, Settings
from tp.history import (
    CSV_HEADER,
    BLE_HISTORY_HOURS,
    LOG_TIMESTAMP_FORMAT,
    SPARKLINE_BOOTSTRAP_MIN_BINS,
    DeviceHistory,
    Reading,
    apply_day_history,
    device_needs_sparkline_bootstrap,
    replace_device_log_window,
    replace_device_memory_window,
)
from tp.history_fetch import bootstrap_sparklines_from_ble


def _sample_packet(
    packet_index: int,
    *,
    temp_c_x10: int = 200,
    humidity: int = 50,
) -> list[int]:
    """Build one 0xA7 day-history notify packet with one valid sample."""
    lo = temp_c_x10 & 0xFF
    hi = (temp_c_x10 >> 8) & 0xFF
    return [
        0xA7,
        packet_index & 0xFF,
        (packet_index >> 8) & 0xFF,
        0x00,
        lo,
        hi,
        humidity,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
        0xFF,
    ]


class DayHistoryParserTests(unittest.TestCase):
    def test_parse_single_packet_timestamp_and_values(self) -> None:
        t0 = datetime(2026, 6, 15, 12, 0, 0)
        readings = _parse_day_packet(_sample_packet(1), t0)
        self.assertEqual(len(readings), 1)
        self.assertEqual(readings[0].timestamp, t0)
        self.assertAlmostEqual(readings[0].temp_f, 68.0, places=1)
        self.assertEqual(readings[0].humidity_pct, 50)

    def test_parse_multiple_packets_dedupes(self) -> None:
        t0 = datetime(2026, 6, 15, 12, 0, 0)
        packets = [_sample_packet(1), _sample_packet(2), _sample_packet(2)]
        readings = _parse_day_packets(packets, t0=t0)
        self.assertEqual(len(readings), 2)
        self.assertEqual(readings[0].timestamp, t0)
        self.assertEqual(readings[1].timestamp, t0 + timedelta(minutes=5))

    def test_skip_missing_temp_slots(self) -> None:
        t0 = datetime(2026, 6, 15, 12, 0, 0)
        packet = _sample_packet(1)
        packet[4] = 0xFF
        packet[5] = 0xFF
        self.assertEqual(_parse_day_packet(packet, t0), [])


class DayHistoryStreamParserTests(unittest.TestCase):
    def test_datetime_sync_checksum(self) -> None:
        cmd = _make_datetime_sync_cmd(datetime(2026, 6, 17, 8, 30, 0))
        self.assertEqual(cmd[0], 0xA5)
        self.assertEqual(cmd[-1], sum(cmd[:-1]) & 0xFF)

    def test_decode_stream_history_chunks(self) -> None:
        # One record: 19.1C (0x00BF LE), humidity 51; header count field + padding + checksum
        payload = bytes.fromhex("cccc0104000000bf0033a56666")
        pairs = _decode_stream_history_chunks([payload])
        self.assertEqual(len(pairs), 1)
        self.assertAlmostEqual(pairs[0][0], 19.1, places=1)
        self.assertEqual(pairs[0][1], 51)

    def test_stream_history_to_readings(self) -> None:
        anchor = datetime(2026, 6, 17, 12, 0, 0)
        readings = _stream_history_to_readings([(20.0, 40), (21.0, 41)], anchor)
        self.assertEqual(len(readings), 2)
        self.assertEqual(readings[0].timestamp, anchor - timedelta(minutes=1))
        self.assertEqual(readings[1].timestamp, anchor)
        self.assertAlmostEqual(readings[0].temp_f, 69.8, places=1)
        self.assertAlmostEqual(readings[1].temp_f, 68.0, places=1)

    def test_make_stream_history_cmds_count_endian(self) -> None:
        _, _, cmd3 = _make_stream_history_cmds(2000, datetime(2026, 6, 17, 8, 0, 0))
        self.assertEqual(cmd3[-5], 2000 & 0xFF)
        self.assertEqual(cmd3[-4], (2000 >> 8) & 0xFF)

    def test_stream_record_count_matches_ble_history_hours(self) -> None:
        from tp.ble import DAY_STREAM_RECORD_COUNT

        self.assertEqual(DAY_STREAM_RECORD_COUNT, int(BLE_HISTORY_HOURS * 60))


class DayHistoryMergeTests(unittest.TestCase):
    def setUp(self) -> None:
        self.office = "E0:A4:4B:A4:53:0D"
        self.other = "AA:BB:CC:DD:EE:FF"
        self.now = datetime.now().replace(second=0, microsecond=0)
        self.cutoff = self.now - timedelta(hours=72)

    def _reading(self, minutes_ago: int, temp_f: float, humidity: int) -> Reading:
        return Reading(
            timestamp=self.now - timedelta(minutes=minutes_ago),
            temp_f=temp_f,
            humidity_pct=humidity,
        )

    def test_replace_device_memory_window(self) -> None:
        history = DeviceHistory()
        history.add_reading(self.office, self._reading(30, 70.0, 40))
        history.add_reading(self.office, self._reading(10, 72.0, 41))
        new_rows = [self._reading(20, 71.0, 42), self._reading(5, 73.0, 43)]
        imported = replace_device_memory_window(history, self.office, new_rows)
        self.assertEqual(imported, 2)
        temps = [r.temp_f for r in history.get_readings(self.office)]
        self.assertIn(70.0, temps)
        self.assertIn(71.0, temps)
        self.assertIn(73.0, temps)
        self.assertNotIn(72.0, temps)

    def test_replace_device_memory_window_preserves_data_before_span(self) -> None:
        """Partial BLE history (e.g. after reboot) must not wipe older polled data."""
        history = DeviceHistory()
        history.add_reading(self.office, self._reading(18 * 60, 68.0, 40))
        history.add_reading(self.office, self._reading(17 * 60, 68.5, 41))
        partial_ble = [
            self._reading(minutes, 71.0, 42)
            for minutes in range(11 * 60, -1, -60)
        ]
        imported = replace_device_memory_window(history, self.office, partial_ble)
        self.assertEqual(imported, len(partial_ble))
        temps = [r.temp_f for r in history.get_readings(self.office)]
        self.assertIn(68.0, temps)
        self.assertIn(68.5, temps)
        self.assertIn(71.0, temps)

    def test_replace_device_log_window_preserves_other_macs(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            log_path = Path(tmp) / "tp.log"
            rows = [
                [
                    (self.now - timedelta(hours=2)).strftime(LOG_TIMESTAMP_FORMAT),
                    "Office",
                    "70.0",
                    40,
                    self.office,
                ],
                [
                    (self.now - timedelta(hours=1)).strftime(LOG_TIMESTAMP_FORMAT),
                    "Garage",
                    "65.0",
                    30,
                    self.other,
                ],
                [
                    (self.now - timedelta(days=4)).strftime(LOG_TIMESTAMP_FORMAT),
                    "Office",
                    "60.0",
                    35,
                    self.office,
                ],
            ]
            with log_path.open("w", encoding="utf-8", newline="") as handle:
                writer = csv.writer(handle, lineterminator="\n")
                writer.writerow(CSV_HEADER)
                writer.writerows(rows)

            config = AppConfig(
                ini_path=Path(tmp) / "tp.ini",
                settings=Settings(
                    log_directory=tmp,
                    log_file_name="tp.log",
                    logging_enabled=True,
                ),
                devices={self.office: "Office", self.other: "Garage"},
            )
            new_readings = [self._reading(15, 75.0, 45)]
            error = replace_device_log_window(
                config,
                self.office,
                "Office",
                new_readings,
            )
            self.assertIsNone(error)

            with log_path.open("r", encoding="utf-8", newline="") as handle:
                loaded = list(csv.DictReader(handle))
            office_recent = [
                row
                for row in loaded
                if row["mac"] == self.office
                and datetime.strptime(row["timestamp"], LOG_TIMESTAMP_FORMAT) >= self.cutoff
            ]
            other_recent = [row for row in loaded if row["mac"] == self.other]
            old_office = [
                row
                for row in loaded
                if row["mac"] == self.office and row["temp_f"] == "60.0"
            ]
            self.assertEqual(len(office_recent), 2)
            office_temps = {row["temp_f"] for row in office_recent}
            self.assertEqual(office_temps, {"70.0", "75.0"})
            self.assertEqual(len(other_recent), 1)
            self.assertEqual(len(old_office), 1)

    def test_apply_day_history_rejects_small_sample_sets(self) -> None:
        history = DeviceHistory()
        config = AppConfig(
            ini_path=Path("tp.ini"),
            settings=Settings(logging_enabled=False),
            devices={self.office: "Office"},
        )
        readings = [self._reading(i, 70.0, 40) for i in range(10)]
        imported, error = apply_day_history(
            history,
            config,
            self.office,
            "Office",
            readings,
        )
        self.assertEqual(imported, 0)
        self.assertIsNotNone(error)
        self.assertFalse(history.has_data(self.office))


class SparklineBootstrapTests(unittest.TestCase):
    office = "E0:A4:4B:A4:53:0D"

    def test_device_needs_bootstrap_when_empty(self) -> None:
        history = DeviceHistory()
        self.assertTrue(device_needs_sparkline_bootstrap(history, self.office))

    def test_device_skips_bootstrap_when_bins_filled(self) -> None:
        history = DeviceHistory()
        now = datetime.now().replace(minute=0, second=0, microsecond=0)
        for hour in range(SPARKLINE_BOOTSTRAP_MIN_BINS):
            history.add_reading(
                self.office,
                Reading(
                    timestamp=now - timedelta(hours=hour),
                    temp_f=70.0,
                    humidity_pct=40,
                ),
            )
        self.assertFalse(device_needs_sparkline_bootstrap(history, self.office))

    def test_bootstrap_skipped_when_logging_enabled(self) -> None:
        async def run() -> list[str]:
            history = DeviceHistory()
            config = AppConfig(
                ini_path=Path("tp.ini"),
                settings=Settings(logging_enabled=True),
                devices={self.office: "Office"},
            )
            return await bootstrap_sparklines_from_ble(config, history)

        errors = asyncio.run(run())
        self.assertEqual(errors, [])


if __name__ == "__main__":
    unittest.main()
