"""History fetch modal status formatting."""

from __future__ import annotations

from datetime import datetime

from tp.ble import DayHistoryProgress, HISTORY_FETCH_CHUNK_RECORDS
from tp.history_fetch import DayHistoryResult
from tp.ui.progress import format_progress_bar


def history_fetch_progress_units(progress: DayHistoryProgress) -> tuple[int, int] | None:
    """Return completed/total units for a history-fetch progress bar."""
    if progress.phase == "done":
        return 1, 1
    if progress.phase == "merging":
        total = max(progress.chunk_total, 1)
        return total, total
    if progress.chunk_total > 0:
        total = progress.chunk_total * 100
        completed = max(0, progress.chunk_index - 1) * 100
        if progress.phase == "receiving" and progress.samples > 0:
            chunk_target = max(HISTORY_FETCH_CHUNK_RECORDS, 1)
            within = min(progress.samples, chunk_target)
            completed += int(within * 100 / chunk_target)
        elif progress.phase in {"connecting", "receiving"} and progress.chunk_index:
            completed = max(completed, (progress.chunk_index - 1) * 100 + 5)
        return min(completed, total), total
    if progress.phase in {"preparing", "waiting", "connecting"}:
        return 0, 100
    return None


def _format_elapsed(started_at: datetime | None) -> str:
    if started_at is None:
        return "—"
    seconds = max(0, int((datetime.now() - started_at).total_seconds()))
    minutes, secs = divmod(seconds, 60)
    if minutes:
        return f"{minutes}m {secs}s"
    return f"{secs}s"


def _phase_footer(progress: DayHistoryProgress) -> str:
    if progress.phase == "waiting":
        return "[dim]Queued behind another BLE operation — please wait…[/]"
    if progress.phase == "connecting":
        return "[dim]Opening BLE connection…[/]"
    if progress.phase == "receiving" and progress.samples == 0 and not progress.bytes_received:
        return (
            "[dim]Sensor is preparing history — large exports can take several "
            "minutes before the first samples appear.[/]"
        )
    if progress.phase == "receiving":
        return "[dim]BLE read in progress — please wait…[/]"
    if progress.phase == "merging":
        return "[dim]Merging readings into memory and log…[/]"
    return "[dim]Working…[/]"


def format_history_fetch_status(
    device_name: str,
    mac: str,
    *,
    progress: DayHistoryProgress | None = None,
    result: DayHistoryResult | None = None,
    started_at: datetime | None = None,
    error: str | None = None,
) -> str:
    """Build Rich markup for the history fetch modal body."""
    lines = [
        f"[bold yellow]History Fetch — {device_name}[/]",
        f"[dim]{mac}[/]",
        "",
    ]

    if progress is not None:
        phase = progress.phase.replace("_", " ").title()
        lines.append(f"[bold]Phase:[/] {phase}")
        if progress.chunk_total > 1 and progress.chunk_index:
            lines.append(
                f"  Chunk: [white]{progress.chunk_index}[/] / "
                f"[white]{progress.chunk_total}[/]"
            )
        if progress.message:
            lines.append(f"  {progress.message}")
        progress_units = history_fetch_progress_units(progress)
        if progress_units is not None:
            completed, total = progress_units
            lines.append(
                "  Progress: "
                + format_progress_bar(completed, total, width=28)
            )
        if (
            progress.phase == "receiving"
            or progress.packets
            or progress.samples
            or progress.bytes_received
        ):
            detail_parts = [
                f"Packets: [white]{progress.packets}[/]",
                f"Samples: [white]{progress.samples:,}[/]",
            ]
            if progress.bytes_received:
                detail_parts.append(f"Bytes: [white]{progress.bytes_received:,}[/]")
            lines.append(f"  {'  '.join(detail_parts)}")
        lines.append(f"  Elapsed: [dim]{_format_elapsed(started_at)}[/]")
        lines.append("")
        lines.append(_phase_footer(progress))

    if result is not None:
        if result.ok:
            lines.append("[green]Import complete[/]")
            lines.append(f"  Samples received: [white]{result.sample_count:,}[/]")
            lines.append(f"  Samples merged: [white]{result.imported:,}[/]")
            if result.memory_only:
                lines.append("  Log: [dim]unchanged (logging disabled)[/]")
            elif result.log_replaced:
                lines.append(
                    "  Log: [green]"
                    f"{result.log_rows_written:,} row(s) written for this device "
                    "in the received span[/]"
                )
                if result.log_path:
                    lines.append(f"  [dim]File: {result.log_path}[/]")
            else:
                lines.append("  Log: [yellow]unchanged[/]")
            lines.append("")
            lines.append("[dim]Return to monitoring to see updated sparklines.[/]")
        else:
            lines.append("[red]Import failed[/]")
            if result.error:
                lines.append(f"  [red]{result.error}[/]")
            if result.sample_count:
                lines.append(f"  Samples parsed: [white]{result.sample_count:,}[/]")

    if error and result is None:
        lines.append("[red]Import failed[/]")
        lines.append(f"  [red]{error}[/]")

    if progress is None and result is None and error is None:
        lines.append("[dim]Preparing history fetch…[/]")
        lines.append("")
        lines.append("[dim]Checking BLE availability…[/]")

    return "\n".join(lines)
