"""Launcher directory resolution."""

from __future__ import annotations

import sys
from pathlib import Path


def application_dir() -> Path:
    """Directory beside gol.py, gol.exe, or gol.pyz."""
    if getattr(sys, "frozen", False):
        return Path(sys.executable).resolve().parent
    argv0 = Path(sys.argv[0]).resolve()
    if argv0.suffix in {".pyz", ".py"}:
        return argv0.parent
    return Path.cwd()
