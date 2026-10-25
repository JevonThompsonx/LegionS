# Unlaunched Games

These 33 games have NOT been launched on the current Steam install (no compatdata prefix, no LastPlayTime).
They cannot be played until the game exe paths are accessible (see ARCHITECTURE.md for the symlink problem).

## Unlaunched List

| # | App Name | Directory on Disk | Notes |
|---|----------|-------------------|-------|
| 1 | Dispatch | Dispatch/ | |
| 2 | FINAL Fantasy X-X-2 HD Remaster | FINAL FANTASY X-X-2 HD Remaster/ | |
| 3 | Slime Rancher | Slime Rancher v1.4.4/ | |
| 4 | Kingdom HEARTS HD 2.8 Final Chapter Prologue | KINGDOM HEARTS HD 2.8 Final Chapter Prologue/ | |
| 5 | Necrodancer | Necrodancer/ | ⚠️ Duplicate of Crypt of the NecroDancer |
| 6 | Kingdom HEARTS HD 1.5+2.5 | KINGDOM HEARTS -HD 1.52.5 ReMIX-/ | |
| 7 | Crypt of the NecroDancer | Crypt of the NecroDancer/ | |
| 8 | Deltarune | Deltarune/ | |
| 9 | Portal | Portal/ | |
| 10 | Portal 2 | Portal 2/ | |
| 11 | Proton Plus | bin/ | ⚠️ NOT a game — flatpak, delete |
| 12 | Eiyuden Chronicle Rising | Eiyuden Chronicle Rising/ | |
| 13 | MARVEL Cosmic Invasion | MARVEL Cosmic Invasion/ | |
| 14 | Nine Sols | Nine Sols/ | |
| 15 | Pragmata | PRAGMATA/ | |
| 16 | Schedule I | Schedule I/ | |
| 17 | Unbeatable | UNBEATABLE/ | |
| 18 | Card-en-ciel | Card-en-Ciel/ | No grid artwork |
| 19 | Esoteric Ebb | Esoteric Ebb/ | |
| 20 | Etrian Odyssey | Etrian Odyssey HD/ | |
| 21 | Etrian Odyssey 2 | Etrian Odyssey II HD/ | |
| 22 | Etrian Odyssey 3 | Etrian Odyssey III HD/ | |
| 23 | Final Fantasy III | Final Fantasy III/ | |
| 24 | Furi | Furi/ | |
| 25 | Octopath Traveler 0 | OCTOPATH TRAVELER 0/ | |
| 26 | Of Ash and Steel | Of Ash and Steel/ | |
| 27 | Ogu and the Secret Forest | Ogu and the Secret Forest/ | |
| 28 | Trails Beyond the Horizon | TLoH - Trails beyond the Horizon/ | No grid artwork |
| 29 | Tactics Ogre Reborn | Tactics Ogre Reborn/ | |
| 30 | Tales of Xillia Remastered | Tales of Xillia Remastered/ | |
| 31 | The Great Ace Attorney Chronicles | TGAAC/ | No grid artwork |
| 32 | The Binding of Isaac Repentance | The Binding of Isaac Repentance/ | |
| 33 | The Last of Us | The Last of Us/ | |

## Prerequisites for Unlaunched Games
Before any of these games can be launched:
1. Game exe paths need to resolve. Fix the `/home/jevonx/Games` symlink or mount the game source.
2. Each game must be launched once to create its Proton prefix (compatdata/<appid>/)
3. run link-game-saves.sh to symlink saves into the new prefix

## Games Available on ROG Drive (Micron 2400)
These games are physically present on this machine (at `/run/media/system/bazzite/home/jevonx/Games/`):
Absolum, Beastieball, Celeste, Constance, Dragon Quest I-II, Eden, Esoteric Ebb, Flowstone Saga, Furi, Haste.v1.7.c, LS, Mega Man BN Legacy Collection, Neon.Abyss.2, NSwitch, Ogu and the Secret Forest, Possessor(s), Ra Ra BOOM, RiftOfTheNecroDancer, Risk of Rain 2, Rune Factory 3 Special, Shape of Dreams, Spiritfall, Suikoden I&II HD Remaster, Tactics Ogre Reborn

These are NOT the same games as the ones in this unlaunched list (those are on the rog machine's Games_Data partition, machine offline).
