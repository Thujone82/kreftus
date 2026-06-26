"""Modal progress UI for CSV log export."""

from __future__ import annotations

import asyncio
import webbrowser

from textual import work
from textual.app import ComposeResult
from textual.containers import Vertical, VerticalScroll
from textual.screen import ModalScreen
from textual.widgets import Static

from tp.log_export import (
    build_export_payload,
    load_log_rows_for_export,
    render_log_export_html,
)
from tp.ui.progress import format_progress_bar


class LogExportModal(ModalScreen[None]):
    """Export log to HTML on a background thread with progress feedback."""

    DEFAULT_CSS = """
    LogExportModal {
        align: center middle;
    }
    #log-export-dialog {
        width: 72;
        height: auto;
        max-height: 90%;
        border: thick $primary;
        background: $surface;
        padding: 1 2;
    }
    """

    def __init__(self) -> None:
        super().__init__()
        self._active = True
        self._message = "Starting log export…"
        self._progress_current = 0
        self._progress_total = 100
        self._error: str | None = None
        self._output_name: str | None = None

    def compose(self) -> ComposeResult:
        with Vertical(id="log-export-dialog"):
            yield VerticalScroll(Static("", id="log-export-body"), id="log-export-scroll")
            yield Static("[dim]Please wait…[/]", id="log-export-hint")

    def on_mount(self) -> None:
        self.app.log_export_in_progress = True
        self._refresh_body()
        self._run_export()

    def on_unmount(self) -> None:
        self.app.log_export_in_progress = False

    def _set_progress(self, message: str, current: int, total: int = 100) -> None:
        self._message = message
        self._progress_current = current
        self._progress_total = total
        if self.is_mounted:
            self._refresh_body()

    def _refresh_body(self) -> None:
        if not self.is_mounted:
            return
        lines = [
            "[bold yellow]Export Log to Web[/]",
            "",
        ]
        if self._error:
            lines.extend(
                [
                    "[red]Export failed[/]",
                    f"  [red]{self._error}[/]",
                ]
            )
        elif not self._active and self._output_name:
            lines.extend(
                [
                    "[green]Export complete[/]",
                    f"  Opened [white]{self._output_name}[/] in your browser.",
                ]
            )
        else:
            lines.append(f"[bold]{self._message}[/]")
            lines.append("")
            lines.append(
                "  "
                + format_progress_bar(
                    self._progress_current,
                    self._progress_total,
                    width=28,
                )
            )
            lines.append("")
            lines.append(
                "[dim]Large logs can take a minute — please wait…[/]"
            )
        self.query_one("#log-export-body", Static).update("\n".join(lines))
        hint = self.query_one("#log-export-hint", Static)
        if self._active:
            hint.update("[dim]Export in progress — please wait…[/]")
        else:
            hint.update("[dim]Q to close[/]")

    def check_action(self, action: str, parameters: tuple[object, ...]) -> bool | None:
        if self._active and action == "quit_or_back":
            self.app.notify("Export in progress — please wait…", severity="warning", timeout=3)
            return False
        return True

    def action_quit_or_back(self) -> None:
        if not self._active:
            self.dismiss(None)

    @work
    async def _run_export(self) -> None:
        config = self.app.config
        try:
            self._set_progress("Reading CSV log…", 5)
            rows = await asyncio.to_thread(load_log_rows_for_export, config)
            if not rows:
                self._error = "Log file has no readings for managed devices."
                return

            self._set_progress("Building chart data…", 35)
            payload = await asyncio.to_thread(build_export_payload, config, rows)
            if not any(payload["series"].values()):
                self._error = "No exportable readings after filtering."
                return

            self._set_progress("Rendering HTML report…", 65)
            html = await asyncio.to_thread(render_log_export_html, payload)

            self._set_progress("Writing tp_export.html…", 85)
            output_path, write_error = await asyncio.to_thread(
                _write_export_html,
                config,
                html,
            )
            if write_error:
                self._error = write_error
                return

            self._set_progress("Opening in browser…", 100)
            assert output_path is not None
            try:
                await asyncio.to_thread(webbrowser.open, output_path.as_uri())
            except OSError:
                pass
            self._output_name = output_path.name
        except Exception as exc:  # noqa: BLE001
            self._error = str(exc) or exc.__class__.__name__
        finally:
            self._active = False
            if self.is_mounted:
                self._refresh_body()


def _write_export_html(config, html: str):  # noqa: ANN001
    """Write export file; returns (path, error) like export_log_to_html tail."""
    from tp.log_export import default_export_path

    _ = config
    output_path = default_export_path()
    try:
        output_path.write_text(html, encoding="utf-8")
    except OSError as exc:
        return None, f"Cannot write {output_path}: {exc}"
    return output_path, None
