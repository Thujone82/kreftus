"""Load and save tp.ini configuration."""

from __future__ import annotations

import configparser
import sys
from dataclasses import dataclass, field
from pathlib import Path


POLL_MODE_INCREMENTAL = "incremental"
POLL_MODE_LIVE = "live"
DEFAULT_POLL_MODE = POLL_MODE_INCREMENTAL
VALID_POLL_MODES = frozenset({POLL_MODE_INCREMENTAL, POLL_MODE_LIVE})

TIME_DETAIL_LESS = "less"
TIME_DETAIL_MORE = "more"
DEFAULT_TIME_DETAIL = TIME_DETAIL_LESS
VALID_TIME_DETAILS = frozenset({TIME_DETAIL_LESS, TIME_DETAIL_MORE})


@dataclass
class Settings:
    logging_enabled: bool = False
    log_directory: str = "."
    log_file_name: str = "tp_log.csv"
    poll_mode: str = DEFAULT_POLL_MODE
    time_detail: str = DEFAULT_TIME_DETAIL


@dataclass
class AppConfig:
    settings: Settings = field(default_factory=Settings)
    devices: dict[str, str] = field(default_factory=dict)  # MAC -> display name
    ini_path: Path = field(default_factory=Path)


def normalize_mac(mac: str) -> str:
    """Normalize MAC to uppercase colon-separated form."""
    cleaned = mac.strip().upper().replace("-", ":")
    parts = cleaned.split(":")
    if len(parts) == 6 and all(len(p) <= 2 for p in parts):
        return ":".join(p.zfill(2) for p in parts)
    return cleaned


def filter_devices(
    devices: dict[str, str],
    filter_text: str | None,
) -> list[tuple[str, str]]:
    """Return devices whose display name contains filter_text (case-insensitive)."""
    if not filter_text:
        return list(devices.items())
    needle = filter_text.casefold()
    return [(mac, name) for mac, name in devices.items() if needle in name.casefold()]


def default_device_name(mac: str, *, bluetooth_name: str | None = None) -> str:
    """Default display name from BLE advertisement, else last MAC byte."""
    if bluetooth_name and bluetooth_name.strip():
        return bluetooth_name.strip()
    normalized = normalize_mac(mac)
    suffix = normalized.split(":")[-1]
    return f"Thermometer [{suffix}]"


def application_dir() -> Path:
    """Directory beside the launcher (tp.py, tp.exe, or tp.pyz)."""
    if getattr(sys, "frozen", False):
        return Path(sys.executable).resolve().parent

    argv0 = Path(sys.argv[0]).resolve()
    if argv0.suffix.lower() == ".pyz":
        return argv0.parent
    if argv0.suffix.lower() in {".py", ".pyw"}:
        return argv0.parent

    return Path.cwd()


def default_ini_path() -> Path:
    return application_dir() / "tp.ini"


def _make_parser() -> configparser.ConfigParser:
    """INI parser using '=' only so MAC addresses can be keys."""
    parser = configparser.ConfigParser(delimiters=("=",), interpolation=None)
    parser.optionxform = str  # preserve MAC address casing
    return parser


def load_config(ini_path: Path | None = None) -> AppConfig:
    path = ini_path or default_ini_path()
    config = AppConfig(ini_path=path)

    if not path.exists():
        return config

    parser = _make_parser()
    parser.read(path, encoding="utf-8")

    if parser.has_section("Settings"):
        section = parser["Settings"]
        config.settings.logging_enabled = section.getboolean(
            "LoggingEnabled", fallback=False
        )
        config.settings.log_directory = section.get("LogDirectory", ".")
        config.settings.log_file_name = section.get("LogFileName", "tp_log.csv")
        poll_mode = section.get("PollMode", DEFAULT_POLL_MODE).strip().lower()
        if poll_mode in VALID_POLL_MODES:
            config.settings.poll_mode = poll_mode
        time_detail = section.get("TimeDetail", DEFAULT_TIME_DETAIL).strip().lower()
        if time_detail in VALID_TIME_DETAILS:
            config.settings.time_detail = time_detail

    if parser.has_section("Devices"):
        for mac, name in parser.items("Devices"):
            if mac.startswith(";") or not mac.strip():
                continue
            config.devices[normalize_mac(mac)] = name.strip()

    return config


def save_config(config: AppConfig) -> None:
    parser = _make_parser()
    parser["Settings"] = {
        "LoggingEnabled": str(config.settings.logging_enabled).lower(),
        "LogDirectory": config.settings.log_directory,
        "LogFileName": config.settings.log_file_name,
        "PollMode": config.settings.poll_mode,
        "TimeDetail": config.settings.time_detail,
    }
    parser["Devices"] = {mac: name for mac, name in config.devices.items()}

    parent = config.ini_path.parent
    if not parent.exists():
        parent.mkdir(parents=True, exist_ok=True)
    with config.ini_path.open("w", encoding="utf-8") as handle:
        parser.write(handle)


def resolved_log_directory(config: AppConfig) -> Path:
    """Resolve log directory; relative paths are beside the launcher."""
    directory = Path(config.settings.log_directory).expanduser()
    if not directory.is_absolute():
        directory = application_dir() / directory
    return directory.resolve()


def resolved_log_path(config: AppConfig) -> Path:
    return resolved_log_directory(config) / config.settings.log_file_name


def poll_mode_label(mode: str) -> str:
    if mode == POLL_MODE_LIVE:
        return "live (single snapshot)"
    return "incremental (minute history)"


def time_detail_label(mode: str) -> str:
    if mode == TIME_DETAIL_MORE:
        return "More (90M/4/8/12/24/36/72H)"
    return "Less (4/24/72H)"


def rename_log_file(
    config: AppConfig,
    old_filename: str,
    new_filename: str,
    *,
    overwrite: bool = False,
) -> str | None:
    """Rename the CSV log when the filename setting changes.

    Returns an error message, or ``"exists"`` when *new_filename* is taken and
  overwrite was not requested.
    """
    old_name = old_filename.strip()
    new_name = new_filename.strip()
    if not new_name:
        return "Log filename cannot be empty."
    if old_name == new_name:
        return None

    old_path = resolved_log_directory(config) / old_name
    new_path = resolved_log_directory(config) / new_name
    if not old_path.is_file():
        return None
    if new_path.exists() and new_path.resolve() != old_path.resolve():
        if not overwrite:
            return "exists"
        try:
            new_path.unlink()
        except OSError as exc:
            return f"Cannot remove {new_path}: {exc}"
    try:
        old_path.rename(new_path)
    except OSError as exc:
        return f"Cannot rename {old_path} to {new_path}: {exc}"
    return None


def probe_log_directory(directory: str) -> tuple[Path | None, str | None]:
    """Verify that a log directory exists and is writable without creating the log file."""
    if not directory.strip():
        return None, "Log directory cannot be empty."

    dir_path = Path(directory).expanduser()
    if not dir_path.is_absolute():
        dir_path = application_dir() / dir_path
    dir_path = dir_path.resolve()

    try:
        dir_path.mkdir(parents=True, exist_ok=True)
        probe_file = dir_path / ".tp_write_probe"
        probe_file.write_text("", encoding="utf-8")
        probe_file.unlink()
    except OSError as exc:
        return dir_path, f"Cannot write to {dir_path}: {exc}"
    return dir_path, None


def probe_log_path(directory: str, filename: str) -> tuple[Path | None, str | None]:
    """Resolve and verify that a log file can be opened for append."""
    if not filename.strip():
        return None, "Log filename cannot be empty."

    dir_path = Path(directory).expanduser()
    if not dir_path.is_absolute():
        dir_path = application_dir() / dir_path
    log_path = (dir_path / filename.strip()).resolve()

    try:
        log_path.parent.mkdir(parents=True, exist_ok=True)
        with log_path.open("a", encoding="utf-8"):
            pass
    except OSError as exc:
        return log_path, f"Cannot write to {log_path}: {exc}"
    return log_path, None
