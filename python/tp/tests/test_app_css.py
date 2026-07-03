"""Tests for app-level Textual CSS."""

from __future__ import annotations

import asyncio
import unittest

from tp.config import AppConfig
from tp.ui.app import TPApp


class AppCssTests(unittest.IsolatedAsyncioTestCase):
    async def test_tpapp_css_loads_without_stylesheet_error(self) -> None:
        app = TPApp(config=AppConfig())
        async with app.run_test(size=(80, 24)):
            await asyncio.sleep(0)


if __name__ == "__main__":
    unittest.main()
