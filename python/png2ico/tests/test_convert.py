"""Tests for PNG → ICO conversion."""

from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from PIL import Image

from png2ico.convert import ICON_SIZES, INPUT_SIZE, Png2IcoError, convert_png_to_ico


def _write_png(path: Path, size: tuple[int, int], color: tuple[int, int, int, int] = (0, 128, 255, 255)) -> Path:
    Image.new("RGBA", size, color).save(path, format="PNG")
    return path


class ConvertTests(unittest.TestCase):
    def test_writes_ico_beside_png_with_same_stem(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            self.assertEqual(ico_path, png_path.with_suffix(".ico"))
            self.assertTrue(ico_path.is_file())

    def test_ico_contains_common_windows_sizes(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            with Image.open(ico_path) as ico:
                self.assertEqual(set(ico.info["sizes"]), set(ICON_SIZES))

    def test_wrong_size_is_an_error(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", (128, 128))
            with self.assertRaises(Png2IcoError) as ctx:
                convert_png_to_ico(png_path)
            self.assertIn("256x256", str(ctx.exception))
            self.assertIn("128x128", str(ctx.exception))
            self.assertFalse(png_path.with_suffix(".ico").exists())

    def test_non_square_256_is_an_error(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", (256, 128))
            with self.assertRaises(Png2IcoError) as ctx:
                convert_png_to_ico(png_path)
            self.assertIn("256x128", str(ctx.exception))

    def test_missing_file_is_an_error(self) -> None:
        missing = Path("does-not-exist.png")
        with self.assertRaises(Png2IcoError) as ctx:
            convert_png_to_ico(missing)
        self.assertIn("File not found", str(ctx.exception))

    def test_non_png_extension_is_an_error(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            jpeg_path = Path(temp_dir) / "app.jpg"
            Image.new("RGB", INPUT_SIZE, (255, 0, 0)).save(jpeg_path, format="JPEG")
            with self.assertRaises(Png2IcoError) as ctx:
                convert_png_to_ico(jpeg_path)
            self.assertIn(".png", str(ctx.exception))

    def test_png_extension_but_not_png_data_is_an_error(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            fake_png = Path(temp_dir) / "app.png"
            Image.new("RGB", INPUT_SIZE, (255, 0, 0)).save(fake_png, format="JPEG")
            with self.assertRaises(Png2IcoError):
                convert_png_to_ico(fake_png)

    def test_progress_reports_each_step(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", INPUT_SIZE)
            steps: list[str] = []
            convert_png_to_ico(png_path, progress=steps.append)
            joined = "\n".join(steps)
            self.assertTrue(any(line.startswith("Checking input:") for line in steps))
            self.assertIn("Reading PNG...", steps)
            self.assertIn("  size OK (256x256)", steps)
            self.assertTrue(any("Generating Windows icon sizes:" in line for line in steps))
            self.assertTrue(any(line.startswith("Writing ICO:") for line in steps))
            self.assertIn("wrote", joined)

    def test_uppercase_png_extension_is_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "APP.PNG", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            self.assertEqual(ico_path.name, "APP.ico")
            self.assertTrue(ico_path.is_file())


if __name__ == "__main__":
    unittest.main()
