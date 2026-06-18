"""Unit tests for Bluetooth radio recovery helpers."""

from __future__ import annotations

import unittest

from tp.ble_radio import is_bluetooth_powered_off_error


class BluetoothPoweredOffDetectionTests(unittest.TestCase):
    def test_detects_bleak_powered_off_reason(self) -> None:
        try:
            from bleak.exc import (
                BleakBluetoothNotAvailableError,
                BleakBluetoothNotAvailableReason,
            )
        except ImportError:
            self.skipTest("bleak not installed")

        exc = BleakBluetoothNotAvailableError(
            "Bluetooth radio is not powered on. Turn on Bluetooth and try again.",
            BleakBluetoothNotAvailableReason.POWERED_OFF,
        )
        self.assertTrue(is_bluetooth_powered_off_error(exc))

    def test_detects_message_text(self) -> None:
        self.assertTrue(
            is_bluetooth_powered_off_error(
                RuntimeError("Bluetooth radio is not powered on. Turn on Bluetooth.")
            )
        )

    def test_ignores_unrelated_errors(self) -> None:
        self.assertFalse(is_bluetooth_powered_off_error(RuntimeError("Device was not found")))


if __name__ == "__main__":
    unittest.main()
