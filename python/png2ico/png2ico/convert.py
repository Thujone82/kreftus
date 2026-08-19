"""Convert a 256×256 PNG into a Windows ICO with common icon sizes."""

from __future__ import annotations

import argparse
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

ProgressFn = Callable[[str], None]
APP_TITLE = "PNG2ICO Converter"


class Png2IcoError(Exception):
    """Raised when the input PNG cannot be converted."""


def _format_size(size: tuple[int, int]) -> str:
    return f"{size[0]}x{size[1]}"


def _format_icon_sizes() -> str:
    return ", ".join(_format_size(size) for size in ICON_SIZES)


def convert_png_to_ico(
    png_path: Path,
    *,
    progress: ProgressFn | None = None,
) -> Path:
    """Resize a 256×256 PNG to the common Windows icon sizes and write an ICO.

    The ICO is written beside the PNG, using the same stem and a ``.ico`` suffix.
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
    log(f"Writing ICO: {output_path}")
    rgba.save(output_path, format="ICO", sizes=ICON_SIZES)
    log(f"  wrote {output_path.resolve()} ({output_path.stat().st_size} bytes)")
    return output_path


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="png2ico",
        description=(
            "Convert a 256x256 PNG into a Windows ICO containing the common "
            "icon sizes (16, 24, 32, 48, 64, 128, 256)."
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
