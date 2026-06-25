"""24-hour sparkline binning (pipe.ps1 pattern)."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import Callable

SPARK_CHARS = "▁▂▃▄▅▆▇█"
BIN_COUNT = 24
SPARKLINE_WINDOWS: tuple[tuple[str, float], ...] = (
    ("4H", 4),
    ("24H", 24),
    ("72H", 72),
)
DASHBOARD_SPARKLINE_WINDOWS: tuple[tuple[str, float], ...] = (
    ("24H", 24),
    ("72H", 72),
    ("4H", 4),
)
DEFAULT_DASHBOARD_SPARKLINE_HOURS = 24.0


@dataclass
class SparklineResult:
    glyphs: str
    binned_values: list[float | None]
    min_value: float | None
    max_value: float | None
    current: float | None
    bin_count: int = BIN_COUNT


def next_dashboard_sparkline_window(
    hours: float = DEFAULT_DASHBOARD_SPARKLINE_HOURS,
) -> tuple[str, float]:
    """Return the next dashboard sparkline window (24H → 72H → 4H → 24H)."""
    windows = DASHBOARD_SPARKLINE_WINDOWS
    for index, (_label, window_hours) in enumerate(windows):
        if window_hours == hours:
            return windows[(index + 1) % len(windows)]
    return windows[0]


def dashboard_sparkline_label(hours: float = DEFAULT_DASHBOARD_SPARKLINE_HOURS) -> str:
    """Label for the active dashboard sparkline window."""
    for label, window_hours in DASHBOARD_SPARKLINE_WINDOWS:
        if window_hours == hours:
            return label
    return DASHBOARD_SPARKLINE_WINDOWS[0][0]


def _points_in_window(
    points: list[tuple[datetime, float]],
    end_time: datetime | None,
    hours: float,
) -> list[tuple[datetime, float]]:
    end = end_time or datetime.now()
    window_start = end - timedelta(hours=hours)
    return [point for point in points if window_start <= point[0] < end]


def window_value_extremes(
    points: list[tuple[datetime, float]],
    end_time: datetime | None = None,
    *,
    hours: float = 24,
    value_getter: Callable[[tuple[datetime, float]], float] | None = None,
) -> tuple[float | None, float | None]:
    """Minimum and maximum raw sample values in a sparkline time window."""
    if value_getter is None:
        value_getter = lambda item: item[1]
    values = [value_getter(point) for point in _points_in_window(points, end_time, hours)]
    if not values:
        return None, None
    return min(values), max(values)


def build_sparkline(
    points: list[tuple[datetime, float]],
    end_time: datetime | None = None,
    *,
    hours: float = 24,
    bin_count: int = BIN_COUNT,
    value_getter: Callable[[tuple[datetime, float]], float] | None = None,
) -> SparklineResult:
    """Bin points into fixed-width sparkline glyphs ending at end_time.

    Each window uses ``bin_count`` bins spanning ``hours`` (e.g. 24 bins × 1 h = 24H).
    Empty bins stay empty (space glyph). With a single populated bin the rightmost
    bin renders as ▁; additional bins scale relative to min/max across filled bins.
    """
    if value_getter is None:
        value_getter = lambda item: item[1]

    end = end_time or datetime.now()
    in_window = _points_in_window(points, end_time, hours)

    bin_size = timedelta(hours=hours / bin_count)
    binned: list[float | None] = []
    bin_start = end

    for _ in range(bin_count):
        bin_end = bin_start
        bin_start = bin_end - bin_size
        in_bin = [
            value_getter(p)
            for p in in_window
            if bin_start <= p[0] < bin_end
        ]
        if in_bin:
            binned.insert(0, sum(in_bin) / len(in_bin))
        else:
            binned.insert(0, None)

    valid = [v for v in binned if v is not None]

    if not valid:
        return SparklineResult(" " * bin_count, binned, None, None, None, bin_count)

    min_value = min(valid)
    max_value = max(valid)
    current = next(v for v in reversed(binned) if v is not None)
    value_range = max_value - min_value
    single_bin = len(valid) == 1

    glyphs: list[str] = []
    for value in binned:
        if value is None:
            glyphs.append(" ")
            continue
        if single_bin or value_range < 1e-8:
            glyphs.append(SPARK_CHARS[0])
            continue
        normalized = (value - min_value) / value_range
        index = min(len(SPARK_CHARS) - 1, int(normalized * (len(SPARK_CHARS) - 1)))
        glyphs.append(SPARK_CHARS[index])

    return SparklineResult(
        "".join(glyphs), binned, min_value, max_value, current, bin_count
    )


def populated_hour_bin_count(
    points: list[tuple[datetime, float]],
    end_time: datetime | None = None,
) -> int:
    """Count populated 1-hour bins in the default 24H dashboard window."""
    return populated_bin_count(points, end_time=end_time, hours=24)


def populated_bin_count(
    points: list[tuple[datetime, float]],
    end_time: datetime | None = None,
    *,
    hours: float = 24,
) -> int:
    """Count sparkline bins with at least one sample in the given window."""
    if not points:
        return 0
    result = build_sparkline(points, end_time=end_time, hours=hours)
    return sum(1 for value in result.binned_values if value is not None)


def colored_sparkline_markup(
    result: SparklineResult,
    color_fn: Callable[[float], str],
    *,
    empty_bin: str = " ",
) -> str:
    """Build Rich markup string with per-glyph band colors."""
    parts: list[str] = []
    for glyph, value in zip(result.glyphs, result.binned_values):
        if glyph == " " or value is None:
            parts.append(empty_bin)
            continue
        color = color_fn(value)
        parts.append(f"[{color}]{glyph}[/]")
    return "".join(parts)


def format_sparkline_row(
    sparkline_markup: str,
    *,
    hours_label: str = "24H",
    fixed_label: bool = False,
) -> str:
    """Wrap colored sparkline glyphs with a time-axis label."""
    if fixed_label:
        padded_label = f"{hours_label:>3} Ago"
        prefix = f"[dim]┕ {padded_label} |[/]"
    else:
        prefix = f"[dim]┕ {hours_label} Ago |[/]"
    return f"{prefix}{sparkline_markup}[dim]| Now[/]"
