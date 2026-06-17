"""Orchestrate BLE day-history fetch and history merge."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from tp.ble import DayHistoryProgress, read_day_history
from tp.config import AppConfig, normalize_mac
from tp.history import DeviceHistory, apply_day_history

HistoryFetchProgressCallback = Callable[[DayHistoryProgress], Awaitable[None] | None]


@dataclass
class DayHistoryResult:
    ok: bool
    imported: int = 0
    sample_count: int = 0
    packet_count: int = 0
    memory_only: bool = False
    log_replaced: bool = False
    error: str | None = None


async def _emit_progress(
    progress_cb: HistoryFetchProgressCallback | None,
    update: DayHistoryProgress,
) -> None:
    if progress_cb is None:
        return
    maybe = progress_cb(update)
    if maybe is not None:
        await maybe


async def fetch_day_history_for_device(
    config: AppConfig,
    history: DeviceHistory,
    mac: str,
    device_name: str,
    progress_cb: HistoryFetchProgressCallback | None = None,
) -> DayHistoryResult:
    """Fetch 24H BLE history for one device and merge into memory/log."""
    target_mac = normalize_mac(mac)
    packet_count = 0

    async def ble_progress(update: DayHistoryProgress) -> None:
        nonlocal packet_count
        if update.packets:
            packet_count = update.packets
        await _emit_progress(progress_cb, update)

    try:
        readings = await read_day_history(target_mac, progress=ble_progress)
    except Exception as exc:  # noqa: BLE001
        message = str(exc) or exc.__class__.__name__
        await _emit_progress(
            progress_cb,
            DayHistoryProgress(phase="error", message=message),
        )
        return DayHistoryResult(ok=False, error=message, packet_count=packet_count)

    await _emit_progress(
        progress_cb,
        DayHistoryProgress(
            phase="merging",
            packets=packet_count,
            samples=len(readings),
            message="Merging into history…",
        ),
    )
    imported, merge_error = apply_day_history(
        history,
        config,
        target_mac,
        device_name,
        readings,
    )
    if merge_error:
        await _emit_progress(
            progress_cb,
            DayHistoryProgress(phase="error", message=merge_error),
        )
        return DayHistoryResult(
            ok=False,
            imported=0,
            sample_count=len(readings),
            packet_count=packet_count,
            error=merge_error,
        )

    memory_only = not config.settings.logging_enabled
    await _emit_progress(
        progress_cb,
        DayHistoryProgress(
            phase="done",
            packets=packet_count,
            samples=len(readings),
            message="Complete",
        ),
    )
    return DayHistoryResult(
        ok=True,
        imported=imported,
        sample_count=len(readings),
        packet_count=packet_count,
        memory_only=memory_only,
        log_replaced=not memory_only,
    )
