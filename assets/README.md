# Assets

Most files in this directory come from the original **POKéMON Simulator V4.5** (Michael Madsen, BirdieSoft, 2004). The port embeds them in its binaries, so it plays and looks exactly like the original.

These files do not come from the original:

- the Go files;
- the fonts and icons in `wine/` and `liberation/`;
- the manual's screenshots;
- the appearance animations in `anim/`.

The original has no animations.

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
| `help/*.png` | The manual's screenshots | Taken in the port, of the scenes that `POKéGAME.HLP`'s screenshots show | Michael Madsen's windows and texts. The sprites in them: Nintendo / Creatures / GAME FREAK |
| `anim/front/*.png`, `anim/front.json` | The animation POKéMON Emerald plays when each of the 386 POKéMON appears in battle: its distinct frames side by side, then each frame's place and how many Game Boy Advance frames it shows for. Not in the original game | Recordings of Emerald's animations from the [Bulbagarden Archives](https://archives.bulbagarden.net/wiki/Category:Emerald_sprites), repacked | Nintendo / Creatures / GAME FREAK. These are the official POKéMON Emerald front sprites. The archive's contributors recorded them and claim fair use for them on the archive; that claim grants nothing to this project |

A build with `-tags noanim` leaves `anim/` out, and the game then shows the still sprites only.

The form layouts in `internal/ui/classic/layout/` hold the original's window positions, sizes, captions, menus and event handler names. They are Michael Madsen's design.

## Fonts and icons

The original draws its windows with fonts and icons that come with Windows and can't be redistributed. The port uses free stand-ins instead. They don't come from the original, and their own licences allow embedding them:

| Path | Stands in for | Content | Licence |
|---|---|---|---|
| `wine/sserife.fon` | MS Sans Serif 8 pt, the font of every window and dialog | Wine 11.0's bitmap MS Sans Serif, byte for byte. Its characters have the same widths as Microsoft's | © 2004 Huw D M Davies, Dmitry Timoshkov. [LGPL 2.1](wine/LGPL-2.1.txt) or later. Its source is `wine/ms_sans_serif.sfd`, from Wine 11.0's source tree, which builds it as that file's `#pragma makedep font sserife.fon` line describes |
| `wine/oic_*.png` | The warning, error, information and question icons of message dialogs | The 32×32, 32-bit images of Wine 11.0's `oic_bang.ico`, `oic_hand.ico`, `oic_note.ico` and `oic_ques.ico` | The Wine project. [LGPL 2.1](wine/LGPL-2.1.txt) or later. Their sources are the `wine/oic_*.svg` files, from Wine 11.0's source tree |
| `liberation/LiberationSans-*.ttf` | Arial, used by the red titles of the start menu and the loading screen, and by the help manual | Liberation Sans 2.1.5: Regular, Bold and Italic | © 2010 Google Corporation, © 2012 Red Hat, Inc. [SIL Open Font License 1.1](liberation/OFL-1.1.txt) |

The binaries embed the `.fon`, `.png` and `.ttf` files. Release archives ship the licence texts and the `.sfd` and `.svg` sources next to them.

## Original files

The files that come from the original were taken from these files of its V4.5 release:

| File | SHA-256 |
|---|---|
| `PokeGame.exe` | `0db513d5a7f75bbe88c3ac80552019d7ae60826fad86f06a434b20defc62cb25` |
| `POKéGAME.dat` | `ae55e8a0717d3c9c7909af2ea7ea5dd6af44f60ea66012ba8709a63378f06484` |
| `POKéGAME.HLP` | `888a452e7640e9e1ebf16f2f0dae4dff07351a9bf8c8d0d712668f6d8ed868b5` |
