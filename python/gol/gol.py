#!/usr/bin/env python3
"""GoLPy — Conway's Game of Life — entry point."""

from __future__ import annotations

import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from gol.help_text import build_parser


def main(argv: list[str] | None = None) -> int:
    args = build_parser(tui_only=False).parse_args(argv)
    speed = max(10, min(200, args.speed))
    if args.tui:
        from gol.ui.tui.app import run_tui

        run_tui(
            mode=args.mode,  # type: ignore[arg-type]
            pattern=args.pattern,
            speed=speed,
            debug=args.debug,
        )
    else:
        from gol.ui.app import run_app

        run_app(
            mode=args.mode,  # type: ignore[arg-type]
            pattern=args.pattern,
            speed=speed,
            debug=args.debug,
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
