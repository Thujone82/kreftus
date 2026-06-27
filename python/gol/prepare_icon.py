"""Prepare icon-32.ico for PyInstaller / Windows shell display."""

from __future__ import annotations

import struct
import sys
from pathlib import Path


def _parse_entries(data: bytes) -> tuple[list[bytes], list[bytes]]:
    if data[:4] != b"\x00\x00\x01\x00":
        raise ValueError("Not a valid .ico file")

    count = struct.unpack_from("<H", data, 4)[0]
    entries: list[bytes] = []
    images: list[bytes] = []

    for index in range(count):
        offset = 6 + index * 16
        entry = data[offset : offset + 16]
        image_offset = struct.unpack_from("<I", entry, 12)[0]
        image_size = struct.unpack_from("<I", entry, 8)[0]
        payload = data[image_offset : image_offset + image_size]
        if payload.startswith(b"\x89PNG\r\n\x1a\n"):
            continue
        entries.append(entry)
        images.append(payload)

    if not entries:
        raise ValueError("No BMP icon entries found in source .ico")

    return entries, images


def prepare_icon(src: Path, dst: Path) -> None:
    data = src.read_bytes()
    entries, images = _parse_entries(data)

    header = b"\x00\x00\x01\x00" + struct.pack("<H", len(entries))
    image_offset = 6 + 16 * len(entries)
    directory = bytearray()
    payloads = bytearray()

    for entry, payload in zip(entries, images):
        entry = bytearray(entry)
        struct.pack_into("<I", entry, 8, len(payload))
        struct.pack_into("<I", entry, 12, image_offset)
        directory.extend(entry)
        payloads.extend(payload)
        image_offset += len(payload)

    dst.write_bytes(header + bytes(directory) + bytes(payloads))


def reapply_icon(exe: Path, ico: Path) -> None:
    from PyInstaller.utils.win32.icon import CopyIcons_FromIco

    CopyIcons_FromIco(str(exe.resolve()), [str(ico.resolve())])


def main(argv: list[str] | None = None) -> int:
    args = argv if argv is not None else sys.argv[1:]
    if len(args) == 3 and args[0] == "reapply":
        exe = Path(args[1])
        ico = Path(args[2])
        if not exe.is_file():
            print(f"Executable not found: {exe}", file=sys.stderr)
            return 1
        if not ico.is_file():
            print(f"Icon not found: {ico}", file=sys.stderr)
            return 1
        reapply_icon(exe, ico)
        print(f"Re-applied icon to: {exe}")
        return 0

    if len(args) != 2:
        print(
            f"Usage: {Path(__file__).name} <source.ico> <output.ico>\n"
            f"       {Path(__file__).name} reapply <target.exe> <icon.ico>",
            file=sys.stderr,
        )
        return 1

    src = Path(args[0])
    dst = Path(args[1])
    if not src.is_file():
        print(f"Icon not found: {src}", file=sys.stderr)
        return 1

    prepare_icon(src, dst)
    print(f"Prepared icon: {dst}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
