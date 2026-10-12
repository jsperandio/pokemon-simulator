# POKéMON Simulator

A faithful, cross-platform port of **POKéMON Simulator V4.5** to [Go](https://go.dev) and [Ebitengine](https://ebitengine.org). The original is a Windows freeware game made by Michael Madsen (BirdieSoft) in 2002–2004.

The original only runs on Windows, or on other systems through Wine. This port makes it playable again on Linux, Windows, macOS and in the browser, with the same rules, data and look.

> **Status:** in development. The game's rules are done: the tests replay recordings of the original game and get the same results, draw for draw. The classic interface is playable on the desktop and in the browser, with the manual and the music; a few details of the original's windows are still to come.

## Playing

The game plays as the original did, and its manual is under Help → Contents (F1). A few things differ by platform:

- **Saving.** As in the original, you choose a save file when you start a new game, and the game saves when you quit (File → Quit, or closing the window).
- **Music.** Music → Open file... plays MP3, Ogg Vorbis and WAV files. The original played what Windows Media Player could, but WMA, MIDI and video files aren't supported. On Linux, music needs a PulseAudio or PipeWire sound server, as desktop distributions have.
- **In the browser,** the game keeps its files in the browser's storage, where they stay between visits. The file dialogs have two extra buttons, next to Up One Level: **Upload** copies files from your computer, such as saves from the original or music, and **Download** gives you a copy of the file you choose. Files you drop on the game are uploaded too. Closing or reloading the page doesn't save, so the browser asks first while a game is open; use File → Quit. Exported pages open in a new tab.

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
