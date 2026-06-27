#!/usr/bin/env python3
"""GoLPy — terminal-only entry point (no pygame)."""

from __future__ import annotations

import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from gol.help_text import build_parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser(tui_only=True).parse_args(argv)
    speed = max(10, min(200, args.speed))
    from gol.ui.tui.app import run_tui

    run_tui(
        mode=args.mode,  # type: ignore[arg-type]
        pattern=args.pattern,
        speed=speed,
        debug=args.debug,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
