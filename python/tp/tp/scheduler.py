"""Five-minute grid-aligned scheduling."""

from __future__ import annotations

import asyncio
from datetime import datetime, timedelta

POLL_INTERVAL = timedelta(minutes=5)
STALE_AFTER = timedelta(minutes=5)
RETRY_INTERVAL = timedelta(minutes=1)


def floor_to_boundary(from_time: datetime | None = None) -> datetime:
    """Largest 5-minute grid time (:00, :05, …) not after from_time."""
    now = from_time or datetime.now()
    floored_minute = (now.minute // 5) * 5
    return now.replace(minute=floored_minute, second=0, microsecond=0)


def chunk_end(chunk_start: datetime) -> datetime:
    return chunk_start + POLL_INTERVAL


def is_measurement_stale(
    last_updated: datetime | None,
    from_time: datetime | None = None,
) -> bool:
    """True when a device has no reading newer than STALE_AFTER."""
    if last_updated is None:
        return True
    now = from_time or datetime.now()
    return now - last_updated > STALE_AFTER


def updated_in_chunk(last_updated: datetime | None, chunk_start: datetime) -> bool:
    if last_updated is None:
        return False
    return last_updated >= chunk_start


def stale_macs_for_chunk(
    devices: dict[str, str],
    last_updated_fn,
    chunk_start: datetime,
    from_time: datetime | None = None,
) -> frozenset[str]:
    """Devices that have not received a successful update this 5-minute chunk."""
    _ = from_time  # reserved for tests
    stale: set[str] = set()
    for mac in devices:
        if not updated_in_chunk(last_updated_fn(mac), chunk_start):
            stale.add(mac)
    return frozenset(stale)


def next_retry_time(
    *,
    chunk_start: datetime,
    last_retry_at: datetime | None,
    from_time: datetime | None = None,
) -> datetime | None:
    """Next 1-minute retry instant, or None if the chunk boundary comes first."""
    now = from_time or datetime.now()
    boundary = chunk_end(chunk_start)
    candidate = (
        (last_retry_at + RETRY_INTERVAL)
        if last_retry_at is not None
        else now + RETRY_INTERVAL
    )
    if candidate >= boundary:
        return None
    return candidate


def next_five_minute_boundary(from_time: datetime | None = None) -> datetime:
    """Return the next scheduled poll time strictly after from_time."""
    return schedule_next_poll_after(from_time or datetime.now())


def schedule_next_poll_after(from_time: datetime) -> datetime:
    """Next grid boundary after a poll completes (or for initial scheduling)."""
    return floor_to_boundary(from_time) + timedelta(minutes=5)


def seconds_until(from_time: datetime, target: datetime) -> float:
    return max(0.0, (target - from_time).total_seconds())


def seconds_until_next_boundary(from_time: datetime | None = None) -> float:
    """Seconds until the next poll time after from_time (display helper)."""
    now = from_time or datetime.now()
    return seconds_until(now, next_five_minute_boundary(now))


async def wait_for_next_boundary(from_time: datetime | None = None) -> datetime:
    """Sleep until the next 5-minute boundary and return that timestamp."""
    now = from_time or datetime.now()
    target = next_five_minute_boundary(now)
    delay = seconds_until(now, target)
    if delay > 0:
        await asyncio.sleep(delay)
    return target
