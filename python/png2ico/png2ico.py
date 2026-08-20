#!/usr/bin/env python3
"""png2ico — convert a square PNG (256–2048) into a multi-size Windows ICO."""

from __future__ import annotations

import sys
from pathlib import Path

_ROOT = Path(__file__).resolve().parent
if str(_ROOT) not in sys.path:
    sys.path.insert(0, str(_ROOT))

from png2ico.convert import main


if __name__ == "__main__":
    raise SystemExit(main())
