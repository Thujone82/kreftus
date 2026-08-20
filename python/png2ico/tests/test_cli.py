"""Tests for the png2ico CLI."""

from __future__ import annotations

import importlib.util
import io
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path

from PIL import Image

from png2ico.convert import APP_TITLE, ICON_SIZES, INPUT_SIZE


def _load_entry_module():
    path = Path(__file__).resolve().parents[1] / "png2ico.py"
    spec = importlib.util.spec_from_file_location("png2ico_entry", path)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class CliTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.entry = _load_entry_module()

    def test_cli_writes_ico_and_prints_success(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = Path(temp_dir) / "logo.png"
            Image.new("RGBA", INPUT_SIZE, (40, 200, 80, 255)).save(png_path, format="PNG")
            stdout = io.StringIO()
            with redirect_stdout(stdout):
                code = self.entry.main([str(png_path)])
            ico_path = png_path.with_suffix(".ico")
            output = stdout.getvalue()
            self.assertEqual(code, 0)
            self.assertIn(APP_TITLE, output)
            self.assertIn("Checking input:", output)
            self.assertIn("Reading PNG...", output)
            self.assertIn("size OK (256x256)", output)
            self.assertIn("Generating Windows icon sizes:", output)
            self.assertIn("Writing ICO:", output)
            self.assertIn("256x256 as PNG", output)
            self.assertIn(f"Success: {ico_path.resolve()}", output)
            self.assertTrue(ico_path.is_file())
            with Image.open(ico_path) as ico:
                self.assertEqual(set(ico.info["sizes"]), set(ICON_SIZES))

    def test_cli_wrong_size_exits_nonzero(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = Path(temp_dir) / "logo.png"
            Image.new("RGBA", (64, 64), (40, 200, 80, 255)).save(png_path, format="PNG")
            stdout = io.StringIO()
            stderr = io.StringIO()
            with redirect_stdout(stdout), redirect_stderr(stderr):
                code = self.entry.main([str(png_path)])
            self.assertEqual(code, 1)
            self.assertIn(APP_TITLE, stdout.getvalue())
            self.assertIn("Checking input:", stdout.getvalue())
            self.assertIn("Reading PNG...", stdout.getvalue())
            self.assertIn("256x256", stderr.getvalue())
            self.assertIn("512x512", stderr.getvalue())
            self.assertIn("64x64", stderr.getvalue())
            self.assertIn("Failed.", stderr.getvalue())
            self.assertFalse(png_path.with_suffix(".ico").exists())

    def test_cli_missing_file_exits_nonzero(self) -> None:
        stdout = io.StringIO()
        stderr = io.StringIO()
        with redirect_stdout(stdout), redirect_stderr(stderr):
            code = self.entry.main(["missing.png"])
        self.assertEqual(code, 1)
        self.assertIn(APP_TITLE, stdout.getvalue())
        self.assertIn("Checking input: missing.png", stdout.getvalue())
        self.assertIn("File not found", stderr.getvalue())
        self.assertIn("Failed.", stderr.getvalue())

    def test_help_flags(self) -> None:
        for flag in ("-h", "-help", "--help"):
            with self.subTest(flag=flag):
                stdout = io.StringIO()
                with redirect_stdout(stdout), self.assertRaises(SystemExit) as ctx:
                    self.entry.main([flag])
                self.assertEqual(ctx.exception.code, 0)
                output = stdout.getvalue()
                self.assertIn(APP_TITLE, output)
                self.assertIn("usage:", output)
                self.assertIn("256x256", output)
                self.assertIn("512x512", output)
                self.assertIn("1024x1024", output)
                self.assertIn("2048x2048", output)


if __name__ == "__main__":
    unittest.main()
