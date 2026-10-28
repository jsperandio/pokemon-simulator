# POKéMON Simulator

<p align="center">
  <img src="docs/screenshot.png" width="800" alt="A battle in the classic interface: Ash's Pikachu against a wild Mewtwo, with Charizard, Bulbasaur, Squirtle, Pidgeotto and Kingler in the party">
</p>

A cross-platform port of **POKéMON Simulator** to [Go](https://go.dev) and [Ebitengine](https://ebitengine.org). The original is a Windows freeware game made by Michael Madsen (BirdieSoft) in 2002–2004. The port starts from its V4.5, which you can still download from [its page on BirdieSoft's site](https://birdiesoft.dk/software.php?id=2).

The original only runs on Windows, or on other systems through Wine. This port makes it playable again on Linux, Windows, macOS and in the browser, with the same rules, data and look, and adds a few [improvements](#improvements) of its own.

> **The original, as it was:** [v4.5.1](https://github.com/jsperandio/pokemon-simulator/releases/tag/v4.5.1) is the faithful port, with nothing added. Take its archives, or [play it in the browser](https://jsperandio.github.io/pokemon-simulator-web/v4.5.1/), to have the game exactly as Michael Madsen made it.

> **Status:** the port is complete. The game's rules are done: the tests replay recordings of the original game and get the same results, draw for draw. The classic interface plays the whole game on the desktop and in the browser, with the manual and the music. The versions from 4.6 on add improvements on top.

## Play in the browser

Each link opens that version's game, with nothing to download:

| Version | | What it is | Notes |
|---|---|---|---|
| [v4.6.0](https://jsperandio.github.io/pokemon-simulator-web/v4.6.0/) | Improved | the original with the [improvements](#improvements) | [release notes](https://github.com/jsperandio/pokemon-simulator/releases/tag/v4.6.0) |
| [v4.5.1](https://jsperandio.github.io/pokemon-simulator-web/v4.5.1/) | Original | the 2004 game, as Michael Madsen made it | [release notes](https://github.com/jsperandio/pokemon-simulator/releases/tag/v4.5.1) · [BirdieSoft's page](https://birdiesoft.dk/software.php?id=2) |

[Playing in the browser](docs/playing.md#in-the-browser) says where the game keeps your files.

## Download

Each release has an archive per system on the repository's [Releases page](https://github.com/jsperandio/pokemon-simulator/releases). Nothing needs installing: extract the archive and run the program in it. The programs aren't signed, so Windows and macOS warn about them the first time.

- **Windows** (`windows-amd64.zip`): run `pokemon-simulator.exe`. If SmartScreen says it protected your PC, choose More info, then Run anyway.
- **macOS** (`macos-arm64.zip` for Apple silicon, `macos-amd64.zip` for Intel): macOS won't open the program until you allow it. Either run `xattr -d com.apple.quarantine pokemon-simulator` in Terminal, in its folder, or try to open it once and then choose Open Anyway in System Settings → Privacy & Security. It runs in a Terminal window, beside the game's.
- **Linux** (`linux-amd64.tar.gz` or `linux-arm64.tar.gz`): run `./pokemon-simulator`. It needs X11 or XWayland and OpenGL, as desktop distributions have. To add it to the menu, see [Linux menu entry](#linux-menu-entry).
- **In the browser** (`web.zip`): put its files on a web server. Opening `index.html` from the disk doesn't work, as browsers don't load the game from there.

### Linux menu entry

The Linux archive has two optional scripts for the desktop's menu and taskbar, which you run in its folder:

- `./install-desktop.sh` adds the game to the menu and gives its windows the game's icon in the taskbar. It installs only for your user and points the entry at that folder, so run it again if you move the folder.
- `./uninstall-desktop.sh` removes the entry. It leaves the folder alone, so delete that yourself to remove the game.

## Playing

The game plays as the original did, apart from the [improvements](#improvements). Its manual is under Help → Contents (F1).

- **Saving:** choose a file for a new game; the game saves when you quit.
- **Music:** Music → Open file... plays MP3, Ogg Vorbis and WAV.
- **In the browser:** files stay in the browser; Upload and Download move them.
- **On a phone or a tablet:** a tap is a click; hold the device sideways.
- **On Linux with GNOME:** if GNOME's title bar helper stops, the game starts without one.
- **If the game can't start,** it says why in a message box.

More details: [docs/playing.md](docs/playing.md).

## Improvements

The original doesn't have these:

- Animated sprites
- Shiny POKéMON
- Don't show this message again
- Fill the screen
- Search Again
- Coloured Battle Info
- Small interface changes

More details: [docs/improvements.md](docs/improvements.md).

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

Every `make` build takes its version from the latest `v*` tag. `make dist` names the archives after it, and it goes into the Windows program's properties and the game's title: a build on a release's tag shows its version, and any other build, like one from `go run .`, shows "(dev)". `make dist` needs GNU tar and zip, and network access the first time, to fetch [go-winres](https://github.com/tc-hib/go-winres), which gives the Windows program its icon.

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
- **The Bulbagarden Archives' contributors:** the recordings of POKéMON Emerald's appearance animations.
- **Hajime Hoshi and contributors:** Ebitengine.
- **The Wine project:** the MS Sans Serif replacement font and the dialog icons.
- **Red Hat and Google:** the Liberation Sans font.
