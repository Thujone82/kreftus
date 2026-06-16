"""Session-scoped debug logging to debug.log in the configured log directory."""

from __future__ import annotations

from datetime import datetime
from pathlib import Path

from tp.config import AppConfig, resolved_log_directory

DEBUG_FILE_NAME = "debug.log"
_TIMESTAMP_FORMAT = "%Y-%m-%d %H:%M:%S.%f"

_enabled = False
_log_path: Path | None = None


def is_enabled() -> bool:
    return _enabled


def log_path() -> Path | None:
    return _log_path


def set_debug_enabled(config: AppConfig | None, enabled: bool) -> None:
    """Enable or disable debug logging for this session."""
    global _enabled, _log_path
    _enabled = enabled
    if not enabled or config is None:
        _log_path = None
        _sync_bleak_logging(False, None)
        return
    _log_path = resolved_log_directory(config) / DEBUG_FILE_NAME
    _sync_bleak_logging(True, _log_path)
    write("debug logging enabled", config=config)


def _sync_bleak_logging(enabled: bool, log_path: Path | None) -> None:
    try:
        from tp.ble import configure_bleak_debug_logging

        configure_bleak_debug_logging(enabled, log_path)
    except Exception:
        pass


def write(message: str, *, config: AppConfig | None = None) -> None:
    """Append a timestamped line when debug mode is on."""
    if not _enabled:
        return
    path = _log_path
    if path is None and config is not None:
        path = resolved_log_directory(config) / DEBUG_FILE_NAME
    if path is None:
        return
    stamp = datetime.now().strftime(_TIMESTAMP_FORMAT)[:-3]
    line = f"{stamp} {message}\n"
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        with path.open("a", encoding="utf-8", newline="\n") as handle:
            handle.write(line)
    except OSError:
        pass


def write_exception(message: str, exc: BaseException, *, config: AppConfig | None = None) -> None:
    write(f"{message}: {type(exc).__name__}: {exc}", config=config)
