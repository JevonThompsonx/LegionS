from __future__ import annotations

import json
import logging
import os
import shutil
import stat as _stat
import tempfile
from datetime import datetime, timezone
from pathlib import Path

from shortcut_builder import normalize_posix_path, similarity_score

MANIFEST_VERSION = 1
MANIFEST_FILENAME = "shortcuts-manifest.json"

ENTRY_REQUIRED_FIELDS = frozenset({
    "app_name", "exe_path", "appid", "status", "last_seen",
})

VALID_STATUSES = frozenset({"active", "removed"})


def _safe_int(value, default: int = 0) -> int:
    try:
        return int(value) if value is not None else default
    except (TypeError, ValueError):
        return default


def _serialize_entry(entry: dict) -> dict:
    status = str(entry.get("status", "active"))
    if status not in VALID_STATUSES:
        status = "active"
    return {
        "app_name": str(entry.get("app_name", "")),
        "exe_path": str(entry.get("exe_path", "")),
        "appid": _safe_int(entry.get("appid")),
        "icon_path": str(entry.get("icon_path", "")),
        "status": status,
        "last_seen": str(entry.get("last_seen", "")),
    }


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds") + "Z"


class GameManifest:
    def __init__(self, manifest_path: str | Path) -> None:
        self.path = Path(manifest_path)
        self._log = logging.getLogger("steam_game_manager")
        self._entries: list[dict] = []
        self._load()

    def _load(self) -> None:
        path = self.path
        if not path.exists() or path.stat().st_size == 0:
            self._entries = []
            return

        try:
            raw = path.read_text(encoding="utf-8")
            data = json.loads(raw)
        except (json.JSONDecodeError, UnicodeDecodeError, OSError) as error:
            self._log.warning("Manifest corrupt at %s (%s); creating .bak backup", path, error)
            try:
                bak_path = path.with_suffix(".json.bak")
                shutil.copy2(path, bak_path)
                self._log.info("Corrupt manifest backed up to %s", bak_path)
            except OSError as bak_error:
                self._log.warning("Could not create manifest backup: %s", bak_error)
            self._entries = []
            return

        if not isinstance(data, dict):
            self._log.warning("Manifest root is not a dict; resetting")
            self._entries = []
            return

        if data.get("version") != MANIFEST_VERSION:
            self._log.warning(
                "Manifest version %s does not match expected %s; resetting",
                data.get("version"), MANIFEST_VERSION,
            )
            self._entries = []
            return

        entries = data.get("entries", [])
        if not isinstance(entries, list):
            self._log.warning("Manifest 'entries' is not a list; resetting")
            self._entries = []
            return

        serialized: list[dict] = []
        for entry in entries:
            try:
                serialized.append(_serialize_entry(entry))
            except (TypeError, ValueError, AttributeError) as error:
                self._log.warning("Skipping malformed manifest entry: %s", error)
        self._entries = serialized

    def _save(self) -> None:
        data = {
            "version": MANIFEST_VERSION,
            "entries": self._entries,
        }
        payload = json.dumps(data, indent=2, ensure_ascii=False)
        path = self.path
        path.parent.mkdir(parents=True, exist_ok=True)
        temp_path: Path | None = None
        try:
            with tempfile.NamedTemporaryFile(
                delete=False, dir=path.parent, suffix=".tmp", mode="w", encoding="utf-8"
            ) as handle:
                handle.write(payload)
                temp_path = Path(handle.name)
            temp_path.replace(path)
            path.chmod(_stat.S_IRUSR | _stat.S_IWUSR)
        except OSError:
            if temp_path is not None and temp_path.exists():
                try:
                    temp_path.unlink()
                except OSError:
                    pass
            raise

    def _normalize_exe(self, exe_path: str) -> str:
        if not exe_path:
            return ""
        normalized = normalize_posix_path(exe_path)
        path = Path(normalized)
        if not path.is_absolute():
            return ""
        return str(path)

    @property
    def entries(self) -> list[dict]:
        return list(self._entries)

    def add_or_update(
        self,
        app_name: str,
        exe_path: str,
        appid: int,
        icon_path: str = "",
        status: str = "active",
    ) -> None:
        if status not in VALID_STATUSES:
            self._log.warning("Invalid status %r in add_or_update; defaulting to 'active'", status)
            status = "active"
        normalized = self._normalize_exe(exe_path)
        for entry in self._entries:
            if self._normalize_exe(entry.get("exe_path", "")) == normalized:
                entry.update({
                    "app_name": app_name,
                    "appid": int(appid),
                    "icon_path": icon_path,
                    "status": status,
                    "last_seen": _now_iso(),
                })
                self._save()
                return
        self._entries.append({
            "app_name": app_name,
            "exe_path": normalized or exe_path,
            "appid": int(appid),
            "icon_path": icon_path,
            "status": status,
            "last_seen": _now_iso(),
        })
        self._save()

    def set_status(self, exe_path: str, new_status: str) -> bool:
        if new_status not in VALID_STATUSES:
            return False
        normalized = self._normalize_exe(exe_path)
        for entry in self._entries:
            if self._normalize_exe(entry.get("exe_path", "")) == normalized:
                entry["status"] = new_status
                entry["last_seen"] = _now_iso()
                self._save()
                return True
        return False

    def find_by_exe(self, exe_path: str) -> dict | None:
        normalized = self._normalize_exe(exe_path)
        for entry in self._entries:
            if self._normalize_exe(entry.get("exe_path", "")) == normalized:
                return dict(entry)
        return None

    def find_by_appid(self, appid: int) -> dict | None:
        for entry in self._entries:
            if int(entry.get("appid", 0)) == appid:
                return dict(entry)
        return None

    def find_by_name(self, app_name: str, threshold: float = 0.90) -> list[dict]:
        if not app_name:
            return []
        matches: list[tuple[float, dict]] = []
        for entry in self._entries:
            entry_name = str(entry.get("app_name", ""))
            if not entry_name:
                continue
            score = similarity_score(app_name, entry_name)
            if score >= threshold:
                matches.append((score, entry))
        matches.sort(key=lambda item: item[0], reverse=True)
        return [dict(e) for _, e in matches]

    def get_active_entries(self) -> list[dict]:
        return [dict(e) for e in self._entries if e.get("status") == "active"]

    def get_removed_entries(self) -> list[dict]:
        return [dict(e) for e in self._entries if e.get("status") == "removed"]

    def get_active_exe_paths(self) -> set[str]:
        return {
            self._normalize_exe(e.get("exe_path", ""))
            for e in self._entries
            if e.get("status") == "active" and e.get("exe_path")
        }

    def remove_entry(self, exe_path: str) -> bool:
        normalized = self._normalize_exe(exe_path)
        before = len(self._entries)
        self._entries = [
            e for e in self._entries
            if self._normalize_exe(e.get("exe_path", "")) != normalized
        ]
        if len(self._entries) < before:
            self._save()
            return True
        return False

    def entry_count(self) -> int:
        return len(self._entries)
