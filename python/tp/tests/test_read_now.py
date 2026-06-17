"""Unit tests for live (now) reading parsing."""

from __future__ import annotations

import unittest

from tp.ble import _parse_now_buffer
from tp.ui.helpers import format_device_label_row


class NowBufferParserTests(unittest.TestCase):
    def test_parse_positive_temp(self) -> None:
        # c2 00 00 be 00 33 2c -> 19.0 C, 51% RH (Office sensor example)
        reading = _parse_now_buffer([0xC2, 0x00, 0x00, 0xBE, 0x00, 0x33, 0x2C])
        self.assertAlmostEqual(reading.temp_f, 66.2, places=1)
        self.assertEqual(reading.humidity_pct, 51)

    def test_parse_signed_negative_temp(self) -> None:
        # -5.0 C -> 23.0 F; signed little-endian 0xFFCE = -50
        reading = _parse_now_buffer([0xC2, 0x00, 0x00, 0xCE, 0xFF, 0x28, 0x00])
        self.assertAlmostEqual(reading.temp_f, 23.0, places=1)
        self.assertEqual(reading.humidity_pct, 40)

    def test_rejects_short_buffer(self) -> None:
        with self.assertRaises(RuntimeError):
            _parse_now_buffer([0xC2, 0x00, 0x00])


class DeviceLabelRowTests(unittest.TestCase):
    def test_fetch_arrows_cyan_while_connecting(self) -> None:
        row = format_device_label_row("Office", fetching=True, fetch_step="connecting")
        self.assertIn("[bold cyan]▶[/]", row)
        self.assertIn("[bold cyan]◀[/]", row)

    def test_fetch_arrows_green_for_sync(self) -> None:
        row = format_device_label_row("Office", fetching=True, fetch_step="sync")
        self.assertIn("[bold green]▶[/]", row)

    def test_fetch_arrows_yellow_for_passive(self) -> None:
        row = format_device_label_row("Office", fetching=True, fetch_step="passive")
        self.assertIn("[bold yellow]▶[/]", row)

    def test_no_arrows_when_idle(self) -> None:
        row = format_device_label_row("Office", fetching=False)
        self.assertNotIn("▶", row)


if __name__ == "__main__":
    unittest.main()
