#!/usr/bin/env python3
"""Non-interactive wrapper — auto-selects ALL games."""
import sys
sys.path.insert(0, "/run/media/system/Games_Data/Add-Non-Steam-Games-Linux")

from config import LOG_FILE_NAME
from logger_setup import setup_logging
from steam_paths import find_steam_install_path, list_steam_users
from main import scan_and_add_games, download_all_artwork
from game_scanner import DiscoveredGame

# Monkey-patch select_games_to_add to be non-interactive
import main as main_mod

def auto_select_games(candidates):
    """Auto-select all games, picking first candidate for ambiguous ones."""
    if not candidates:
        print("No new non-Steam game candidates were found.")
        return []
    
    confirmed = []
    for c in candidates:
        if c.ambiguous and c.candidates:
            # Pick the first candidate for ambiguous games
            from game_scanner import DiscoveredGame
            resolved = DiscoveredGame(
                app_name=c.app_name,
                exe_path=c.candidates[0],
                source_dir=c.source_dir,
                score=c.score,
                ambiguous=False,
                candidates=c.candidates,
            )
            confirmed.append(resolved)
            print(f"  [auto] {c.app_name} -> {c.candidates[0]}")
        else:
            confirmed.append(c)
            print(f"  [auto] {c.app_name} -> {c.exe_path}")
    
    print(f"\nAuto-selected {len(confirmed)} games.")
    return confirmed

main_mod.select_games_to_add = auto_select_games

def main():
    logger = setup_logging(LOG_FILE_NAME)
    steam_path = find_steam_install_path()
    if steam_path is None:
        print("Steam path not found")
        return 1
    
    users = list_steam_users(steam_path)
    if not users:
        print("No Steam users found")
        return 1
    
    print(f"Steam path: {steam_path}")
    print(f"Users: {[u.user_id for u in users]}")
    
    print("\n=== Scanning and adding games ===")
    added = scan_and_add_games(steam_path, users, logger)
    print(f"\nAdded {added} games total.")
    
    print("\n=== Downloading artwork ===")
    downloaded = download_all_artwork(users, logger)
    print(f"Downloaded {downloaded} artwork files.")
    
    print("\n=== Done! Start Steam to see your games. ===")
    return 0

if __name__ == "__main__":
    sys.exit(main())
