"""Convert a 256×256 PNG into a Windows ICO with common icon sizes."""

from __future__ import annotations

import argparse
import io
import struct
import sys
from collections.abc import Callable
from pathlib import Path

from PIL import Image, UnidentifiedImageError

INPUT_SIZE = (256, 256)
ICON_SIZES = [
    (16, 16),
    (24, 24),
    (32, 32),
    (48, 48),
    (64, 64),
    (128, 128),
    (256, 256),
]
PNG_SIZE = (256, 256)

ProgressFn = Callable[[str], None]
APP_TITLE = "PNG2ICO Converter"


class Png2IcoError(Exception):
    """Raised when the input PNG cannot be converted."""


def _format_size(size: tuple[int, int]) -> str:
    return f"{size[0]}x{size[1]}"


def _format_icon_sizes() -> str:
    return ", ".join(_format_size(size) for size in ICON_SIZES)


def _png_payload(image: Image.Image) -> bytes:
    buffer = io.BytesIO()
    image.save(buffer, format="PNG")
    return buffer.getvalue()


def _and_mask_bytes(width: int, height: int) -> bytes:
    row_bytes = ((width + 31) // 32) * 4
    return bytes(row_bytes * height)


def _bmp_dib_payload(image: Image.Image) -> bytes:
    """32-bit ICO DIB: XOR bitmap plus AND mask (height stored as 2×)."""
    width, height = image.size
    rgba = image.convert("RGBA")
    flipped = rgba.transpose(Image.Transpose.FLIP_TOP_BOTTOM)
    red, green, blue, alpha = flipped.split()
    bgra = Image.merge("RGBA", (blue, green, red, alpha))
    xor = bgra.tobytes()
    and_mask = _and_mask_bytes(width, height)
    header = struct.pack(
        "<IiiHHIIiiII",
        40,
        width,
        height * 2,
        1,
        32,
        0,
        len(xor) + len(and_mask),
        0,
        0,
        0,
        0,
    )
    return header + xor + and_mask


def write_windows_ico(frames: list[tuple[tuple[int, int], Image.Image]], path: Path) -> None:
    """Write a Vista-style ICO: PNG for 256×256, 32-bit BMP/DIB for smaller sizes."""
    payloads: list[tuple[tuple[int, int], bytes]] = []
    for size, image in frames:
        if size == PNG_SIZE:
            payloads.append((size, _png_payload(image)))
        else:
            payloads.append((size, _bmp_dib_payload(image)))

    count = len(payloads)
    offset = 6 + 16 * count
    directory = bytearray()
    blob = bytearray()
    for (width, height), payload in payloads:
        directory.append(0 if width >= 256 else width)
        directory.append(0 if height >= 256 else height)
        directory.append(0)
        directory.append(0)
        directory.extend(struct.pack("<HHI", 1, 32, len(payload)))
        directory.extend(struct.pack("<I", offset))
        blob.extend(payload)
        offset += len(payload)

    header = b"\x00\x00\x01\x00" + struct.pack("<H", count)
    path.write_bytes(header + bytes(directory) + bytes(blob))


def convert_png_to_ico(
    png_path: Path,
    *,
    progress: ProgressFn | None = None,
) -> Path:
    """Resize a 256×256 PNG to the common Windows icon sizes and write an ICO.

    The ICO is written beside the PNG, using the same stem and a ``.ico`` suffix.
    256×256 is stored as PNG; smaller sizes use 32-bit BMP/DIB so Explorer and
    ``.exe`` embedding still have classic bitmap entries.
    """
    log: ProgressFn = progress if progress is not None else (lambda _msg: None)
    png_path = Path(png_path)

    log(f"Checking input: {png_path}")
    if not png_path.is_file():
        raise Png2IcoError(f"File not found: {png_path}")
    if png_path.suffix.lower() != ".png":
        raise Png2IcoError(f"Input must be a .png file: {png_path}")
    log(f"  found {png_path.resolve()}")

    log("Reading PNG...")
    try:
        with Image.open(png_path) as image:
            image.load()
            if image.format != "PNG":
                raise Png2IcoError(f"Input is not a PNG image: {png_path}")
            log(f"  format PNG, size {_format_size(image.size)}")
            if image.size != INPUT_SIZE:
                raise Png2IcoError(
                    f"Input must be {_format_size(INPUT_SIZE)}, "
                    f"got {_format_size(image.size)}"
                )
            log(f"  size OK ({_format_size(INPUT_SIZE)})")
            rgba = image.convert("RGBA")
    except Png2IcoError:
        raise
    except UnidentifiedImageError as exc:
        raise Png2IcoError(f"Cannot read PNG: {png_path}") from exc
    except OSError as exc:
        raise Png2IcoError(f"Cannot read PNG: {png_path} ({exc})") from exc

    output_path = png_path.with_suffix(".ico")
    log(f"Generating Windows icon sizes: {_format_icon_sizes()}")
    log("  256x256 as PNG; 16–128 as 32-bit BMP/DIB")
    log(f"Writing ICO: {output_path}")
    frames = [
        (
            size,
            rgba if size == rgba.size else rgba.resize(size, Image.Resampling.LANCZOS),
        )
        for size in ICON_SIZES
    ]
    write_windows_ico(frames, output_path)
    log(f"  wrote {output_path.resolve()} ({output_path.stat().st_size} bytes)")
    return output_path


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="png2ico",
        description=(
            "Convert a 256x256 PNG into a Windows ICO containing the common "
            "icon sizes (16, 24, 32, 48, 64, 128, 256). 256x256 is stored as "
            "PNG; smaller sizes use 32-bit BMP/DIB."
        ),
        add_help=False,
    )
    parser.add_argument(
        "-h",
        "-help",
        "--help",
        action="help",
        help="Show this help message and exit",
    )
    parser.add_argument(
        "png",
        type=Path,
        help="Path to a 256x256 PNG file",
    )
    return parser.parse_args(argv)


def _print_step(message: str) -> None:
    print(message, flush=True)


def main(argv: list[str] | None = None) -> int:
    print(APP_TITLE, flush=True)
    args = parse_args(argv)
    try:
        output_path = convert_png_to_ico(args.png, progress=_print_step)
    except Png2IcoError as exc:
        print(f"error: {exc}", file=sys.stderr)
        print("Failed.", file=sys.stderr)
        return 1
    print(f"Success: {output_path.resolve()}", flush=True)
    return 0
