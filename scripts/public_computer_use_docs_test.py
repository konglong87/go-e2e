#!/usr/bin/env python3
"""Regression checks for the public Computer Use documentation surface."""

from __future__ import annotations

from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
PUBLIC_DOCS = (
    ROOT / "README.md",
    ROOT / "docs/README.md",
    ROOT / "desktop-v2/README.md",
    ROOT / "docs/usage/computer-use.md",
    ROOT / "docs/plans/2026-09-27-computer-use-product-functions.md",
)
REMOVED_HISTORICAL_DOCS = (
    ROOT / "docs/plans/2026-09-26-computer-use-acceptance.md",
    ROOT / "docs/plans/2026-10-04-computer-use-autonomous-loop-p0.md",
    ROOT / "docs/reports/computer-use-acceptance-20260926.md",
)
STALE_STATUS_MARKERS = (
    "final model closure remains separate and unrun",
    "Remains open; no replay or further model retry while deterministic work is unfinished",
    "但不能据此称为全部功能稳定版验收通过",
)
PRIVATE_PATH_MARKERS = (
    "/Users/",
    "/private/var/",
    "Bearer ",
)


class PublicComputerUseDocsTest(unittest.TestCase):
    def test_current_public_entry_points_exist(self) -> None:
        for path in PUBLIC_DOCS:
            self.assertTrue(path.is_file(), f"missing public documentation entry point: {path}")

        guide = (ROOT / "docs/usage/computer-use.md").read_text(encoding="utf-8")
        for marker in (
            "Developer Preview",
            "macOS",
            "single-display",
            "Developer ID signing",
            "second physical display",
        ):
            self.assertIn(marker, guide, f"Computer Use guide is missing scope marker: {marker}")

    def test_public_docs_do_not_publish_stale_completion_status(self) -> None:
        for path in PUBLIC_DOCS:
            content = path.read_text(encoding="utf-8")
            for marker in STALE_STATUS_MARKERS:
                self.assertNotIn(marker, content, f"stale status in {path}: {marker}")

    def test_public_reader_docs_do_not_contain_private_local_details(self) -> None:
        for path in PUBLIC_DOCS:
            content = path.read_text(encoding="utf-8")
            for marker in PRIVATE_PATH_MARKERS:
                self.assertNotIn(marker, content, f"private detail in {path}: {marker}")

    def test_superseded_acceptance_documents_are_not_active(self) -> None:
        for path in REMOVED_HISTORICAL_DOCS:
            self.assertFalse(path.exists(), f"superseded historical document remains active: {path}")


if __name__ == "__main__":
    unittest.main()
