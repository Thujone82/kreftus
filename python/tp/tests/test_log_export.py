"""Tests for CSV log export to HTML."""

from __future__ import annotations

import json
import tempfile
import unittest
from datetime import datetime, timedelta
from pathlib import Path

from tp.config import AppConfig, Settings
from tp.log_export import (
    build_export_payload,
    can_export_log,
    export_log_to_html,
    filter_series_for_hours,
    load_log_rows_for_export,
    render_log_export_html,
)
from tp.history import CSV_HEADER


class LogExportTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.root = Path(self.temp_dir.name)
        self.mac = "E0:A4:4B:A4:53:0D"
        self.other_mac = "F7:94:C0:18:DD:7D"
        self.log_path = self.root / "tp_log.csv"
        self.now = datetime.now().replace(second=0, microsecond=0)
        self.config = AppConfig(
            ini_path=self.root / "tp.ini",
            settings=Settings(
                logging_enabled=True,
                log_directory=str(self.root),
                log_file_name="tp_log.csv",
            ),
            devices={
                self.mac: "Office",
                self.other_mac: "Guest Room",
            },
        )
        self._write_log(
            [
                (self.now - timedelta(minutes=3), "Office", "70.0", 40, self.mac),
                (self.now - timedelta(minutes=2), "Office", "32.0", 10, self.mac),
                (self.now - timedelta(minutes=1), "Office", "32.0", 10, self.mac),
                (self.now, "Guest Room", "80.0", 55, self.other_mac),
                (self.now - timedelta(minutes=5), "Other", "60.0", 30, "AA:BB:CC:DD:EE:FF"),
            ]
        )

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def _write_log(self, rows: list[tuple]) -> None:
        lines = [",".join(CSV_HEADER)]
        for row in rows:
            ts, device, temp, humidity, mac = row
            lines.append(
                f"{ts.strftime('%Y-%m-%d %H:%M:%S')},{device},{temp},{humidity},{mac}"
            )
        self.log_path.write_text("\n".join(lines) + "\n", encoding="utf-8")

    def test_loads_only_managed_devices(self) -> None:
        rows = load_log_rows_for_export(self.config)
        macs = {row.mac for row in rows}
        self.assertEqual(macs, {self.mac, self.other_mac})
        self.assertEqual(len(rows), 4)

    def test_payload_strips_sentinel_runs(self) -> None:
        rows = load_log_rows_for_export(self.config)
        payload = build_export_payload(self.config, rows)
        office = payload["series"][self.mac]
        self.assertEqual(len(office), 1)
        self.assertEqual(office[0]["temp_f"], 70.0)

    def test_render_html_embeds_payload_and_echarts(self) -> None:
        rows = load_log_rows_for_export(self.config)
        payload = build_export_payload(self.config, rows)
        html = render_log_export_html(payload)
        self.assertIn("echarts.min.js", html)
        self.assertIn("Office", html)
        self.assertIn('type: "time"', html)
        self.assertIn("dataZoom", html)
        self.assertIn('sampling: "lttb"', html)
        self.assertIn("Marquee zoom", html)
        embedded = html.split('id="export-data">', 1)[1].split("</script>", 1)[0]
        parsed = json.loads(embedded)
        self.assertEqual(parsed["devices"][0]["name"], "Office")

    def test_filter_series_for_hours(self) -> None:
        points = [
            {"t": (self.now - timedelta(hours=2)).strftime("%Y-%m-%dT%H:%M:%S"), "temp_f": 70.0, "humidity_pct": 40},
            {"t": (self.now - timedelta(minutes=30)).strftime("%Y-%m-%dT%H:%M:%S"), "temp_f": 71.0, "humidity_pct": 41},
        ]
        filtered = filter_series_for_hours(points, 1, end_time=self.now)
        self.assertEqual(len(filtered), 1)
        self.assertEqual(filtered[0]["temp_f"], 71.0)

    def test_export_log_to_html_writes_file(self) -> None:
        output, error = export_log_to_html(self.config)
        self.assertIsNone(error)
        assert output is not None
        self.assertTrue(output.is_file())
        self.assertIn("TemPy Log Export", output.read_text(encoding="utf-8"))

    def test_export_errors_when_log_missing(self) -> None:
        self.log_path.unlink()
        output, error = export_log_to_html(self.config)
        self.assertIsNone(output)
        self.assertIn("not found", error or "")

    def test_can_export_log_false_when_log_missing(self) -> None:
        self.log_path.unlink()
        self.assertFalse(can_export_log(self.config))

    def test_can_export_log_true_when_readings_exist(self) -> None:
        self.assertTrue(can_export_log(self.config))

    def test_can_export_log_false_when_only_sentinels(self) -> None:
        self._write_log(
            [
                (self.now - timedelta(minutes=2), "Office", "32.0", 10, self.mac),
                (self.now - timedelta(minutes=1), "Office", "32.0", 10, self.mac),
            ]
        )
        self.assertFalse(can_export_log(self.config))


if __name__ == "__main__":
    unittest.main()
