"""Orchestrate BLE day-history fetch and history merge."""

from __future__ import annotations

from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from tp.ble import DayHistoryProgress, read_day_history
from tp.config import AppConfig, normalize_mac
from tp.history import DeviceHistory, apply_day_history, device_needs_sparkline_bootstrap

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
    """Fetch 72H BLE history for one device and merge into memory/log."""
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


async def bootstrap_sparklines_from_ble(
    config: AppConfig,
    history: DeviceHistory,
    *,
    on_device_start: HistoryBootstrapStartCallback | None = None,
    progress_cb: HistoryFetchProgressCallback | None = None,
    stop_requested: StopRequestedCallback | None = None,
) -> list[str]:
    """Pull 72H BLE history for devices missing sparkline data when logging is off."""
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
        )
        if not result.ok:
            message = result.error or "72H history fetch failed"
            errors.append(f"{name}: {message}")
    return errors
