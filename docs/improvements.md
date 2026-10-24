# Improvements

The original POKéMON Simulator V4.5 doesn't have these. The port's v4.5.0 is the original as it was; the versions after it add them.

- [Animated sprites](#animated-sprites)
- [Don't show this message again](#dont-show-this-message-again)
- [Where the choices are kept](#where-the-choices-are-kept)

## Animated sprites

A POKéMON coming into the Battle Arena plays its animation from POKéMON Emerald: the wild POKéMON when it appears, and yours when you choose it. The POKéDEX plays it too, for the POKéMON you choose there. Then the original's still sprite shows.

To turn them off, clear Enable animated sprites in File → Preferences..., under Graphics.

## Don't show this message again

The message every battle starts with, "Select a POKéMON from the Statistics Window", has a check box that hides it from then on. To bring it back, check Select a POKéMON reminder in File → Preferences..., under Messages, or click Restore Defaults there.

## Where the choices are kept

The game keeps the choices of File → Preferences... in `settings.ini`, in the system's folder for programs' settings (`~/.config/pokemon-simulator/` on Linux, `%AppData%\pokemon-simulator\` on Windows, `~/Library/Application Support/pokemon-simulator/` on macOS), or in the browser's storage.
