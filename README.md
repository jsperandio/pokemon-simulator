# POKéMON Simulator

A faithful, cross-platform port of **POKéMON Simulator V4.5** to [Go](https://go.dev) and [Ebitengine](https://ebitengine.org). The original is a Windows freeware game made by Michael Madsen (BirdieSoft) in 2002–2004.

The original only runs on Windows, or on other systems through Wine. This port makes it playable again on Linux, Windows, macOS and in the browser, with the same rules, data and look.

> **Status:** early development. The game's rules are done: the tests replay recordings of the original game and get the same results, draw for draw. The user interface isn't done, so the game is not playable yet.

## Building

You need Go 1.25 or later. No C compiler is needed, because every build uses `CGO_ENABLED=0`.

| Command | What it does |
|---|---|
| `make run` | Run the game from source |
| `make test` | Run the tests |
| `make dist` | Build Linux, Windows and macOS binaries plus the WebAssembly build into `dist/` |
| `make web` | Build only the WebAssembly version into `dist/web/` |
| `make serve` | Build and serve the WebAssembly version at http://localhost:8080 (needs Python 3) |

On Linux, the game loads the system's graphics and audio libraries at runtime. Desktop distributions already include them.

## Legal

This is an unofficial, non-commercial fan project. It is not affiliated with, endorsed or sponsored by Nintendo, Creatures Inc., GAME FREAK inc. or The Pokémon Company. POKéMON and all related names, characters and images are trademarks and copyrights of their respective owners.

Michael Madsen (BirdieSoft) created the original POKéMON Simulator. This port reimplements it so it keeps running on modern systems.

- **Source code:** licensed under the [GNU Affero General Public License v3.0 only](LICENSE).
- **Assets:** POKéMON names, sprites and icons, and the original game's data, texts, manual and window layouts. They are **not** covered by the AGPL and remain the property of their owners. [assets/README.md](assets/README.md) lists where each one comes from and who owns it. As the copyright holders of the code, we allow the program's binaries to be distributed with these assets embedded (an additional permission under section 7 of the AGPL).
- **Fonts and icons:** free stand-ins for the Windows fonts and icons the original uses, under their own licences (LGPL 2.1 or later, and SIL OFL 1.1). See [assets/README.md](assets/README.md#fonts-and-icons).
- **No monetisation:** the project will never include ads, sales or paid features.

If you hold rights to any of this material and want it removed, please open an issue on this repository.

## Credits

- **Michael Madsen (BirdieSoft):** creator of the original POKéMON Simulator.
- **Nintendo, Creatures Inc. and GAME FREAK inc.:** POKéMON.
- **Hajime Hoshi and contributors:** Ebitengine.
- **The Wine project:** the MS Sans Serif replacement font and the dialog icons.
- **Red Hat and Google:** the Liberation Sans font.
