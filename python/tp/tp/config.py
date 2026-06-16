"""Load and save tp.ini configuration."""

from __future__ import annotations

import configparser
import sys
from dataclasses import dataclass, field
from pathlib import Path


@dataclass
class Settings:
    logging_enabled: bool = False
    log_directory: str = "."
    log_file_name: str = "tp.log"


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
        config.settings.log_file_name = section.get("LogFileName", "tp.log")

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
