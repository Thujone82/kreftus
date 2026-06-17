"""Unit tests for BLE device resolution cache."""

from __future__ import annotations

import unittest
from unittest.mock import MagicMock

from tp.ble import (
    RESOLVE_CACHE_TTL,
    _connect_strategies_for,
    _device_cache,
    _invalidate_ble_device_cache,
    _remember_ble_device,
    get_cached_ble_device,
)


class BleDeviceCacheTests(unittest.TestCase):
    def setUp(self) -> None:
        _device_cache.clear()

    def test_remember_and_hit_cache(self) -> None:
        device = MagicMock()
        device.address = "E0:A4:4B:A4:53:0D"
        _remember_ble_device("e0:a4:4b:a4:53:0d", device, strategy="device-cached")
        cached = get_cached_ble_device("E0:A4:4B:A4:53:0D")
        self.assertIs(cached, device)

    def test_cache_expires(self) -> None:
        device = MagicMock()
        device.address = "E0:A4:4B:A4:53:0D"
        _remember_ble_device("e0:a4:4b:a4:53:0d", device)
        entry = _device_cache["E0:A4:4B:A4:53:0D"]
        entry.cached_at -= RESOLVE_CACHE_TTL + 1
        self.assertIsNone(get_cached_ble_device("E0:A4:4B:A4:53:0D"))

    def test_invalidate_cache(self) -> None:
        device = MagicMock()
        device.address = "E0:A4:4B:A4:53:0D"
        _remember_ble_device("e0:a4:4b:a4:53:0d", device)
        _invalidate_ble_device_cache("e0:a4:4b:a4:53:0d")
        self.assertIsNone(get_cached_ble_device("E0:A4:4B:A4:53:0D"))

    def test_preferred_strategy_moves_to_front(self) -> None:
        device = MagicMock()
        device.address = "E0:A4:4B:A4:53:0D"
        _remember_ble_device("e0:a4:4b:a4:53:0d", device, strategy="device-cached")
        strategies = _connect_strategies_for("e0:a4:4b:a4:53:0d")
        self.assertEqual(strategies[0].name, "device-cached")


if __name__ == "__main__":
    unittest.main()
