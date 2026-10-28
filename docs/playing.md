# Playing

The game plays as the original did, apart from the [improvements](improvements.md), and its manual is under Help → Contents (F1). These are the things that differ by platform.

- [Saving](#saving)
- [Music](#music)
- [In the browser](#in-the-browser)
- [On a phone or a tablet](#on-a-phone-or-a-tablet)
- [On Linux with GNOME](#on-linux-with-gnome)
- [If the game can't start](#if-the-game-cant-start)

## Saving

As in the original, you choose a save file when you start a new game, and the game saves when you quit (File → Quit, or closing the window).

## Music

Music → Open file... plays MP3, Ogg Vorbis and WAV files. The original played what Windows Media Player could, but WMA, MIDI and video files aren't supported. On Linux, music needs a PulseAudio or PipeWire sound server, as desktop distributions have.

## In the browser

The game keeps its files in the browser's storage, where they stay between visits. It needs WebGL 2, which current browsers have.

- **Upload and Download.** The file dialogs have two extra buttons, next to Up One Level. **Upload** copies files from your computer, such as saves from the original or music, and **Download** gives you a copy of the file you choose. Files you drop on the game are uploaded too.
- **Saves between versions.** The versions from 4.6 on keep their files apart from v4.5's. To carry a save over, Download it in one and Upload it in the other.
- **Quitting.** Closing or reloading the page doesn't save, so the browser asks first while a game is open. Use File → Quit.
- **Exported pages** open in a new tab.

## On a phone or a tablet

A tap is a click, and two quick taps on one spot are a double click. Tapping a text box, such as the file name in Save As, brings up the keyboard, whose Enter key works as Enter does. The word being typed shows underlined until the keyboard types it in.

The game's 800×600 screen is easier to play with the device held sideways.

## On Linux with GNOME

On GNOME 44 or later, the game's window gets its title bar from a GNOME helper, `mutter-x11-frames`, as every X11 program's does. If that helper has stopped, GNOME shows none of these windows. The game notices after a few seconds and starts again without a title bar, and a notification says so. Move the window with the Super key and the mouse, and close it with Alt+F4 or File → Quit. Logging out and in again brings the title bars back.

## If the game can't start

It says why in a notification or a message box, as there may be no terminal to read it in.
