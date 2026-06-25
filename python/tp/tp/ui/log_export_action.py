"""TUI action for log export to web."""

from __future__ import annotations

import webbrowser

from tp.log_export import export_log_to_html


def export_log_to_web(app) -> None:
    """Write tp_export.html beside the launcher and open it in the browser."""
    output_path, error = export_log_to_html(app.config)
    if error:
        app.notify(error, severity="error", timeout=8)
        return
    assert output_path is not None
    try:
        webbrowser.open(output_path.as_uri())
    except OSError:
        app.notify(
            f"Exported {output_path.name} (could not open browser).",
            timeout=6,
        )
        return
    app.notify(f"Opened {output_path.name} in browser.", timeout=6)
