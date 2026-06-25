"""Scheduled poll helpers."""

from __future__ import annotations

from datetime import datetime

from tp.ble import INCREMENTAL_HISTORY_MAX_RECORDS
from tp.config import POLL_MODE_INCREMENTAL, POLL_MODE_LIVE
from tp.history import DeviceHistory

INCREMENTAL_DEFAULT_RECORDS = 5


def uses_incremental_history(poll_mode: str) -> bool:
    return poll_mode != POLL_MODE_LIVE


def incremental_history_record_count(
    history: DeviceHistory,
    mac: str,
    *,
    max_records: int = INCREMENTAL_HISTORY_MAX_RECORDS,
) -> int:
    """Minutes of BLE history to request since the last stored reading."""
    now_minute = datetime.now().replace(second=0, microsecond=0)
    last = history.last_updated(mac)
    if last is None:
        return min(INCREMENTAL_DEFAULT_RECORDS, max_records)

    last_minute = last.replace(second=0, microsecond=0)
    gap = int((now_minute - last_minute).total_seconds() // 60)
    if gap < 1:
        return 1
    return min(gap, max_records)
