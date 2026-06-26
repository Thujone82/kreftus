"""Orchestrate BLE day-history fetch and history merge."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from tp.ble import BleWaitDetailCallback, DayHistoryProgress, read_day_history
from tp.config import AppConfig, normalize_mac, resolved_log_path
from tp.history import (
    BLE_HISTORY_MAX_RECORDS,
    SPARKLINE_BOOTSTRAP_HISTORY_HOURS,
    DeviceHistory,
    apply_day_history,
    device_needs_sparkline_bootstrap,
)

HistoryFetchProgressCallback = Callable[[DayHistoryProgress], Awaitable[None] | None]
HistoryBootstrapStartCallback = Callable[
    [str, str, int, int], Awaitable[None] | None
]
StopRequestedCallback = Callable[[], bool]


@dataclass
class DayHistoryResult:
    ok: bool
    imported: int = 0
    sample_count: int = 0
    packet_count: int = 0
    log_rows_written: int = 0
    memory_only: bool = False
    log_replaced: bool = False
    log_path: str | None = None
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
    *,
    record_count: int | None = None,
    ble_wait_detail: BleWaitDetailCallback | None = None,
) -> DayHistoryResult:
    """Fetch BLE history for one device and merge into memory/log."""
    target_mac = normalize_mac(mac)
    packet_count = 0

    await _emit_progress(
        progress_cb,
        DayHistoryProgress(phase="connecting", message="Connecting to sensor…"),
    )

    async def ble_progress(update: DayHistoryProgress) -> None:
        nonlocal packet_count
        if update.packets:
            packet_count = update.packets
        await _emit_progress(progress_cb, update)

    try:
        target_records = (
            record_count if record_count is not None else BLE_HISTORY_MAX_RECORDS
        )
        readings = await read_day_history(
            target_mac,
            progress=ble_progress,
            record_count=target_records,
            wait_detail=ble_wait_detail,
        )
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
    imported, log_rows_written, merge_error = apply_day_history(
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
            log_rows_written=0,
            error=merge_error,
        )

    memory_only = not config.settings.logging_enabled
    log_path = None if memory_only else str(resolved_log_path(config))
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
        log_rows_written=log_rows_written,
        memory_only=memory_only,
        log_replaced=log_rows_written > 0,
        log_path=log_path,
    )


async def bootstrap_sparklines_from_ble(
    config: AppConfig,
    history: DeviceHistory,
    *,
    on_device_start: HistoryBootstrapStartCallback | None = None,
    progress_cb: HistoryFetchProgressCallback | None = None,
    stop_requested: StopRequestedCallback | None = None,
) -> list[str]:
    """Pull BLE history for devices missing sparkline data when logging is off."""
    if config.settings.logging_enabled:
        return []

    targets = [
        (mac, name)
        for mac, name in config.devices.items()
        if device_needs_sparkline_bootstrap(history, mac)
    ]
    if not targets:
        return []

    errors: list[str] = []
    total = len(targets)
    for index, (mac, name) in enumerate(targets):
        if stop_requested and stop_requested():
            break
        if on_device_start is not None:
            maybe = on_device_start(mac, name, index, total)
            if maybe is not None:
                await maybe
        result = await fetch_day_history_for_device(
            config,
            history,
            mac,
            name,
            progress_cb,
            record_count=int(SPARKLINE_BOOTSTRAP_HISTORY_HOURS * 60),
        )
        if not result.ok:
            message = result.error or "History fetch failed"
            errors.append(f"{name}: {message}")
    return errors
