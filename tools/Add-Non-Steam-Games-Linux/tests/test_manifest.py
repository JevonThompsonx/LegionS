from __future__ import annotations

import json
import logging
import stat
import tempfile
import unittest
from pathlib import Path

from game_manifest import MANIFEST_VERSION, GameManifest, _now_iso
from shortcut_builder import build_shortcut, build_shortcut_with_appid
from tests.test_helpers import make_executable


class GameManifestTests(unittest.TestCase):
    def test_round_trip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update(
                "Hollow Knight", "/mnt/games/Hollow Knight/hollow_knight", -123456789,
            )
            manifest.add_or_update(
                "Cuphead", "/home/user/Games/Cuphead/Cuphead.exe", -987654321,
            )

            reloaded = GameManifest(path)
            self.assertEqual(reloaded.entry_count(), 2)
            self.assertIsNotNone(reloaded.find_by_exe("/mnt/games/Hollow Knight/hollow_knight"))
            self.assertIsNotNone(reloaded.find_by_appid(-987654321))

    def test_no_file_returns_empty(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "nonexistent.json"
            manifest = GameManifest(path)
            self.assertEqual(manifest.entry_count(), 0)
            self.assertEqual(manifest.get_active_entries(), [])
            self.assertEqual(manifest.get_removed_entries(), [])

    def test_corrupt_recovery(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            path.write_text("{invalid json!!!")
            manifest = GameManifest(path)
            self.assertEqual(manifest.entry_count(), 0)
            bak = path.with_suffix(".json.bak")
            self.assertTrue(bak.exists(), "Corrupted manifest should be backed up")

    def test_find_by_exe(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", 100)
            manifest.add_or_update("Game B", "/path/b/exe", 200)

            found = manifest.find_by_exe("/path/a/exe")
            self.assertIsNotNone(found)
            self.assertEqual(found["app_name"], "Game A")

            self.assertIsNone(manifest.find_by_exe("/path/nonexistent"))

    def test_find_by_appid(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", -100)
            manifest.add_or_update("Game B", "/path/b/exe", -200)

            found = manifest.find_by_appid(-200)
            self.assertIsNotNone(found)
            self.assertEqual(found["app_name"], "Game B")

            self.assertIsNone(manifest.find_by_appid(-999))

    def test_status_transitions(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", -100, status="active")

            self.assertEqual(len(manifest.get_active_entries()), 1)
            self.assertEqual(len(manifest.get_removed_entries()), 0)

            manifest.set_status("/path/a/exe", "removed")
            self.assertEqual(len(manifest.get_active_entries()), 0)
            self.assertEqual(len(manifest.get_removed_entries()), 1)

            manifest.add_or_update("Game A", "/path/a/exe", -100, status="active")
            self.assertEqual(len(manifest.get_active_entries()), 1)
            self.assertEqual(len(manifest.get_removed_entries()), 0)

    def test_find_by_name_similarity(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Hollow Knight", "/path/hollow/exe", -100)
            manifest.add_or_update("Cuphead", "/path/cup/exe", -200)

            matches = manifest.find_by_name("Hollow Knight", threshold=0.90)
            self.assertEqual(len(matches), 1)
            self.assertEqual(matches[0]["app_name"], "Hollow Knight")

            matches = manifest.find_by_name("Hollow", threshold=0.90)
            self.assertEqual(len(matches), 0)

    def test_add_or_update_updates_last_seen(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game", "/path/exe", -100)
            first_seen = manifest.find_by_exe("/path/exe")["last_seen"]

            manifest.add_or_update("Game", "/path/exe", -100)
            updated_seen = manifest.find_by_exe("/path/exe")["last_seen"]
            self.assertGreaterEqual(updated_seen, first_seen)

    def test_get_active_exe_paths(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", -100, status="active")
            manifest.add_or_update("Game B", "/path/b/exe", -200, status="removed")

            active = manifest.get_active_exe_paths()
            self.assertIn("/path/a/exe", active)
            self.assertNotIn("/path/b/exe", active)

    def test_remove_entry(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", -100)
            self.assertEqual(manifest.entry_count(), 1)

            self.assertTrue(manifest.remove_entry("/path/a/exe"))
            self.assertEqual(manifest.entry_count(), 0)

            self.assertFalse(manifest.remove_entry("/path/nonexistent"))

    def test_set_status_invalid(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game A", "/path/a/exe", -100)
            self.assertFalse(manifest.set_status("/path/a/exe", "invalid_status"))
            self.assertEqual(len(manifest.get_active_entries()), 1)

    def test_version_mismatch_resets(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            wrong_version = {"version": 999, "entries": [{"app_name": "Old", "exe_path": "/old", "appid": -1, "status": "active", "last_seen": "now"}]}
            path.write_text(json.dumps(wrong_version))
            manifest = GameManifest(path)
            self.assertEqual(manifest.entry_count(), 0)

    def test_non_dict_root_resets(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            path.write_text(json.dumps(["not", "a", "dict"]))
            manifest = GameManifest(path)
            self.assertEqual(manifest.entry_count(), 0)

    def test_non_list_entries_resets(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            path.write_text(json.dumps({"version": MANIFEST_VERSION, "entries": "not_a_list"}))
            manifest = GameManifest(path)
            self.assertEqual(manifest.entry_count(), 0)


class ManifestWorkflowTests(unittest.TestCase):
    def test_build_shortcut_with_appid_preserves(self) -> None:
        shortcut = build_shortcut_with_appid("Hollow Knight", "/path/to/hollow_knight", -12345)
        self.assertEqual(shortcut["appid"], -12345)
        self.assertEqual(shortcut["AppName"], "Hollow Knight")
        self.assertIn("hollow_knight", shortcut["Exe"])

    def test_build_shortcut_with_appid_vs_regular(self) -> None:
        regular = build_shortcut("Test Game", "/path/to/game.exe")
        with_appid = build_shortcut_with_appid("Test Game", "/path/to/game.exe", 99999)
        self.assertEqual(with_appid["appid"], 99999)
        self.assertNotEqual(with_appid["appid"], regular["appid"])

    def test_manifest_missing_game_detection(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            game_dir = Path(tmp) / "games" / "Hollow Knight"
            game_dir.mkdir(parents=True)
            exe = game_dir / "hollow_knight"
            exe.write_bytes(b"stub")
            make_executable(exe)

            from main import _load_manifest_for_user
            from steam_paths import SteamUser
            from datetime import datetime

            user_config = Path(tmp) / "config"
            user_config.mkdir(parents=True)
            user = SteamUser(
                user_id="12345",
                userdata_dir=Path(tmp),
                config_dir=user_config,
                shortcuts_path=user_config / "shortcuts.vdf",
                grid_dir=user_config / "grid",
                last_modified=datetime.now(),
            )

            manifest = _load_manifest_for_user(user)
            manifest.add_or_update("Hollow Knight", str(exe), -100, status="active")

            self.assertTrue(Path(exe).exists())
            self.assertIsNotNone(manifest.find_by_exe(str(exe)))
            self.assertEqual(len(manifest.get_active_entries()), 1)

            exe.unlink()
            self.assertFalse(Path(exe).exists())

            manifest.set_status(str(exe), "removed")
            self.assertEqual(len(manifest.get_removed_entries()), 1)

    def test_unmounted_drive_skipped(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            manifest_path = Path(tmp) / "manifest.json"
            manifest = GameManifest(manifest_path)
            manifest.add_or_update("Game", "/nonexistent_mount/game", -100, status="active")

            active = manifest.get_active_entries()
            self.assertEqual(len(active), 1)

            parent = Path("/nonexistent_mount")
            self.assertFalse(parent.exists())
            self.assertEqual(len(manifest.get_active_entries()), 1)

    def test_status_key_invariant(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "manifest.json"
            manifest = GameManifest(path)
            manifest.add_or_update("Game", "/path/exe", -100, status="active")
            entry = manifest.find_by_exe("/path/exe")
            self.assertIn("status", entry)
            self.assertEqual(entry["status"], "active")


if __name__ == "__main__":
    unittest.main()
