"""Tests for TemPy CLI argument parsing and session overrides."""

from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from tp.config import TIME_DETAIL_LESS, TIME_DETAIL_MORE, AppConfig, Settings, load_config, save_config


def _load_entry_module():
    path = Path(__file__).resolve().parents[1] / "tp.py"
    spec = importlib.util.spec_from_file_location("tempy_entry", path)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class CliArgsTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.entry = _load_entry_module()

    def test_parse_more_flag(self) -> None:
        args = self.entry.parse_args(["-more"])
        self.assertTrue(args.more)
        self.assertFalse(self.entry.parse_args([]).more)

    def test_more_overrides_config_for_snapshot_without_saving(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            ini_path = root / "tp.ini"
            config = AppConfig(
                ini_path=ini_path,
                settings=Settings(time_detail=TIME_DETAIL_LESS),
                devices={"F7:94:C0:18:DD:7D": "Office"},
            )
            save_config(config)

            printed: list[str] = []

            class FakeConsole:
                def print(self, text: object) -> None:
                    printed.append(str(text))

            with (
                patch.object(self.entry, "default_ini_path", return_value=ini_path),
                patch.object(self.entry, "Console", FakeConsole),
                patch.object(
                    self.entry,
                    "_render_snapshot",
                    side_effect=lambda cfg, device_filter=None: cfg.settings.time_detail,
                ),
            ):
                self.entry.main(["-x", "-more"])

            self.assertEqual(printed, [TIME_DETAIL_MORE])
            reloaded = load_config(ini_path)
            self.assertEqual(reloaded.settings.time_detail, TIME_DETAIL_LESS)


if __name__ == "__main__":
    unittest.main()
