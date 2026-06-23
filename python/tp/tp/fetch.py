"""Fetch cycle orchestration."""

from __future__ import annotations

import asyncio
from datetime import datetime
from typing import Awaitable, Callable

from tp.ble import (
    DEVICE_READ_TIMEOUT,
    NOW_READ_CONNECTING,
    clear_ble_device_cache,
    format_ble_error,
    inter_device_delay_seconds,
    prefetch_ble_device,
    read_now,
)
from tp.ble_radio import (
    ensure_bluetooth_enabled_for_polling,
    maybe_restart_bluetooth_radio_after_total_failure,
)
from tp.config import AppConfig
from tp.debug_log import write as debug_write
from tp.debug_log import write_exception as debug_write_exception
from tp.history import DeviceHistory, PollResult, Reading, append_poll_results_to_log

ProgressCallback = Callable[[int, int, str, str], Awaitable[None] | None]
NowReadPhaseCallback = Callable[[str, str, str], Awaitable[None] | None]
ResultCallback = Callable[[PollResult], Awaitable[None] | None]

START_MARKER = "__start__"


def _whole_fleet_cycle_failed(
    config: AppConfig,
    *,
    only_macs: frozenset[str] | None,
    total: int,
    ok: int,
) -> bool:
    """True when every device in the cycle failed and the cycle covered the full fleet."""
    if ok > 0 or total == 0:
        return False
    fleet_size = len(config.devices)
    if fleet_size == 0:
        return False
    if only_macs is None:
        return total == fleet_size
    return total == fleet_size and len(only_macs) == fleet_size


async def _fetch_one_device(
    mac: str,
    name: str,
    *,
    config: AppConfig,
    on_now_phase: NowReadPhaseCallback | None = None,
) -> PollResult:
    debug_write(f"fetch: reading {name} ({mac})", config=config)

    async def phase_cb(phase: str) -> None:
        if on_now_phase is None:
            return
        maybe = on_now_phase(mac, name, phase)
        if asyncio.iscoroutine(maybe):
            await maybe

    try:
        now = await asyncio.wait_for(
            read_now(mac, progress=phase_cb),
            timeout=DEVICE_READ_TIMEOUT,
        )
        reading = Reading(
            timestamp=datetime.now(),
            temp_f=now.temp_f,
            humidity_pct=now.humidity_pct,
        )
        debug_write(
            f"fetch: ok {name} temp={reading.temp_f}F humidity={reading.humidity_pct}%",
            config=config,
        )
        return PollResult(mac=mac, device_name=name, reading=reading)
    except asyncio.CancelledError:
        raise
    except TimeoutError:
        debug_write(
            f"fetch: timed out {name} ({mac}) after {DEVICE_READ_TIMEOUT:.0f}s",
            config=config,
        )
        return PollResult(
            mac=mac,
            device_name=name,
            reading=None,
            error=f"Read timed out ({int(DEVICE_READ_TIMEOUT)}s)",
        )
    except Exception as exc:  # noqa: BLE001
        debug_write_exception(f"fetch: failed {name} ({mac})", exc, config=config)
        return PollResult(
            mac=mac,
            device_name=name,
            reading=None,
            error=format_ble_error(exc),
        )


async def run_fetch_cycle(
    config: AppConfig,
    history: DeviceHistory,
    *,
    only_macs: frozenset[str] | None = None,
    progress: ProgressCallback | None = None,
    on_now_phase: NowReadPhaseCallback | None = None,
    on_result: ResultCallback | None = None,
    had_prior_success: bool = False,
    allow_radio_recovery: bool = True,
) -> tuple[list[PollResult], list[str]]:
    """Collect live readings from managed devices (all or a subset)."""
    batch, errors = await _run_fetch_cycle_once(
        config,
        history,
        only_macs=only_macs,
        progress=progress,
        on_now_phase=on_now_phase,
        on_result=on_result,
    )

    ok = sum(1 for result in batch if result.reading is not None)
    total = len(batch)
    if (
        allow_radio_recovery
        and had_prior_success
        and _whole_fleet_cycle_failed(config, only_macs=only_macs, total=total, ok=ok)
    ):
        debug_write(
            f"fetch: entire fleet failed ({ok}/{total}); attempting Bluetooth radio restart",
            config=config,
        )
        if await maybe_restart_bluetooth_radio_after_total_failure():
            clear_ble_device_cache()
            batch, errors = await _run_fetch_cycle_once(
                config,
                history,
                only_macs=only_macs,
                progress=progress,
                on_now_phase=on_now_phase,
                on_result=on_result,
            )
            retry_ok = sum(1 for result in batch if result.reading is not None)
            debug_write(
                f"fetch: post-restart cycle ({retry_ok}/{len(batch)} ok)",
                config=config,
            )

    return batch, errors


async def _run_fetch_cycle_once(
    config: AppConfig,
    history: DeviceHistory,
    *,
    only_macs: frozenset[str] | None = None,
    progress: ProgressCallback | None = None,
    on_now_phase: NowReadPhaseCallback | None = None,
    on_result: ResultCallback | None = None,
) -> tuple[list[PollResult], list[str]]:
    """Single pass over managed devices (no fleet-failure radio recovery)."""
    devices = [
        (mac, name)
        for mac, name in config.devices.items()
        if only_macs is None or mac in only_macs
    ]
    total = len(devices)
    errors: list[str] = []
    batch: list[PollResult] = []

    if total == 0:
        debug_write("fetch: cycle skipped (no devices)", config=config)
        return [], errors

    scope = "all" if only_macs is None else f"{len(only_macs)} selected"
    debug_write(f"fetch: cycle start ({total} device(s), {scope})", config=config)

    if not await ensure_bluetooth_enabled_for_polling():
        message = "Bluetooth is off — enable Bluetooth to poll sensors"
        debug_write(f"fetch: cycle aborted ({message})", config=config)
        return [
            PollResult(mac=mac, device_name=name, reading=None, error=message)
            for mac, name in devices
        ], [message]

    if progress:
        maybe = progress(0, total, START_MARKER, "")
        if asyncio.iscoroutine(maybe):
            await maybe

    for index, (mac, name) in enumerate(devices, start=1):
        if progress:
            maybe = progress(index - 1, total, name, mac)
            if asyncio.iscoroutine(maybe):
                await maybe
        if on_now_phase:
            maybe = on_now_phase(mac, name, NOW_READ_CONNECTING)
            if asyncio.iscoroutine(maybe):
                await maybe
        result = await _fetch_one_device(
            mac,
            name,
            config=config,
            on_now_phase=on_now_phase,
        )
        history.record_fetch_result(result)
        if result.reading is not None:
            history.add_reading(result.mac, result.reading)
        batch.append(result)
        if result.error:
            errors.append(f"{name}: {result.error}")
        if on_result:
            maybe = on_result(result)
            if asyncio.iscoroutine(maybe):
                await maybe
        if index < total:
            next_mac, next_name = devices[index]
            debug_write(
                f"fetch: prefetching {next_name} ({next_mac}) during inter-device delay",
                config=config,
            )
            prefetch_task = asyncio.create_task(prefetch_ble_device(next_mac))
            await asyncio.sleep(inter_device_delay_seconds())
            await prefetch_task

    if progress:
        maybe = progress(total, total, "Saving results", "")
        if asyncio.iscoroutine(maybe):
            await maybe

    log_error = append_poll_results_to_log(config, batch)
    if log_error:
        errors.append(log_error)
        debug_write(f"fetch: log append error: {log_error}", config=config)

    ok = sum(1 for r in batch if r.reading is not None)
    debug_write(f"fetch: cycle done ({ok}/{total} ok, {len(errors)} error(s))", config=config)
    return batch, errors
