#!/usr/bin/env python3
"""Regenerate gol/patterns.json from gol/index.html (dev helper, run once)."""

from __future__ import annotations

import json
import re
from pathlib import Path

_REPO = Path(__file__).resolve().parents[2]
HTML = _REPO / "gol" / "index.html"
OUT = Path(__file__).resolve().parent / "gol" / "patterns.json"


def main() -> None:
    text = HTML.read_text(encoding="utf-8")

    labels: dict[str, str] = {}
    for match in re.finditer(
        r'<option value="([^"]+)">([^<]+)</option>', text
    ):
        key, label = match.group(1), match.group(2)
        if key != "none":
            labels[key] = label

    p_match = re.search(r"const P = (\{[\s\S]*?\n  \});", text)
    if not p_match:
        raise SystemExit("Could not find pattern object P in index.html")

    raw = p_match.group(1)
    # JS object -> JSON: quote keys
    raw = re.sub(r"(\w+):", r'"\1":', raw)
    coords_map: dict[str, list[list[int]]] = json.loads(raw)

    patterns = {}
    for key, cells in coords_map.items():
        patterns[key] = {
            "label": labels.get(key, key),
            "cells": cells,
        }

    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(patterns, indent=2) + "\n", encoding="utf-8")
    print(f"Wrote {len(patterns)} patterns to {OUT}")


if __name__ == "__main__":
    main()
