"""Unit tests for Bluetooth radio recovery helpers."""

from __future__ import annotations

import unittest
from unittest.mock import AsyncMock, patch

from tp.ble_radio import (
    BT_DISABLED_REQUEST,
    BluetoothPermissionRequest,
    ensure_bluetooth_enabled_for_polling,
    is_bluetooth_powered_off_error,
    maybe_restart_bluetooth_radio,
    maybe_restart_bluetooth_radio_after_total_failure,
    set_bluetooth_permission_callback,
)
import tp.ble_radio as ble_radio


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


class BluetoothPermissionTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        ble_radio._last_radio_restart_at = None
        ble_radio._last_permission_denied_at = None

    def tearDown(self) -> None:
        ble_radio._last_radio_restart_at = None
        ble_radio._last_permission_denied_at = None
        set_bluetooth_permission_callback(None)

    async def test_enable_requires_permission(self) -> None:
        with patch(
            "tp.ble_radio.is_bluetooth_radio_disabled",
            new=AsyncMock(return_value=True),
        ):
            self.assertFalse(await ensure_bluetooth_enabled_for_polling())

    async def test_enable_after_approval(self) -> None:
        set_bluetooth_permission_callback(lambda _request: True)
        with (
            patch(
                "tp.ble_radio.is_bluetooth_radio_disabled",
                new=AsyncMock(return_value=True),
            ),
            patch(
                "tp.ble_radio.enable_bluetooth_radio",
                new=AsyncMock(return_value=True),
            ) as enable_mock,
        ):
            self.assertTrue(await ensure_bluetooth_enabled_for_polling())
            enable_mock.assert_awaited_once()

    async def test_powered_off_error_waits_for_permission(self) -> None:
        approved: list[BluetoothPermissionRequest] = []

        def _approve(request: BluetoothPermissionRequest) -> bool:
            approved.append(request)
            return True

        set_bluetooth_permission_callback(_approve)
        with (
            patch(
                "tp.ble_radio.is_bluetooth_radio_disabled",
                new=AsyncMock(return_value=True),
            ),
            patch(
                "tp.ble_radio.enable_bluetooth_radio",
                new=AsyncMock(return_value=True),
            ),
        ):
            exc = RuntimeError("Bluetooth radio is not powered on. Turn on Bluetooth.")
            self.assertTrue(await maybe_restart_bluetooth_radio(exc))
        self.assertEqual(approved, [BT_DISABLED_REQUEST])

    async def test_powered_off_error_skips_prompt_when_radio_on(self) -> None:
        approved: list[BluetoothPermissionRequest] = []

        def _approve(request: BluetoothPermissionRequest) -> bool:
            approved.append(request)
            return True

        set_bluetooth_permission_callback(_approve)
        with (
            patch(
                "tp.ble_radio.is_bluetooth_radio_disabled",
                new=AsyncMock(return_value=False),
            ),
            patch(
                "tp.ble_radio.enable_bluetooth_radio",
                new=AsyncMock(return_value=True),
            ) as enable_mock,
        ):
            exc = RuntimeError("Bluetooth radio is not powered on. Turn on Bluetooth.")
            self.assertFalse(await maybe_restart_bluetooth_radio(exc))
            enable_mock.assert_not_called()
        self.assertEqual(approved, [])

    async def test_total_failure_restart_without_permission(self) -> None:
        with patch(
            "tp.ble_radio.restart_bluetooth_radio",
            new=AsyncMock(return_value=True),
        ) as restart_mock:
            self.assertTrue(await maybe_restart_bluetooth_radio_after_total_failure())
            restart_mock.assert_awaited_once()


if __name__ == "__main__":
    unittest.main()
