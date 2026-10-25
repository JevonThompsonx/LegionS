from __future__ import annotations

import binascii
import os
import re
import struct
from difflib import SequenceMatcher
from functools import lru_cache
from pathlib import Path

from config import GENERIC_CONTAINER_DIRS, REQUIRED_SHORTCUT_DEFAULTS, SYSTEM_PATH_PARTS

RELEASE_TAG_PATTERNS = [
    r"-gog$",
    r"-codex$",
    r"-plaza$",
    r"-fitgirl$",
    r"-repack$",
    r"\(gog\)$",
    r"\(epic\)$",
    r"\[gog\]$",
]

NOISE_PATTERNS = [
    r"[-_. ]*steamgg[-_. ]*net\b",
    r"[-_. ]*steamrip[-_. ]*com\b",
    r"\bsteamgg(?:\.net)?\b",
    r"\bsteamrip(?:\.com)?\b",
    r"\bfitgirl(?:-repacks(?:\.site)?)?\b",
    r"\bcodex\b",
    r"\bprophet\b",
    r"\bgoldberg(?:\s+emu)?\b",
    r"\breloaded\b",
    r"\brune\b",
    r"\btenoke\b",
    r"\bflt\b",
    r"\bportable\b",
    r"\bno\s*dvd\b",
    r"\bdx\s*11\b",
    r"\bdx\s*12\b",
    r"\bsteam\b",
    r"\blinux\b",          # strip "linux" suffix from folder names
    r"\bx86_64\b",
    r"\bx86\b",
    r"\bx64\b",
]

TRAILING_NOISE_PATTERNS = [
    r"\blauncher\b$",
    r"\bgame\b$",
]

ROMAN_NUMERALS = {
    "1": "I",
    "2": "II",
    "3": "III",
    "4": "IV",
    "5": "V",
    "6": "VI",
    "7": "VII",
    "8": "VIII",
    "9": "IX",
    "10": "X",
}

ROMAN_TO_ARABIC = {value: key for key, value in ROMAN_NUMERALS.items()}

# ── Precompiled regexes ──────────────────────────────────────────────────────
# Second audit: clean_game_name ran ~62 regex applications/call (16 explicit
# re.sub + 2 loops over 31 patterns), none precompiled. Compiling once at module
# scope removes all per-call pattern parsing.
_RELEASE_TAG_RE = [re.compile(p, re.IGNORECASE) for p in RELEASE_TAG_PATTERNS]
_NOISE_RE = [re.compile(p, re.IGNORECASE) for p in NOISE_PATTERNS]
_TRAILING_NOISE_RE = [re.compile(p, re.IGNORECASE) for p in TRAILING_NOISE_PATTERNS]

_NON_ALNUM_RE = re.compile(r"[^a-z0-9]+")

# clean_game_name inline passes
_CGN_STEAMRIP_RE = re.compile(r"[-_. ]*steamrip(?:\.com)?", re.IGNORECASE)
_CGN_STEAMGG_RE = re.compile(r"[-_. ]*steamgg(?:\.net)?", re.IGNORECASE)
_CGN_CAMEL_RE = re.compile(r"(?<=[a-z])(?=[A-Z]{2,}\b)")
_CGN_UCAMEL_RE = re.compile(r"(?<=[A-Z])(?=[A-Z][a-z])")
_CGN_VER_RE = re.compile(r"[-_.]v?\d+(?:\.\d+)+.*$", re.IGNORECASE)
_CGN_LCAMEL_RE = re.compile(r"(?<=[a-z])(?=[A-Z])")
_CGN_ALNUM_DIGIT_RE = re.compile(r"(?<=[A-Za-z])(?=\d)")
_CGN_DIGIT_ALNUM_RE = re.compile(r"(?<=\d)(?=[A-Za-z])")
_CGN_UUCAMEL_RE = re.compile(r"(?<=[A-Z])(?=[A-Z][A-Z][a-z])")
_CGN_DOT_RE = re.compile(r"\.(?!\d)")
_CGN_DASH_RE = re.compile(r"\s+-\s+")
_CGN_COLON_RE = re.compile(r"\s*[:]\s*")
_CGN_WS_RE = re.compile(r"\s+")

# _smart_title roman-numeral passes
_SMART_TITLE_ROMAN_RE = [
    (re.compile(r"\bIi\b"), "II"),
    (re.compile(r"\bIii\b"), "III"),
    (re.compile(r"\bIv\b"), "IV"),
    (re.compile(r"\bVi\b"), "VI"),
    (re.compile(r"\bVii\b"), "VII"),
    (re.compile(r"\bViii\b"), "VIII"),
    (re.compile(r"\bIx\b"), "IX"),
]

# build_search_aliases roman/arabic variant expansion
_ROMAN_TO_ARABIC_RE = [(re.compile(rf"\b{roman}\b"), arabic) for roman, arabic in ROMAN_TO_ARABIC.items()]
_ARABIC_TO_ROMAN_RE = [(re.compile(rf"\b{arabic}\b"), roman) for arabic, roman in ROMAN_NUMERALS.items()]

MANUAL_TITLE_ALIASES = {
    "acodyssey": ["Assassin's Creed Odyssey"],
    "ac odyssey": ["Assassin's Creed Odyssey"],
    "digimon world next order": ["Digimon World: Next Order"],
    "ghost of tsushima dc": ["Ghost of Tsushima Director's Cut"],
    "horizon zero dawn": ["Horizon Zero Dawn"],
    "lego the incredibles": ["LEGO The Incredibles"],
    "smt iii nocturne hd remaster": ["Shin Megami Tensei III: Nocturne - HD Remaster"],
    "ys viii lacrimosa of dana": ["Ys VIII: Lacrimosa of Dana"],
    # ── Games on this CachyOS system ─────────────────────────────────────────
    "absolum": ["Absolum"],
    # BALLxPIT — exe stem "Balls" weakly matches dir name; alias ensures correct title
    "ballxpit": ["BALLxPIT"],
    "celeste": ["Celeste"],
    "chrono trigger steamrip com": ["Chrono Trigger"],
    "chrono trigger steamrip": ["Chrono Trigger"],
    "chrono trigger": ["Chrono Trigger"],
    "constance": ["Constance"],
    "cross blitz early access": ["Cross Blitz"],
    "cross blitz": ["Cross Blitz"],
    "cuphead": ["Cuphead"],
    "disco elysium": ["Disco Elysium"],
    "dragon quest i ii steamgg net": ["Dragon Quest I & II"],
    "dragon quest i ii": ["Dragon Quest I & II"],
    "dragon quest iii hd 2d remake": ["Dragon Quest III HD-2D Remake"],
    "etrian odyssey origins collection": ["Etrian Odyssey Origins Collection"],
    # Final Fantasy Pixel Remaster — folder names use the "PR" abbreviation
    "final fantasy i pr": ["Final Fantasy I Pixel Remaster"],
    "final fantasy ii pr": ["Final Fantasy II Pixel Remaster"],
    "final fantasy iii pr": ["Final Fantasy III Pixel Remaster"],
    "final fantasy iv pr": ["Final Fantasy IV Pixel Remaster"],
    "final fantasy v pr": ["Final Fantasy V Pixel Remaster"],
    "final fantasy vi pr": ["Final Fantasy VI Pixel Remaster"],
    "furi": ["Furi"],
    "hollow knight silksong": ["Hollow Knight: Silksong"],
    "monster hunter stories": ["Monster Hunter Stories"],
    "neon abyss 2": ["Neon Abyss 2"],
    # Possessor(s) — parentheses survive clean_game_name; alias restores them
    "possessor s": ["Possessor(s)"],
    "possessor": ["Possessor(s)"],
    "risk of rain 2": ["Risk of Rain 2"],
    "shape of dreams": ["Shape of Dreams"],
    # Suikoden — folder name uses "&", exe stem uses "and"
    "suikoden i and ii hd remaster": ["Suikoden I & II HD Remaster"],
    "suikoden i ii hd remaster": ["Suikoden I & II HD Remaster"],
    # Ys I / II — GOG releases; long subtitles from the alias
    "ys i": ["Ys I: Ancient Ys Vanished"],
    "ys ii": ["Ys II: Ancient Ys Vanished – The Final Chapter"],
}

# Curated official titles keyed by normalize_lookup_text(app_name). Checked
# BEFORE MANUAL_TITLE_ALIASES in clean_game_name() so every shortcut resolves to
# its exact official/SGDB title. Covers all game shortcuts on this system;
# tool/emulator shortcuts and entry "Octopath Traveler 0" (Citron launcher) are
# intentionally absent.
OFFICIAL_TITLE_ALIASES: dict[str, str] = {
    "alabaster dawn": "Alabaster Dawn",
    "card en ciel": "Card-en-Ciel",
    "cuphead": "Cuphead",
    "danganronpa v3 killing harmony": "Danganronpa V3: Killing Harmony",
    "deltarune": "DELTARUNE",
    "disgaea 7 vows of the virtueless": "Disgaea 7: Vows of the Virtueless",
    "dispatch": "Dispatch",
    "eiyuden chronicle rising": "Eiyuden Chronicle: Rising",
    "esoteric ebb": "Esoteric Ebb",
    "etrian odyssey": "Etrian Odyssey",
    "etrian odyssey 2": "Etrian Odyssey II",
    "etrian odyssey 3": "Etrian Odyssey III",
    "rune factory guardians of azuma": "Rune Factory: Guardians of Azuma",
    "final fantasy iii": "FINAL FANTASY III",
    "final fantasy x x 2 hd remaster": "Final Fantasy X/X-2 HD Remaster",
    "furi": "Furi",
    "ghost trick phantom detective": "Ghost Trick: Phantom Detective",
    "haste": "HASTE",
    "hollow knight": "Hollow Knight",
    "hollow knight silksong": "Hollow Knight: Silksong",
    "jedi survivor": "STAR WARS Jedi: Survivor",
    "lossless scaling": "Lossless Scaling",
    "marvel cosmic invasion": "MARVEL Cosmic Invasion",
    "crypt of the necrodancer": "Crypt of the NecroDancer",
    "nine sols": "Nine Sols",
    "octopath traveler": "OCTOPATH TRAVELER",
    "of ash and steel": "Of Ash and Steel",
    "ogu and the secret forest": "Ogu and the Secret Forest",
    "peak": "PEAK",
    "portal": "Portal",
    "portal 2": "Portal 2",
    "pragmata": "PRAGMATA",
    "schedule i": "Schedule I",
    "slime rancher": "Slime Rancher",
    "trails beyond the horizon": "The Legend of Heroes: Trails beyond the Horizon",
    "tactics ogre reborn": "Tactics Ogre: Reborn",
    "tales of xillia remastered": "Tales of Xillia Remastered",
    "the great ace attorney chronicles": "The Great Ace Attorney Chronicles",
    "the binding of isaac repentance": "The Binding of Isaac: Repentance",
    "the last of us": "The Last of Us Part I",
    "unbeatable": "UNBEATABLE",
    "hades": "Hades",
    "dragon marked for death": "Dragon Marked for Death",
    "elden ring": "Elden Ring",
    "elden ring nightreign": "ELDEN RING NIGHTREIGN",
    "fez": "FEZ",
    "it takes two": "It Takes Two",
    "octopath traveler ii": "OCTOPATH TRAVELER II",
    "tales of arise": "Tales of Arise",
    "the witcher 3 wild hunt": "The Witcher 3: Wild Hunt",
    "beastieball": "Beastieball",
    "dragon quest i ii hd 2d remake": "Dragon Quest I & II HD-2D Remake",
    "spiritfall": "Spiritfall",
    "mega man battle network legacy vol 2": "Mega Man Battle Network Legacy Collection Vol. 2",
    "absolum": "Absolum",
    "risk of rain 2": "Risk of Rain 2",
    "flowstone saga": "Flowstone Saga",
    "split fiction": "Split Fiction",
}


def official_title_for(app_name: str) -> str | None:
    return OFFICIAL_TITLE_ALIASES.get(normalize_lookup_text(app_name))


@lru_cache(maxsize=4096)
def normalize_lookup_text(value: str) -> str:
    text = _NON_ALNUM_RE.sub(" ", value.lower())
    return " ".join(text.split())


def _smart_title(value: str) -> str:
    words: list[str] = []
    small_words = {"a", "an", "and", "of", "the", "to", "for", "in"}
    for index, raw_word in enumerate(value.split()):
        lowered = raw_word.lower()
        if raw_word.isupper() and len(raw_word) <= 6:
            words.append(raw_word)
            continue
        if lowered in small_words and index != 0:
            words.append(lowered)
            continue
        if lowered == "vs":
            words.append("vs.")
            continue
        words.append(lowered.capitalize())
    titled = " ".join(words)
    for pattern, replacement in _SMART_TITLE_ROMAN_RE:
        titled = pattern.sub(replacement, titled)
    return titled


def normalize_posix_path(path: str) -> str:
    """Normalize a POSIX path string without ever converting forward slashes.

    ``pathlib.Path`` converts ``/`` to ``\\`` on Windows, which is wrong here
    because this code always produces Linux paths (even when tested on Windows).
    We therefore do all normalization at the string level.
    """
    value = str(path or "").strip().strip('"').strip()
    if not value:
        return ""
    # Expand ~ at the string level so we avoid Path() slash conversion
    if value.startswith("~/") or value == "~":
        home = Path.home().as_posix()
        value = home + value[1:]
    # Collapse double slashes but do NOT call Path() — keep forward slashes
    while "//" in value:
        value = value.replace("//", "/")
    return value


def quote_posix_path(path: str, trailing_slash: bool = False) -> str:
    normalized = normalize_posix_path(path)
    if not normalized:
        return ""
    if trailing_slash and not normalized.endswith("/"):
        normalized = f"{normalized}/"
    return f'"{normalized}"'


def unquote_posix_path(path: str) -> str:
    return normalize_posix_path(path)


def has_glob_pattern(path: str) -> bool:
    value = str(path or "")
    return any(token in value for token in ("*", "?", "[", "]", "(", ")", "|"))


def is_concrete_exe_path(path: str) -> bool:
    """On Linux, a concrete exe path is any absolute path that doesn't contain globs."""
    value = unquote_posix_path(path)
    if not value:
        return False
    if has_glob_pattern(value):
        return False
    # Accept absolute paths to any file (no .exe requirement on Linux)
    return value.startswith("/") or value.startswith("~")


def generate_shortcut_id(exe: str, app_name: str) -> int:
    unique_id = f"{exe}{app_name}"
    crc = binascii.crc32(unique_id.encode("utf-8")) & 0xFFFFFFFF
    shortcut_id = crc | 0x80000000
    return struct.unpack("i", struct.pack("I", shortcut_id))[0]


def get_unsigned_id(signed_appid: int) -> int:
    return struct.unpack("I", struct.pack("i", int(signed_appid)))[0]


@lru_cache(maxsize=4096)
def clean_game_name(folder_name: str) -> str:
    name = folder_name.strip()
    # Strip distribution-site suffixes BEFORE camelCase splitting so patterns
    # like "SteamRIP.com" and "SteamGG.NET" are removed while still contiguous.
    name = _CGN_STEAMRIP_RE.sub("", name)
    name = _CGN_STEAMGG_RE.sub("", name)
    name = _CGN_CAMEL_RE.sub(" ", name)
    for pattern in _RELEASE_TAG_RE:
        name = pattern.sub("", name)
    name = _CGN_UCAMEL_RE.sub(" ", name)
    name = _CGN_VER_RE.sub("", name)
    name = _CGN_LCAMEL_RE.sub(" ", name)
    name = _CGN_ALNUM_DIGIT_RE.sub(" ", name)
    name = _CGN_DIGIT_ALNUM_RE.sub(" ", name)
    name = _CGN_UUCAMEL_RE.sub(" ", name)
    name = name.replace("_", " ")
    name = _CGN_DOT_RE.sub(" ", name)
    name = _CGN_DASH_RE.sub(" - ", name)
    for pattern in _NOISE_RE:
        name = pattern.sub(" ", name)
    for pattern in _TRAILING_NOISE_RE:
        name = pattern.sub("", name)
    name = _CGN_COLON_RE.sub(": ", name)
    name = _CGN_WS_RE.sub(" ", name).strip(" -_.")
    normalized_lookup = normalize_lookup_text(name)
    official_title = OFFICIAL_TITLE_ALIASES.get(normalized_lookup)
    if official_title:
        return official_title
    manual_aliases = MANUAL_TITLE_ALIASES.get(normalized_lookup)
    if manual_aliases:
        return manual_aliases[0]
    if name:
        name = _smart_title(name)
    return name or folder_name.strip()


def build_search_aliases(app_name: str, exe_path: str = "", start_dir: str = "") -> list[str]:
    candidates: list[str] = []

    def add(value: str) -> None:
        cleaned = clean_game_name(value)
        if not cleaned:
            return
        if cleaned not in candidates:
            candidates.append(cleaned)

    add(app_name)

    normalized_app_name = normalize_lookup_text(app_name)
    for alias in MANUAL_TITLE_ALIASES.get(normalized_app_name, []):
        add(alias)

    for raw_path in (exe_path, start_dir):
        normalized_exe = unquote_posix_path(raw_path)
        if not normalized_exe:
            continue
        path = Path(normalized_exe)
        # Strip common extensions
        stem = path.stem
        if path.suffix.lower() in (".sh", ".exe", ".bin", ".run", ".x86_64", ".x86"):
            add(stem)
        for parent in [path.parent, *path.parents[1:3]]:
            if not parent.name:
                continue
            if parent.name.lower() in GENERIC_CONTAINER_DIRS:
                continue
            if parent.name.lower() in SYSTEM_PATH_PARTS:
                continue
            add(parent.name)

    expanded: list[str] = []
    for candidate in candidates:
        if candidate not in expanded:
            expanded.append(candidate)
        normalized_candidate = normalize_lookup_text(candidate)
        for alias in MANUAL_TITLE_ALIASES.get(normalized_candidate, []):
            if alias not in expanded:
                expanded.append(alias)
        arabic_variant = candidate
        for pattern, arabic in _ROMAN_TO_ARABIC_RE:
            arabic_variant = pattern.sub(arabic, arabic_variant)
        if arabic_variant != candidate and arabic_variant not in expanded:
            expanded.append(arabic_variant)
        roman_variant = candidate
        for pattern, roman in _ARABIC_TO_ROMAN_RE:
            roman_variant = pattern.sub(roman, roman_variant)
        if roman_variant != candidate and roman_variant not in expanded:
            expanded.append(roman_variant)

    return expanded


def is_probably_invalid_app_name(name: str) -> bool:
    value = (name or "").strip()
    if not value:
        return True
    alnum_count = sum(character.isalnum() for character in value)
    return alnum_count == 0


def derive_app_name_from_path(exe_path: str) -> str:
    normalized = unquote_posix_path(exe_path)
    if not normalized:
        return "Unknown Game"
    path = Path(normalized)
    for parent in [path.parent, *path.parents[1:]]:
        if not parent.name:
            continue
        if parent.name.lower() in GENERIC_CONTAINER_DIRS:
            continue
        return clean_game_name(parent.name)
    return clean_game_name(path.stem)


def prettify_exe_stem(stem: str) -> str:
    value = clean_game_name(stem)
    manual = MANUAL_TITLE_ALIASES.get(normalize_lookup_text(value))
    if manual:
        return manual[0]
    return value


def get_shortcut_exe_value(shortcut: dict) -> str:
    for key in ("exe", "Exe"):
        value = str(shortcut.get(key, "")).strip()
        if value:
            return value
    return ""


def get_shortcut_openvr_value(shortcut: dict) -> int:
    for key in ("openvr", "OpenVR"):
        value = shortcut.get(key)
        if value is not None:
            try:
                return int(value)
            except (TypeError, ValueError):
                return 0
    return 0


def normalize_tags(tags: object) -> dict[str, str]:
    if isinstance(tags, dict):
        ordered_values = [str(value) for _, value in sorted(tags.items(), key=lambda item: str(item[0])) if str(value).strip()]
    elif isinstance(tags, (list, tuple, set)):
        ordered_values = [str(value) for value in tags if str(value).strip()]
    else:
        ordered_values = []
    return {str(index): value for index, value in enumerate(ordered_values)}


def build_shortcut(app_name: str, exe_path: str, icon_path: str = "", launch_options: str = "") -> dict:
    normalized_exe = normalize_posix_path(exe_path)
    # Derive parent directory via string split to avoid Path() converting / to \ on Windows
    if normalized_exe and "/" in normalized_exe:
        start_dir_str = normalized_exe.rsplit("/", 1)[0] + "/"
    elif normalized_exe:
        start_dir_str = "./"
    else:
        start_dir_str = "./"
    quoted_exe = quote_posix_path(normalized_exe)
    quoted_start_dir = f'"{start_dir_str}"' if start_dir_str else ""
    final_name = clean_game_name(app_name) if app_name else derive_app_name_from_path(normalized_exe)
    appid = generate_shortcut_id(quoted_exe, final_name)

    shortcut = {
        "appid": appid,
        "AppName": final_name,
        "Exe": quoted_exe,
        "StartDir": quoted_start_dir,
        "icon": normalize_posix_path(icon_path),
        "sortas": "",
        "tags": {},
    }
    shortcut.update(REQUIRED_SHORTCUT_DEFAULTS)
    if launch_options:
        shortcut["LaunchOptions"] = launch_options
    return shortcut


def build_shortcut_with_appid(app_name: str, exe_path: str, original_appid: int, icon_path: str = "", launch_options: str = "") -> dict:
    shortcut = build_shortcut(app_name, exe_path, icon_path, launch_options)
    shortcut["appid"] = int(original_appid)
    return shortcut


def normalize_shortcut(shortcut: dict) -> dict:
    original = dict(shortcut)
    exe_path = unquote_posix_path(get_shortcut_exe_value(original))
    app_name = str(original.get("AppName", "")).strip()
    if is_probably_invalid_app_name(app_name):
        app_name = derive_app_name_from_path(exe_path or str(original.get("ShortcutPath", "")))
    app_name = clean_game_name(app_name)

    normalized = dict(original)
    normalized["AppName"] = app_name
    if is_concrete_exe_path(get_shortcut_exe_value(original)):
        normalized["Exe"] = quote_posix_path(exe_path)
    else:
        normalized["Exe"] = get_shortcut_exe_value(original)
    normalized.pop("exe", None)

    if is_concrete_exe_path(get_shortcut_exe_value(original)) and exe_path:
        normalized["StartDir"] = quote_posix_path(str(Path(exe_path).parent), trailing_slash=True)
    else:
        start_dir = unquote_posix_path(str(original.get("StartDir", "")))
        normalized["StartDir"] = quote_posix_path(start_dir, trailing_slash=True) if start_dir else ""

    normalized["icon"] = normalize_posix_path(str(original.get("icon", "")))
    normalized["sortas"] = str(original.get("sortas", ""))
    normalized["ShortcutPath"] = str(original.get("ShortcutPath", REQUIRED_SHORTCUT_DEFAULTS["ShortcutPath"]))
    normalized["LaunchOptions"] = str(original.get("LaunchOptions", REQUIRED_SHORTCUT_DEFAULTS["LaunchOptions"]))

    for field, default in REQUIRED_SHORTCUT_DEFAULTS.items():
        if field in {"ShortcutPath", "LaunchOptions"}:
            continue
        value = original.get(field, default)
        if isinstance(default, int):
            try:
                normalized[field] = int(value)
            except (TypeError, ValueError):
                normalized[field] = default
        else:
            normalized[field] = str(value)

    normalized["tags"] = normalize_tags(original.get("tags", {}))
    normalized["OpenVR"] = get_shortcut_openvr_value(original)
    normalized.pop("openvr", None)

    try:
        normalized["appid"] = int(original.get("appid", 0))
    except (TypeError, ValueError):
        quoted_exe = normalized.get("Exe", "")
        if quoted_exe and normalized["AppName"]:
            normalized["appid"] = generate_shortcut_id(quoted_exe, normalized["AppName"])
        else:
            normalized["appid"] = 0

    return normalized


def normalized_exe_identity(exe_or_shortcut: str | dict) -> str:
    if isinstance(exe_or_shortcut, dict):
        value = get_shortcut_exe_value(exe_or_shortcut)
    else:
        value = str(exe_or_shortcut)
    return unquote_posix_path(value)


def shortcut_completeness_score(shortcut: dict) -> int:
    score = 0
    app_name = str(shortcut.get("AppName", "")).strip()
    exe_value = get_shortcut_exe_value(shortcut).strip()
    start_dir = str(shortcut.get("StartDir", "")).strip()

    if app_name and not is_probably_invalid_app_name(app_name):
        score += 3
    if exe_value:
        score += 3
    if exe_value.startswith('"') and exe_value.endswith('"'):
        score += 1
    if start_dir:
        score += 2
    if start_dir.startswith('"') and start_dir.endswith('"'):
        score += 1
    if shortcut.get("AllowDesktopConfig") == 1:
        score += 1
    if shortcut.get("AllowOverlay") == 1:
        score += 1
    if get_shortcut_openvr_value(shortcut) == 0:
        score += 1
    if shortcut.get("icon"):
        score += 1

    exe_path = unquote_posix_path(exe_value)
    if exe_path and Path(exe_path).exists():
        score += 3
    return score


@lru_cache(maxsize=8192)
def similarity_score(left: str, right: str) -> float:
    normalized_left = _NON_ALNUM_RE.sub("", left.lower())
    normalized_right = _NON_ALNUM_RE.sub("", right.lower())
    if not normalized_left or not normalized_right:
        return 0.0
    return SequenceMatcher(None, normalized_left, normalized_right).ratio()
