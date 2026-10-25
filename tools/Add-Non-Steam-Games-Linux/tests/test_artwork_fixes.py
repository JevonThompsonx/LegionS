from __future__ import annotations

import logging
import tempfile
import unittest
from pathlib import Path

from artwork_manager import (
    ARTWORK_REQUESTS,
    SteamGridDBClient,
    download_artwork_for_shortcut,
    is_tool_or_emulator_shortcut,
    _existing_artwork,
)
from fixer import diagnose_shortcuts
from shortcut_builder import build_shortcut, build_search_aliases, normalize_shortcut


class _RequestLoggingClient(SteamGridDBClient):
    """Records style keys requested by fetch_best_artwork (official = no styles param)."""

    def __init__(self, official_data: bool = False) -> None:
        self.styles: list[str | None] = []
        self.official_data = official_data

    def _request_json(self, endpoint: str, params: dict | None = None) -> dict | None:
        style = (params or {}).get("styles")
        self.styles.append(style)
        if style is None and not self.official_data:
            return {"success": True, "data": []}
        return {"success": True, "data": [{"score": 9, "url": "https://example.com/art.png", "id": 1}]}


class ArtworkFixesHorizontalGlobTests(unittest.TestCase):
    def test_existing_artwork_horizontal_ignores_suffixed_variants(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            grid_dir = Path(temp_dir)
            for name in ("123_hero.png", "123_logo.png", "123_icon.png", "123p.png"):
                (grid_dir / name).write_bytes(b"art")

            found = _existing_artwork(grid_dir, 123, "horizontal")
            self.assertIsNone(found, "suffix variants must not match horizontal artwork")

    def test_existing_artwork_horizontal_finds_plain_appid_file(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            grid_dir = Path(temp_dir)
            (grid_dir / "123.png").write_bytes(b"art")

            found = _existing_artwork(grid_dir, 123, "horizontal")
            self.assertIsNotNone(found)
            self.assertEqual(found.name, "123.png")


class ArtworkFixesStylePreferenceTests(unittest.TestCase):
    def test_fetch_best_artwork_tries_official_then_alternate(self) -> None:
        client = _RequestLoggingClient()
        art = SteamGridDBClient.fetch_best_artwork(client, 42, "horizontal")

        self.assertEqual(client.styles, [None, "alternate"])
        self.assertEqual(art["url"], "https://example.com/art.png")

    def test_fetch_best_artwork_stops_at_official(self) -> None:
        client = _RequestLoggingClient(official_data=True)
        art = SteamGridDBClient.fetch_best_artwork(client, 42, "portrait")

        self.assertEqual(client.styles, [None])
        self.assertEqual(art["url"], "https://example.com/art.png")


class ArtworkFixesSearchTermsTests(unittest.TestCase):
    def test_search_aliases_exclude_system_path_parts(self) -> None:
        aliases = build_search_aliases("Super Adventure", "/run/media/system/Games_Data/Super Adventure/super_adventure.exe")

        lowered = [value.lower() for value in aliases]
        self.assertNotIn("games data", lowered)
        self.assertNotIn("system", lowered)
        self.assertNotIn("media", lowered)
        self.assertIn("Super Adventure", aliases)


class ArtworkFixesToolSkipTests(unittest.TestCase):
    def test_is_tool_or_emulator_shortcut_detects_appimage_exe(self) -> None:
        shortcut = build_shortcut("Test Tool", "/var/home/jevonx/.local/bin/Ascendara.AppImage")
        self.assertTrue(is_tool_or_emulator_shortcut(shortcut))

    def test_is_tool_or_emulator_shortcut_detects_name(self) -> None:
        shortcut = build_shortcut("Linux-Yuzu", "/run/media/system/Games_Data/Linux-Yuzu-EA-4176.AppImage")
        self.assertTrue(is_tool_or_emulator_shortcut(shortcut))

    def test_is_tool_or_emulator_shortcut_detects_citron_launcher(self) -> None:
        shortcut = build_shortcut("Octopath Traveler 0", "/run/media/system/Games_Data/OCTOPATH TRAVELER 0/citron-cmd.exe")
        self.assertTrue(is_tool_or_emulator_shortcut(shortcut))

    def test_download_artwork_for_shortcut_skips_tool(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            shortcut = build_shortcut("Switch Citron", "/run/media/system/Games_Data/citron_nightly.AppImage")
            dummy_client = SteamGridDBClient.__new__(SteamGridDBClient)
            result = download_artwork_for_shortcut(shortcut, Path(temp_dir), dummy_client, logging.getLogger("test"))

            self.assertEqual(result.downloaded, 0)
            self.assertEqual(result.failures, 0)
            self.assertEqual(result.skipped, len(ARTWORK_REQUESTS))


class ArtworkFixesRenameTests(unittest.TestCase):
    def test_normalize_shortcut_applies_official_title_and_preserves_appid(self) -> None:
        shortcut = build_shortcut("Card-en-ciel", "/run/media/system/Games_Data/Card-en-Ciel/Card-en-Ciel.exe")
        normalized = normalize_shortcut(shortcut)

        self.assertEqual(normalized["AppName"], "Card-en-Ciel")
        self.assertEqual(normalized["appid"], shortcut["appid"])

    def test_normalize_shortcut_hastens_official_title_and_preserves_appid(self) -> None:
        shortcut = build_shortcut("Haste", "/run/media/system/Games_Data/Haste.v1.7.c/Haste.exe")
        normalized = normalize_shortcut(shortcut)

        self.assertEqual(normalized["AppName"], "HASTE")
        self.assertEqual(normalized["appid"], shortcut["appid"])


class ArtworkFixesDiagnoseTests(unittest.TestCase):
    def test_diagnose_flags_non_official_title_and_skips_tools(self) -> None:
        wrong = build_shortcut("Card-en-Ciel", "/run/media/system/Games_Data/Card-en-Ciel/Card-en-Ciel.exe")
        wrong["AppName"] = "Card-en-ciel"
        tools = {
            "59": build_shortcut("Ascendara.AppImage", "/var/home/jevonx/.local/bin/Ascendara.AppImage"),
            "60": build_shortcut("Linux-Yuzu", "/run/media/system/Games_Data/Linux-Yuzu-EA-4176.AppImage"),
            "61": build_shortcut("Eden-Linux", "/run/media/system/Games_Data/Eden-Linux-v0.2.0.AppImage"),
            "62": build_shortcut("Switch Citron", "/run/media/system/Games_Data/citron_nightly.AppImage"),
        }
        launcher = build_shortcut("Octopath Traveler 0", "/run/media/system/Games_Data/OCTOPATH TRAVELER 0/citron-cmd.exe")
        shortcuts_map = {"0": wrong, **tools, "26": launcher}

        issues = diagnose_shortcuts(shortcuts_map)

        self.assertIn("0", issues)
        self.assertTrue(any("AppName not official title" in issue for issue in issues["0"]))
        for index in ("59", "60", "61", "62", "26"):
            flagged = any("AppName not official title" in issue for issue in issues.get(index, []))
            self.assertFalse(flagged, f"entry {index} must not be flagged for non-official title")


if __name__ == "__main__":
    unittest.main()
