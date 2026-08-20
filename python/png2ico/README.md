# png2ico

**Version 1.1.1**

Convert a square PNG (**256×256**, **512×512**, **1024×1024**, or **2048×2048**) into a Windows `.ico` that contains the common icon sizes. Larger sources are downscaled with LANCZOS so each icon size stays sharp.

**Author:** Kreft&Cursor

## Requirements

- Python 3.11+
- Dependencies: `pip install -r requirements.txt` (`pillow`)

## Quick start

```bash
cd python/png2ico
pip install -r requirements.txt
python png2ico.py path\to\icon.png
python png2ico.py -help
```

The ICO is written beside the PNG, using the same filename:

```text
C:\icons\app.png  →  C:\icons\app.ico
```

Each size is packed into one ICO using the usual Windows Vista layout: **256×256 as PNG** (smaller, valid in Explorer) and **16–128 as 32-bit BMP/DIB** (needed for `.exe` shell icons). Re-run png2ico on the original PNG to refresh an ICO that was built before this.

## Command-line options

| Flag | Description |
|------|-------------|
| `-h`, `-help`, `--help` | Show usage and exit |

## Input rules

- Path must point to an existing `.png` file
- Image must actually be PNG data
- Size must be exactly **256×256**, **512×512**, **1024×1024**, or **2048×2048** (any other size is an error)

## Icon sizes

The output ICO includes:

| Size |
|------|
| 16×16 |
| 24×24 |
| 32×32 |
| 48×48 |
| 64×64 |
| 128×128 |
| 256×256 |

## Tests

```bash
cd python/png2ico
python -m unittest discover tests
```

## Build (Windows)

From `python/png2ico/`:

```powershell
./build.ps1              # png2ico.pyz, then png2ico.exe
./build.ps1 -pyz         # png2ico.pyz only
./build.ps1 -exe         # png2ico.exe only
./build.ps1 -exe -upx    # optional UPX compression of png2ico.exe
```

| Output | Usage |
|--------|--------|
| `png2ico.exe` | Standalone executable (PyInstaller; icon from `png2ico.ico`) |
| `png2ico.pyz` | `python png2ico.pyz` — requires Pillow installed |

## See also

- Project page: [png2ico/](https://kreft.us/png2ico/)

## Changelog

### 1.1.1

- Slim the Windows exe: stop collecting every Pillow submodule so NumPy/OpenBLAS are not bundled.

### 1.1.0

- Accept square PNG inputs of **256×256**, **512×512**, **1024×1024**, or **2048×2048**, then LANCZOS-resize down to the Windows icon sizes.

### 1.0.1

- Standardize ICO encoding to the Windows Vista layout: **256×256 as PNG**, **16–128 as 32-bit BMP/DIB** with a proper AND mask. Fixes invalid all-BMP files from Pillow’s `bitmap_format="bmp"` path so Explorer can open the ICO and `.exe` embedding still has classic bitmap sizes.

### 1.0.0

- Initial release: 256×256 PNG → multi-size Windows ICO
