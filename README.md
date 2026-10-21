# POKéMON Simulator

<p align="center">
  <img src="docs/screenshot.png" width="800" alt="A battle in the classic interface: Ash's Pikachu against a wild Mewtwo, with Charizard, Bulbasaur, Squirtle, Pidgeotto and Kingler in the party">
</p>

A faithful, cross-platform port of **POKéMON Simulator V4.5** to [Go](https://go.dev) and [Ebitengine](https://ebitengine.org). The original is a Windows freeware game made by Michael Madsen (BirdieSoft) in 2002–2004.

The original only runs on Windows, or on other systems through Wine. This port makes it playable again on Linux, Windows, macOS and in the browser, with the same rules, data and look.

> **Status:** complete. The game's rules are done: the tests replay recordings of the original game and get the same results, draw for draw. The classic interface plays the whole game on the desktop and in the browser, with the manual and the music.

## Download

Each release has an archive per system on the repository's Releases page. Nothing needs installing: extract the archive and run the program in it. The programs aren't signed, so Windows and macOS warn about them the first time.

- **Windows** (`windows-amd64.zip`): run `pokemon-simulator.exe`. If SmartScreen says it protected your PC, choose More info, then Run anyway.
- **macOS** (`macos-arm64.zip` for Apple silicon, `macos-amd64.zip` for Intel): macOS won't open the program until you allow it. Either run `xattr -d com.apple.quarantine pokemon-simulator` in Terminal, in its folder, or try to open it once and then choose Open Anyway in System Settings → Privacy & Security. It runs in a Terminal window, beside the game's.
- **Linux** (`linux-amd64.tar.gz` or `linux-arm64.tar.gz`): run `./pokemon-simulator`. It needs X11 or XWayland and OpenGL, as desktop distributions have. To add it to the menu, see [Linux menu entry](#linux-menu-entry).
- **In the browser** (`web.zip`): put its files on a web server, or upload the archive to itch.io as an HTML game. Opening `index.html` from the disk doesn't work, as browsers don't load the game from there.

### Linux menu entry

The Linux archive has two optional scripts for the desktop's menu and taskbar, which you run in its folder:

- `./install-desktop.sh` adds the game to the menu and gives its windows the game's icon in the taskbar. It installs only for your user and points the entry at that folder, so run it again if you move the folder.
- `./uninstall-desktop.sh` removes the entry. It leaves the folder alone, so delete that yourself to remove the game.

## Playing

The game plays as the original did, and its manual is under Help → Contents (F1). A few things differ by platform:

- **Saving.** As in the original, you choose a save file when you start a new game, and the game saves when you quit (File → Quit, or closing the window).
- **Music.** Music → Open file... plays MP3, Ogg Vorbis and WAV files. The original played what Windows Media Player could, but WMA, MIDI and video files aren't supported. On Linux, music needs a PulseAudio or PipeWire sound server, as desktop distributions have.
- **On Linux with GNOME 44 or later,** the game's window gets its title bar from a GNOME helper, `mutter-x11-frames`, as every X11 program's does. If that helper has stopped, GNOME shows none of these windows. The game notices after a few seconds and starts again without a title bar, and a notification says so: move the window with the Super key and the mouse, and close it with Alt+F4 or File → Quit. Logging out and in again brings the title bars back.
- **If the game can't start,** it says why in a notification or a message box, as there may be no terminal to read it in.
- **In the browser,** the game keeps its files in the browser's storage, where they stay between visits. The file dialogs have two extra buttons, next to Up One Level: **Upload** copies files from your computer, such as saves from the original or music, and **Download** gives you a copy of the file you choose. Files you drop on the game are uploaded too. Closing or reloading the page doesn't save, so the browser asks first while a game is open; use File → Quit. Exported pages open in a new tab. The game needs WebGL 2, which current browsers have.

## Building

You need Go 1.26 or later. No C compiler is needed, because every build uses `CGO_ENABLED=0`.

| Command | What it does |
|---|---|
| `make run` | Run the game from source |
| `make test` | Run the tests |
| `make lint` | Check the code of every platform with [golangci-lint](https://golangci-lint.run) 2.14 or later |
| `make dist` | Build the Linux, Windows and macOS programs and the WebAssembly version, and pack each into an archive in `dist/`, with the licences |
| `make web` | Build only the WebAssembly version into `dist/web/` |
| `make serve` | Build and serve the WebAssembly version at http://localhost:8080 (needs Python 3) |

`make dist` names the archives after the latest `v*` tag, such as `v1.0.0`, which also goes into the Windows program's properties. It needs GNU tar and zip, and network access the first time, to fetch [go-winres](https://github.com/tc-hib/go-winres), which gives the Windows program its icon.

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
