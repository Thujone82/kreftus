"""TUI action for log export to web."""

from __future__ import annotations

from tp.log_export import can_export_log
from tp.ui.log_export_modal import LogExportModal


def export_log_to_web(app) -> None:
    """Open export progress modal and write tp_export.html on a worker thread."""
    if getattr(app, "log_export_in_progress", False):
        app.notify("Log export already in progress…", severity="warning", timeout=4)
        return
    if not can_export_log(app.config):
        app.notify("No log data available to export.", severity="warning", timeout=6)
        return
    app.push_screen(LogExportModal())
