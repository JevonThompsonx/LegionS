#!/usr/bin/env python3
"""Non-interactive wrapper — force-refresh official-first artwork for all shortcuts."""
import sys
sys.path.insert(0, "/run/media/system/Games_Data/Add-Non-Steam-Games-Linux")

from config import LOG_FILE_NAME
from logger_setup import setup_logging
from steam_paths import find_steam_install_path, list_steam_users
from main import download_all_artwork


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

    print("\n=== Force-refreshing official-first artwork ===")
    downloaded = download_all_artwork(users, logger, refresh=True)
    print(f"\nTotal artwork files downloaded in refresh: {downloaded}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
