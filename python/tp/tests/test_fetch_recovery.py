"""Unit tests for fetch recovery helpers and fail-streak tracking."""

from __future__ import annotations

import unittest
from datetime import datetime
from unittest.mock import AsyncMock, patch

from tp.config import AppConfig, Settings
from tp.fetch import (
    DEVICE_FAIL_STREAK_RADIO,
    _merge_poll_batches,
    _stuck_device_macs,
    _whole_fleet_cycle_failed,
    run_fetch_cycle,
)
from tp.history import DeviceHistory, PollResult, Reading


def _config_with_devices(*pairs: tuple[str, str]) -> AppConfig:
    devices = {mac: name for mac, name in pairs}
    return AppConfig(settings=Settings(), devices=devices)


class FetchRecoveryHelperTests(unittest.TestCase):
    def test_whole_fleet_failed_requires_full_coverage(self) -> None:
        config = _config_with_devices(("AA", "A"), ("BB", "B"))
        self.assertTrue(
            _whole_fleet_cycle_failed(config, only_macs=None, total=2, ok=0)
        )
        self.assertFalse(
            _whole_fleet_cycle_failed(config, only_macs=None, total=2, ok=1)
        )
        self.assertFalse(
            _whole_fleet_cycle_failed(
                config,
                only_macs=frozenset({"AA"}),
                total=1,
                ok=0,
            )
        )

    def test_merge_poll_batches_replaces_by_mac(self) -> None:
        previous = [
            PollResult(mac="AA", device_name="A", reading=None, error="old"),
            PollResult(mac="BB", device_name="B", reading=None, error="old"),
        ]
        updated = [
            PollResult(
                mac="AA",
                device_name="A",
                reading=Reading(timestamp=datetime.now(), temp_f=70.0, humidity_pct=40),
            )
        ]
        merged = _merge_poll_batches(previous, updated)
        self.assertEqual(len(merged), 2)
        self.assertIsNotNone(merged[0].reading)
        self.assertEqual(merged[1].error, "old")


class FailStreakTests(unittest.TestCase):
    def test_fail_streak_increments_and_resets(self) -> None:
        history = DeviceHistory()
        mac = "AA:BB:CC:DD:EE:FF"
        history.record_fetch_result(
            PollResult(mac=mac, device_name="Garage", reading=None, error="timeout")
        )
        history.record_fetch_result(
            PollResult(mac=mac, device_name="Garage", reading=None, error="timeout")
        )
        self.assertEqual(history.fetch_status(mac).fail_streak, 2)
        self.assertEqual(history.macs_with_fail_streak(2), [mac])

        history.record_fetch_result(
            PollResult(
                mac=mac,
                device_name="Garage",
                reading=Reading(timestamp=datetime.now(), temp_f=64.0, humidity_pct=55),
            )
        )
        self.assertEqual(history.fetch_status(mac).fail_streak, 0)
        self.assertEqual(history.macs_with_fail_streak(1), [])

    def test_stuck_device_macs_respects_only_macs(self) -> None:
        history = DeviceHistory()
        history.record_fetch_result(
            PollResult(mac="AA", device_name="A", reading=None, error="x")
        )
        history.record_fetch_result(
            PollResult(mac="AA", device_name="A", reading=None, error="x")
        )
        history.record_fetch_result(
            PollResult(mac="BB", device_name="B", reading=None, error="x")
        )
        history.record_fetch_result(
            PollResult(mac="BB", device_name="B", reading=None, error="x")
        )
        stuck = _stuck_device_macs(
            history,
            min_streak=DEVICE_FAIL_STREAK_RADIO,
            only_macs=frozenset({"AA"}),
        )
        self.assertEqual(stuck, frozenset({"AA"}))


class StuckDeviceRecoveryTests(unittest.IsolatedAsyncioTestCase):
    async def test_stuck_device_triggers_radio_restart_and_retry(self) -> None:
        config = _config_with_devices(("AA", "Garage"), ("BB", "Office"))
        history = DeviceHistory()
        # Pre-seed two Garage failures so the first cycle looks "stuck".
        for _ in range(2):
            history.record_fetch_result(
                PollResult(mac="AA", device_name="Garage", reading=None, error="timeout")
            )
        self.assertEqual(history.fetch_status("AA").fail_streak, 2)

        fail = PollResult(mac="AA", device_name="Garage", reading=None, error="timeout")
        ok_office = PollResult(
            mac="BB",
            device_name="Office",
            reading=Reading(timestamp=datetime.now(), temp_f=70.0, humidity_pct=40),
        )
        recovered = PollResult(
            mac="AA",
            device_name="Garage",
            reading=Reading(timestamp=datetime.now(), temp_f=64.6, humidity_pct=55),
        )

        with (
            patch(
                "tp.fetch._run_fetch_cycle_once",
                new=AsyncMock(side_effect=[([fail, ok_office], ["Garage: timeout"]), ([recovered], [])]),
            ) as cycle_mock,
            patch(
                "tp.fetch.maybe_restart_bluetooth_radio_after_total_failure",
                new=AsyncMock(return_value=True),
            ) as radio_mock,
            patch("tp.fetch.clear_ble_device_cache") as clear_mock,
            patch(
                "tp.fetch.maybe_reset_bluetooth_stack_after_radio_failure",
                new=AsyncMock(return_value=False),
            ) as stack_mock,
        ):
            batch, errors = await run_fetch_cycle(
                config,
                history,
                had_prior_success=True,
            )

        radio_mock.assert_awaited_once()
        clear_mock.assert_called()
        stack_mock.assert_not_awaited()
        self.assertEqual(cycle_mock.await_count, 2)
        garage = next(result for result in batch if result.mac == "AA")
        self.assertIsNotNone(garage.reading)
        self.assertEqual(errors, [])
