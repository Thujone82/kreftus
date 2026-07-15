"""Multi-window sparkline binning (pipe.ps1 pattern)."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta
from typing import Callable

SPARK_CHARS = "▁▂▃▄▅▆▇█"
BIN_COUNT = 24

# Status / snapshot display order (shortest → longest).
SPARKLINE_WINDOWS_LESS: tuple[tuple[str, float], ...] = (
    ("4H", 4),
    ("24H", 24),
    ("72H", 72),
)
SPARKLINE_WINDOWS_MORE: tuple[tuple[str, float], ...] = (
    ("90M", 1.5),
    ("4H", 4),
    ("8H", 8),
    ("12H", 12),
    ("24H", 24),
    ("36H", 36),
    ("72H", 72),
)
# Default / Less set — kept as SPARKLINE_WINDOWS for callers that only need Less.
SPARKLINE_WINDOWS = SPARKLINE_WINDOWS_LESS

# Monitoring dashboard **T** cycle order.
DASHBOARD_SPARKLINE_WINDOWS_LESS: tuple[tuple[str, float], ...] = (
    ("24H", 24),
    ("72H", 72),
    ("4H", 4),
)
DASHBOARD_SPARKLINE_WINDOWS_MORE: tuple[tuple[str, float], ...] = (
    ("24H", 24),
    ("36H", 36),
    ("72H", 72),
    ("90M", 1.5),
    ("4H", 4),
    ("8H", 8),
    ("12H", 12),
)
DASHBOARD_SPARKLINE_WINDOWS = DASHBOARD_SPARKLINE_WINDOWS_LESS
DEFAULT_DASHBOARD_SPARKLINE_HOURS = 24.0


@dataclass
class SparklineResult:
    glyphs: str
    binned_values: list[float | None]
    min_value: float | None
    max_value: float | None
    current: float | None
    bin_count: int = BIN_COUNT


def _normalized_time_detail(time_detail: str | None) -> str:
    return "more" if (time_detail or "").strip().lower() == "more" else "less"


def sparkline_windows(
    time_detail: str | None = None,
) -> tuple[tuple[str, float], ...]:
    """Return status/snapshot window set for Less or More time detail."""
    if _normalized_time_detail(time_detail) == "more":
        return SPARKLINE_WINDOWS_MORE
    return SPARKLINE_WINDOWS_LESS


def dashboard_sparkline_windows(
    time_detail: str | None = None,
) -> tuple[tuple[str, float], ...]:
    """Return monitoring **T** cycle windows for Less or More time detail."""
    if _normalized_time_detail(time_detail) == "more":
        return DASHBOARD_SPARKLINE_WINDOWS_MORE
    return DASHBOARD_SPARKLINE_WINDOWS_LESS


def next_dashboard_sparkline_window(
    hours: float = DEFAULT_DASHBOARD_SPARKLINE_HOURS,
    *,
    time_detail: str | None = None,
) -> tuple[str, float]:
    """Return the next dashboard sparkline window for the active time-detail set."""
    windows = dashboard_sparkline_windows(time_detail)
    for index, (_label, window_hours) in enumerate(windows):
        if window_hours == hours:
            return windows[(index + 1) % len(windows)]
    return windows[0]


def dashboard_sparkline_label(
    hours: float = DEFAULT_DASHBOARD_SPARKLINE_HOURS,
    *,
    time_detail: str | None = None,
) -> str:
    """Label for the active dashboard sparkline window."""
    windows = dashboard_sparkline_windows(time_detail)
    for label, window_hours in windows:
        if window_hours == hours:
            return label
    return windows[0][0]


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


def format_window_sparkline_rows(
    points: list[tuple[datetime, float]],
    color_fn: Callable[[float], str],
    *,
    indent: str = "",
    empty_bin: str = " ",
    fixed_label: bool = True,
    time_detail: str | None = None,
) -> list[str]:
    """Format sparkline rows for each window in the active time-detail set."""
    lines: list[str] = []
    for label, hours in sparkline_windows(time_detail):
        result = build_sparkline(points, hours=hours)
        core = colored_sparkline_markup(result, color_fn, empty_bin=empty_bin)
        row = format_sparkline_row(core, hours_label=label, fixed_label=fixed_label)
        lines.append(f"{indent}{row}")
    return lines
