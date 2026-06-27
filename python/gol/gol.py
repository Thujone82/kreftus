#!/usr/bin/env python3
"""GoLPy — Conway's Game of Life — entry point."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from gol.ui.app import run_app


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="GoLPy — Conway's Game of Life")
    parser.add_argument(
        "--mode",
        choices=("wrapped", "infinite"),
        default="wrapped",
        help="Grid mode (default: wrapped)",
    )
    parser.add_argument("--pattern", help="Load a built-in pattern by key on startup")
    parser.add_argument(
        "--speed",
        type=int,
        default=100,
        metavar="N",
        help="Simulation speed 10–200 (default: 100)",
    )
    parser.add_argument("-debug", action="store_true", help="Log step stats every 100 generations")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    speed = max(10, min(200, args.speed))
    run_app(
        mode=args.mode,  # type: ignore[arg-type]
        pattern=args.pattern,
        speed=speed,
        debug=args.debug,
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
