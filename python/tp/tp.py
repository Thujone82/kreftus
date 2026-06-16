#!/usr/bin/env python3
"""TemPy — ThermoPro TP35x Monitor — entry point."""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

# Allow running as `python tp.py` from python/tp/
_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from tp.config import default_ini_path, load_config, save_config
from tp.ui.app import run_app


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="TemPy — ThermoPro TP35x Monitor")
    parser.add_argument(
        "-debug",
        "--debug",
        action="store_true",
        help="Enable session debug.log in the configured log directory",
    )
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    args = parse_args(argv)
    ini_path = default_ini_path()
    config = load_config(ini_path)
    if not ini_path.exists():
        save_config(config)
    run_app(config, debug_enabled=args.debug)


if __name__ == "__main__":
    main()
