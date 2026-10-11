# Assets

Everything in this directory, except the Go files, comes from the original **POKéMON Simulator V4.5** (Michael Madsen, BirdieSoft, 2004). The port embeds these files in its binaries, so it plays and looks exactly like the original.

## Rights

The [AGPL](../LICENSE) covers the port's source code only. **It does not cover these assets.** It also does not cover the form layouts translated from the original, in `internal/ui/classic/layout/` (every file there except `layout.go` and the tests). They remain the property of the owners listed below, and nobody in this project can license them to you.

As the copyright holders of the code, we allow the port's binaries to be distributed with these assets embedded, as an additional permission under section 7 of the AGPL. That permission concerns our code only. It grants no rights in the assets.

This is a non-commercial project. If you hold rights to any of this material and want it removed, please open an issue.

## Inventory

| Path | Content | Taken from | Rights holders |
|---|---|---|---|
| `pokegame.dat` | Type chart, species, moves, badges, tournament prizes and trainer names | `POKéGAME.dat`, byte for byte | Michael Madsen (game data). Names of POKéMON, moves, types and badges: Nintendo / Creatures / GAME FREAK |
| `sprites/front64.png` | 386 front sprites, 64×64, 20 per row. Frame *n* is species *n*+1 in "Old POKéDEX" order | Image list `PKMNImages` in `PokeGame.exe` | Nintendo / Creatures / GAME FREAK. These are the official POKéMON Ruby & Sapphire front sprites |
| `sprites/icons32.png` | The same 386 POKéMON at 32×32, in the same layout | Image list `SmallPKMNImg` | Nintendo / Creatures / GAME FREAK |
| `sprites/small16.png` | The 16×16 icons of the File → Quit and Help → Contents menu items | Image list `ImageList1` | Unverified. They are Windows-style icons shipped with the original |
| `sprites/pokeball.png` | The 15×15 POKéBALL shown under a wild POKéMON that is already in the POKéDEX | `Form4.Image1` | Shipped with the original. POKéBALL design: Nintendo / Creatures / GAME FREAK |
| `sprites/icon.png` | Program icon, a Pidgeot, 32×32 | `MAINICON` | Shipped with the original. Pidgeot: Nintendo / Creatures / GAME FREAK |
| `help/*.md` | The game's manual, one page per topic | `POKéGAME.HLP` | Michael Madsen |
| `help/*.png` | The manual's screenshots and figures | `POKéGAME.HLP` | Michael Madsen. The screenshots show sprites owned by Nintendo / Creatures / GAME FREAK |

The form layouts in `internal/ui/classic/layout/` hold the original's window positions, sizes, captions, menus and event handler names. They are Michael Madsen's design.

## How they were extracted

The files were extracted from the original V4.5 release with one-off tools that are not part of this repository:

- **Sprites.** The image lists' 32-bit bitmaps were decoded, and their 1-bit masks became the alpha channel.
- **Pictures.** The background of `pokeball.png` (the colour of its bottom-left pixel) was made transparent, as a transparent Delphi TImage does. `icon.png` uses its icon mask as alpha.
- **Help.** The WinHelp file was decompiled with helpdeco and converted to Markdown, keeping its text, links and images. Each page's title is a Markdown heading. The red "POKéMON Simulator" banner above it in the original is left to the help viewer. The only additions are quoted **Port note** paragraphs, which flag outdated contact details, the dead donation link and instructions that don't apply to the port.

Original files used:

| File | SHA-256 |
|---|---|
| `PokeGame.exe` | `0db513d5a7f75bbe88c3ac80552019d7ae60826fad86f06a434b20defc62cb25` |
| `POKéGAME.dat` | `ae55e8a0717d3c9c7909af2ea7ea5dd6af44f60ea66012ba8709a63378f06484` |
| `POKéGAME.HLP` | `888a452e7640e9e1ebf16f2f0dae4dff07351a9bf8c8d0d712668f6d8ed868b5` |
