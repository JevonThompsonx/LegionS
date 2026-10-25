from __future__ import annotations

import json
import re
import tempfile
import time
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import quote, urlparse

from config import (
    API_KEY_ENV_NAMES,
    ARTWORK_MAX_RETRIES,
    ARTWORK_REQUEST_DELAY_SECONDS,
    ARTWORK_REQUESTS,
    ARTWORK_STYLE_PREFERENCE,
    TOOL_OR_EMULATOR_EXE_PARTS,
    TOOL_OR_EMULATOR_EXE_SUFFIXES,
    TOOL_OR_EMULATOR_NAMES,
    get_env_value,
)
from shortcut_builder import (
    build_search_aliases,
    get_shortcut_exe_value,
    get_unsigned_id,
    normalize_lookup_text,
    normalize_posix_path,
    similarity_score,
)

API_BASE_URL = "https://www.steamgriddb.com/api/v2"
ARTWORK_DOWNLOAD_HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (X11; Linux x86_64) "
        "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
    ),
    "Referer": "https://www.steamgriddb.com/",
}

# N10 (perf): on-disk SGDB API response cache. TTL-bounded so artwork metadata
# (search/autocomplete + per-game artwork listings) survives across runs without
# re-hitting the network for every refresh. Downloads are never cached — only
# the JSON metadata responses from SteamGridDB.
DEFAULT_CACHE_TTL_SECONDS = 24 * 60 * 60  # 24h


def default_cache_dir() -> Path:
    return Path(tempfile.gettempdir()) / "steamgriddb-cache"


class SGDBResponseCache:
    """TTL-bounded on-disk JSON cache for SGDB API responses.

    Keyed by the full request URL (incl. query params) so different searches and
    art-style requests are cached separately. Only successful (200) JSON payloads
    are cached; auth/rate-limit/transport errors bypass the cache entirely so a
    transient failure never poisons a good entry.
    """

    def __init__(self, cache_dir: Path | None = None, ttl_seconds: int = DEFAULT_CACHE_TTL_SECONDS):
        self.cache_dir = cache_dir
        self.ttl_seconds = ttl_seconds
        # Disabled when no cache dir is configured (e.g. tests force None).
        self.enabled = cache_dir is not None

    def _key_path(self, url: str) -> Path | None:
        if not self.enabled or self.cache_dir is None:
            return None
        safe = re.sub(r"[^A-Za-z0-9_.-]", "_", url)
        safe = safe[-200:]
        return self.cache_dir / f"{safe}.json"

    def get(self, url: str) -> dict | None:
        path = self._key_path(url)
        if path is None or not path.exists():
            return None
        try:
            raw = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            return None
        if time.time() - raw.get("cached_at", 0) > self.ttl_seconds:
            try:
                path.unlink()
            except OSError:
                pass
            return None
        return raw.get("payload")

    def put(self, url: str, payload: dict) -> None:
        path = self._key_path(url)
        if path is None:
            return
        try:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(
                json.dumps({"cached_at": time.time(), "payload": payload}, separators=(",", ":")),
                encoding="utf-8",
            )
        except OSError:
            # Best-effort cache; never fail a request because the cache is unwritable.
            pass


@dataclass(slots=True)
class ArtworkResult:
    downloaded: int = 0
    skipped: int = 0
    failures: int = 0
    downloaded_files: list[Path] = field(default_factory=list)


def _mask_api_key(value: str) -> str:
    if len(value) <= 8:
        return "*" * len(value)
    return f"{value[:4]}...{value[-4:]}"


def resolve_api_key(prompt_if_missing: bool = True) -> str | None:
    value = get_env_value(*API_KEY_ENV_NAMES)
    if value:
        return value.strip()

    if not prompt_if_missing:
        return None

    import getpass
    try:
        response = getpass.getpass("Enter your SteamGridDB API key: ").strip()
    except (KeyboardInterrupt, EOFError):
        print()
        return None
    return response or None


class SteamGridDBClient:
    def __init__(self, api_key: str, logger, response_cache: SGDBResponseCache | None = None):
        import requests

        self._requests = requests
        self.api_key = api_key
        self.logger = logger
        self.session = requests.Session()
        self.session.headers.update({"Authorization": f"Bearer {api_key}"})
        self._download_session = requests.Session()
        self._last_request_at = 0.0
        # N10: optional on-disk metadata cache; defaults to the shared temp dir.
        self.response_cache = response_cache if response_cache is not None else SGDBResponseCache(default_cache_dir())

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.close()

    def close(self) -> None:
        self.session.close()
        self._download_session.close()

    def validate_api_key(self) -> None:
        payload = self._request_json("/search/autocomplete/test")
        if payload is None:
            raise RuntimeError("SteamGridDB API validation failed. Check connectivity and confirm the API key is valid.")

    def _authorized_get(self, url: str, timeout: int):
        response = self._download_session.get(url, timeout=timeout)
        if response.status_code != 401:
            return response

        response = self._download_session.get(
            url,
            timeout=timeout,
            headers=ARTWORK_DOWNLOAD_HEADERS,
        )
        if response.status_code != 401:
            return response

        response = self.session.get(
            url,
            timeout=timeout,
            headers=ARTWORK_DOWNLOAD_HEADERS,
        )
        if response.status_code != 401:
            return response

        return self._requests.get(
            url,
            timeout=timeout,
            headers=ARTWORK_DOWNLOAD_HEADERS,
        )

    def _throttle(self) -> None:
        now = time.monotonic()
        elapsed = now - self._last_request_at
        if elapsed < ARTWORK_REQUEST_DELAY_SECONDS:
            time.sleep(ARTWORK_REQUEST_DELAY_SECONDS - elapsed)
        self._last_request_at = time.monotonic()

    def _request_json(self, endpoint: str, params: dict[str, str] | None = None) -> dict | None:
        url = f"{API_BASE_URL}{endpoint}"
        if params:
            from urllib.parse import urlencode

            url = f"{url}?{urlencode(params)}"
        # N10: serve cache hits without hitting the network.
        cached = self.response_cache.get(url)
        if cached is not None:
            self.logger.debug("SGDB cache hit for %s", url)
            return cached
        for attempt in range(ARTWORK_MAX_RETRIES):
            self._throttle()
            try:
                response = self.session.get(url, params=params, timeout=30)
            except self._requests.RequestException as error:
                self.logger.error("SteamGridDB request failed for %s: %s", url, error)
                if attempt == ARTWORK_MAX_RETRIES - 1:
                    return None
                time.sleep(2 ** attempt)
                continue

            if response.status_code == 401:
                raise RuntimeError("SteamGridDB API returned 401. Check the API key in your environment or .env file.")
            if response.status_code == 404:
                self.logger.info("SteamGridDB returned 404 for %s", url)
                return None
            if response.status_code == 429:
                wait_time = 5 * (2 ** attempt)
                self.logger.warning("SteamGridDB rate limit hit for %s; retrying in %s seconds", url, wait_time)
                time.sleep(wait_time)
                continue
            response.raise_for_status()
            payload = response.json()
            if not payload.get("success", False):
                self.logger.warning("SteamGridDB request was not successful for %s", url)
                return None
            # N10: cache successful metadata responses for subsequent runs.
            self.response_cache.put(url, payload)
            return payload
        return None

    def search_game(self, term: str) -> list[dict]:
        payload = self._request_json(f"/search/autocomplete/{quote(term, safe='')}")
        return payload.get("data", []) if payload else []

    def search_game_variants(self, terms: list[str]) -> list[dict]:
        merged: dict[int, dict] = {}
        for term in terms:
            if not term:
                continue
            for match in self.search_game(term):
                raw_id = match.get("id")
                try:
                    match_id = int(raw_id) if raw_id is not None else -1
                except (TypeError, ValueError):
                    continue
                if match_id < 0:
                    continue
                if match_id not in merged:
                    merged[match_id] = match
        return list(merged.values())

    def choose_best_match(self, app_name: str, matches: list[dict]) -> dict | None:
        if not matches:
            return None

        exact = [item for item in matches if normalize_lookup_text(str(item.get("name", ""))) == normalize_lookup_text(app_name)]
        if exact:
            verified = [item for item in exact if item.get("verified")]
            return verified[0] if verified else exact[0]

        def score(item: dict) -> tuple:
            candidate_name = str(item.get("name", "")).strip()
            normalized_candidate = normalize_lookup_text(candidate_name)
            normalized_app = normalize_lookup_text(app_name)
            sim = similarity_score(candidate_name, app_name)
            contains = normalized_app in normalized_candidate or normalized_candidate in normalized_app
            return (
                -(1 if contains else 0),
                -round(sim, 4),
                -(1 if item.get("verified") else 0),
                abs(len(candidate_name) - len(app_name)),
            )

        return sorted(matches, key=score)[0]

    def fetch_best_artwork(self, game_id: int, art_type: str) -> dict | None:
        request = ARTWORK_REQUESTS[art_type]
        if "styles" not in request["params"]:
            return self._best_of(self._request_json(request["endpoint"].format(game_id=game_id), request["params"]))
        for style in ARTWORK_STYLE_PREFERENCE:
            params = dict(request["params"])
            if style == "official":
                params.pop("styles", None)
            else:
                params["styles"] = style
            best = self._best_of(self._request_json(request["endpoint"].format(game_id=game_id), params))
            if best:
                return best
        return None

    @staticmethod
    def _best_of(payload: dict | None) -> dict | None:
        if not payload:
            return None
        data = payload.get("data", [])
        if not data:
            return None
        return sorted(data, key=lambda item: item.get("score", 0), reverse=True)[0]

    # SSRF protection: only allow https to known-good artwork domains
    _ALLOWED_SCHEMES = {"https"}
    _ALLOWED_HOST_SUFFIXES = (
        "steamgriddb.com",
        "cdn.akamai.steamstatic.com",
        "cdn.cloudflare.steamstatic.com",
    )

    def _is_safe_download_url(self, url: str) -> bool:
        """SSRF guard: validate URL scheme + host before fetch."""
        from urllib.parse import urlparse

        try:
            parsed = urlparse(url)
        except (ValueError, TypeError):
            return False

        if parsed.scheme not in self._ALLOWED_SCHEMES:
            return False
        host = (parsed.hostname or "").lower()
        if not host:
            return False
        # Reject private/loopback/link-local
        if host in ("localhost", "127.0.0.1", "0.0.0.0", "::1"):
            return False
        if host.startswith("169.254.") or host.startswith("10.") or host.startswith("192.168."):
            return False
        if host.startswith("172.") and 16 <= int(host.split(".")[1]) <= 31:
            return False
        return any(host == suffix or host.endswith("." + suffix) for suffix in self._ALLOWED_HOST_SUFFIXES)

    def download_to_file(self, url: str, destination: Path) -> bool:
        destination.parent.mkdir(parents=True, exist_ok=True)
        if not self._is_safe_download_url(url):
            self.logger.error("Blocked SSRF-unsafe artwork URL: %s", url)
            return False
        for attempt in range(ARTWORK_MAX_RETRIES):
            self._throttle()
            try:
                response = self._authorized_get(url, timeout=60)
            except self._requests.RequestException as error:
                self.logger.error("Artwork download failed for %s: %s", url, error)
                if attempt == ARTWORK_MAX_RETRIES - 1:
                    return False
                time.sleep(2 ** attempt)
                continue

            if response.status_code == 429:
                wait_time = 5 * (2 ** attempt)
                self.logger.warning("Artwork download rate-limited for %s; retrying in %s seconds", url, wait_time)
                time.sleep(wait_time)
                continue
            if response.status_code == 401:
                self.logger.error("Artwork CDN returned 401 for %s", url)
                if attempt == ARTWORK_MAX_RETRIES - 1:
                    return False
                time.sleep(2 ** attempt)
                continue
            if response.status_code == 404:
                return False
            try:
                response.raise_for_status()
            except self._requests.HTTPError as error:
                self.logger.error("Artwork download failed for %s: %s", url, error)
                return False
            temp_path: Path | None = None
            try:
                with tempfile.NamedTemporaryFile(delete=False, dir=destination.parent, suffix=".tmp") as handle:
                    handle.write(response.content)
                    temp_path = Path(handle.name)
                temp_path.replace(destination)
                return True
            except OSError as error:
                self.logger.error("Failed to persist artwork to %s: %s", destination, error)
                if temp_path is not None and temp_path.exists():
                    try:
                        temp_path.unlink()
                    except OSError:
                        pass
                return False
        return False


def _detect_extension(url: str) -> str:
    suffix = Path(urlparse(url).path).suffix.lower()
    if suffix == ".jpeg":
        return ".jpg"
    if suffix in {".png", ".jpg"}:
        return suffix
    return ".png"


def _existing_artwork(grid_dir: Path, unsigned_appid: int, art_type: str) -> Path | None:
    filename_pattern = ARTWORK_REQUESTS[art_type]["filename"].format(appid=unsigned_appid, ext="*")
    matches = sorted(grid_dir.glob(filename_pattern))
    return matches[0] if matches else None


def is_tool_or_emulator_shortcut(shortcut: dict) -> bool:
    app_name = str(shortcut.get("AppName", "")).strip()
    if app_name in TOOL_OR_EMULATOR_NAMES:
        return True

    exe_path = normalize_posix_path(get_shortcut_exe_value(shortcut))
    if not exe_path:
        return False
    path = Path(exe_path)
    if path.suffix.lower() in TOOL_OR_EMULATOR_EXE_SUFFIXES:
        return True

    parts_pattern = "|".join(re.escape(part) for part in sorted(TOOL_OR_EMULATOR_EXE_PARTS, key=len, reverse=True))
    stem = path.stem.lower()
    if re.search(rf"\b(?:{parts_pattern})\b", stem):
        return True
    for parent in path.parents:
        if re.search(rf"\b(?:{parts_pattern})\b", parent.name.lower()):
            return True
    return False


def download_artwork_for_shortcut(
    shortcut: dict, grid_dir: Path, client: SteamGridDBClient, logger, refresh: bool = False
) -> ArtworkResult:
    result = ArtworkResult()
    app_name = str(shortcut.get("AppName", "")).strip()
    if not app_name:
        logger.info("Skipping artwork because AppName is blank")
        result.skipped += 1
        return result

    if is_tool_or_emulator_shortcut(shortcut):
        logger.info("Skipping artwork for tool/emulator shortcut %s", app_name)
        result.skipped += len(ARTWORK_REQUESTS)
        return result

    unsigned_appid = get_unsigned_id(int(shortcut.get("appid", 0)))
    existing_icon = _existing_artwork(grid_dir, unsigned_appid, "icon")
    all_present = all(_existing_artwork(grid_dir, unsigned_appid, art_type) is not None for art_type in ARTWORK_REQUESTS)

    if not refresh and all_present and existing_icon:
        shortcut["icon"] = normalize_posix_path(str(existing_icon))
        result.skipped += len(ARTWORK_REQUESTS)
        logger.info("Artwork already present for %s", app_name)
        return result

    search_terms = build_search_aliases(app_name, get_shortcut_exe_value(shortcut), str(shortcut.get("StartDir", "")))
    logger.info("Artwork search terms for %s: %s", app_name, ", ".join(search_terms[:5]))
    matches = client.search_game_variants(search_terms)
    best_match = client.choose_best_match(app_name, matches)
    if not best_match:
        logger.info("No SteamGridDB match found for %s", app_name)
        result.failures += len(ARTWORK_REQUESTS)
        return result

    game_id = int(best_match["id"])
    logger.info("Using SteamGridDB match '%s' (%s) for %s", best_match.get("name", app_name), game_id, app_name)

    for art_type in ARTWORK_REQUESTS:
        existing = _existing_artwork(grid_dir, unsigned_appid, art_type)
        if existing is not None and not refresh:
            result.skipped += 1
            if art_type == "icon" and not shortcut.get("icon"):
                shortcut["icon"] = normalize_posix_path(str(existing))
            continue

        art = client.fetch_best_artwork(game_id, art_type)
        if not art or not art.get("url"):
            logger.info("No %s artwork found for %s", art_type, app_name)
            result.failures += 1
            continue

        extension = _detect_extension(str(art["url"]))
        if art_type == "horizontal":
            extension = extension.lstrip(".")
        filename = ARTWORK_REQUESTS[art_type]["filename"].format(appid=unsigned_appid, ext=extension)
        destination = grid_dir / filename

        if client.download_to_file(str(art["url"]), destination):
            result.downloaded += 1
            result.downloaded_files.append(destination)
            logger.info("Downloaded %s artwork for %s to %s", art_type, app_name, destination)
            if art_type == "icon":
                shortcut["icon"] = normalize_posix_path(str(destination))
        else:
            result.failures += 1
            logger.error("Failed to download %s artwork for %s", art_type, app_name)

    if not shortcut.get("icon"):
        existing_icon = _existing_artwork(grid_dir, unsigned_appid, "icon")
        if existing_icon is not None:
            shortcut["icon"] = normalize_posix_path(str(existing_icon))

    return result


def download_artwork_for_shortcuts(
    shortcuts: dict[str, dict], grid_dir: Path, api_key: str, logger, refresh: bool = False
) -> ArtworkResult:
    with SteamGridDBClient(api_key, logger) as client:
        logger.info("Using SteamGridDB API key %s", _mask_api_key(api_key))
        client.validate_api_key()
        total = ArtworkResult()
        grid_dir.mkdir(parents=True, exist_ok=True)

        for shortcut in shortcuts.values():
            result = download_artwork_for_shortcut(shortcut, grid_dir, client, logger, refresh=refresh)
            total.downloaded += result.downloaded
            total.skipped += result.skipped
            total.failures += result.failures
            total.downloaded_files.extend(result.downloaded_files)
    return total


def cleanup_downloaded_artwork(paths: list[Path], logger) -> None:
    for path in paths:
        try:
            if path.exists():
                path.unlink()
        except OSError as error:
            logger.warning("Failed to remove staged artwork %s: %s", path, error)
