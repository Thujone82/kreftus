"""Tests for PNG → ICO conversion."""

from __future__ import annotations

import struct
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from png2ico.convert import ICON_SIZES, INPUT_SIZE, Png2IcoError, convert_png_to_ico

_PNG_MAGIC = b"\x89PNG\r\n\x1a\n"


def _ico_entries(path: Path) -> list[tuple[int, int, str]]:
    data = path.read_bytes()
    count = struct.unpack_from("<H", data, 4)[0]
    entries: list[tuple[int, int, str]] = []
    for index in range(count):
        offset = 6 + index * 16
        width = data[offset] or 256
        height = data[offset + 1] or 256
        image_offset = struct.unpack_from("<I", data, offset + 12)[0]
        payload = data[image_offset : image_offset + 8]
        kind = "png" if payload.startswith(_PNG_MAGIC) else "bmp"
        entries.append((width, height, kind))
    return entries


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
                for size in ICON_SIZES:
                    ico.size = size
                    ico.load()
                    self.assertEqual(ico.size, size)

    def test_ico_uses_png_for_256_and_bmp_for_smaller(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "app.png", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            kinds = {(width, height): kind for width, height, kind in _ico_entries(ico_path)}
            self.assertEqual(len(kinds), len(ICON_SIZES))
            self.assertEqual(kinds[(256, 256)], "png")
            for size in ICON_SIZES:
                if size != (256, 256):
                    self.assertEqual(kinds[size], "bmp")

    def test_prepare_icon_keeps_bmp_sizes_for_exe_embedding(self) -> None:
        from prepare_icon import prepare_icon

        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            png_path = _write_png(root / "app.png", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            embedded = root / "embedded.ico"
            prepare_icon(ico_path, embedded)
            kinds = {(width, height): kind for width, height, kind in _ico_entries(embedded)}
            self.assertNotIn((256, 256), kinds)
            self.assertTrue(kinds)
            self.assertEqual(set(kinds.values()), {"bmp"})

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
            self.assertTrue(any("256x256 as PNG" in line for line in steps))
            self.assertIn("wrote", joined)

    def test_uppercase_png_extension_is_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            png_path = _write_png(Path(temp_dir) / "APP.PNG", INPUT_SIZE)
            ico_path = convert_png_to_ico(png_path)
            self.assertEqual(ico_path.name, "APP.ico")
            self.assertTrue(ico_path.is_file())


if __name__ == "__main__":
    unittest.main()
