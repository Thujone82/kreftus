"""72H history fetch modal status formatting."""

from __future__ import annotations

from datetime import datetime

from tp.ble import DayHistoryProgress
from tp.history_fetch import DayHistoryResult


def _format_elapsed(started_at: datetime | None) -> str:
    if started_at is None:
        return "—"
    seconds = max(0, int((datetime.now() - started_at).total_seconds()))
    minutes, secs = divmod(seconds, 60)
    if minutes:
        return f"{minutes}m {secs}s"
    return f"{secs}s"


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
        f"[bold yellow]72H History — {device_name}[/]",
        f"[dim]{mac}[/]",
        "",
    ]

    if progress is not None:
        phase = progress.phase.replace("_", " ").title()
        lines.append(f"[bold]Phase:[/] {phase}")
        if progress.message:
            lines.append(f"  {progress.message}")
        if progress.packets or progress.samples:
            lines.append(
                f"  Packets: [white]{progress.packets}[/]  "
                f"Samples: [white]{progress.samples}[/]"
            )
        lines.append(f"  Elapsed: [dim]{_format_elapsed(started_at)}[/]")
        lines.append("")
        lines.append("[dim]BLE read in progress — please wait…[/]")

    if result is not None:
        if result.ok:
            lines.append("[green]Import complete[/]")
            lines.append(f"  Samples received: [white]{result.sample_count}[/]")
            lines.append(f"  Samples in 72H window: [white]{result.imported}[/]")
            if result.memory_only:
                lines.append("  Log: [dim]unchanged (logging disabled)[/]")
            else:
                lines.append("  Log: [green]last 72H rows replaced for this device[/]")
            lines.append("")
            lines.append("[dim]Return to monitoring to see updated sparklines.[/]")
        else:
            lines.append("[red]Import failed[/]")
            if result.error:
                lines.append(f"  [red]{result.error}[/]")
            if result.sample_count:
                lines.append(f"  Samples parsed: [white]{result.sample_count}[/]")

    if error and result is None:
        lines.append("[red]Import failed[/]")
        lines.append(f"  [red]{error}[/]")

    if progress is None and result is None and error is None:
        lines.append("[dim]Starting…[/]")

    return "\n".join(lines)
