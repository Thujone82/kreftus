"""Window icon loading."""

from __future__ import annotations

import sys
from importlib import resources
from pathlib import Path

import pygame

_ICON_NAME = "32x32.png"


def window_icon_path() -> Path | None:
    """Return the first available path to the bundled window icon."""
    try:
        pkg_icon = resources.files("gol").joinpath("assets", _ICON_NAME)
        with resources.as_file(pkg_icon) as path:
            if path.is_file():
                return path
    except (FileNotFoundError, OSError, TypeError):
        pass

    if getattr(sys, "frozen", False) and hasattr(sys, "_MEIPASS"):
        frozen = Path(sys._MEIPASS) / "gol" / "assets" / _ICON_NAME
        if frozen.is_file():
            return frozen

    pkg_root = Path(__file__).resolve().parent
    for candidate in (
        pkg_root / "assets" / _ICON_NAME,
        pkg_root.parent / "build" / _ICON_NAME,
    ):
        if candidate.is_file():
            return candidate
    return None


def set_window_icon() -> bool:
    """Set the pygame window title-bar icon when the PNG is available."""
    path = window_icon_path()
    if path is None:
        return False
    try:
        icon = pygame.image.load(str(path))
        pygame.display.set_icon(icon)
        return True
    except pygame.error:
        return False
