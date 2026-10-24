# Improvements

The original POKéMON Simulator V4.5 doesn't have these. The port's v4.5.0 is the original as it was; the versions after it add them.

- [Animated sprites](#animated-sprites)
- [Shiny POKéMON](#shiny-pokémon)
- [Don't show this message again](#dont-show-this-message-again)
- [Where the choices are kept](#where-the-choices-are-kept)

## Animated sprites

A POKéMON coming into the Battle Arena plays its animation from POKéMON Emerald: the wild POKéMON when it appears, and yours when you choose it. The POKéDEX plays it too, for the POKéMON you choose there. Then the original's still sprite shows.

To turn them off, clear Enable animated sprites in File → Preferences..., under Graphics.

## Shiny POKéMON

A POKéMON found with Actions → Fight a wild POKéMON → Search for a POKéMON, or the Mewtwo fought from the same menu, is shiny 1 time in 4,096, as in the games since POKéMON X and Y. So is each starter of a new game, rolled on its own: all three can be shiny. The other POKéMON, such as trainers', never are.

- **In the starter choice,** a shiny starter has its shiny colours and the mark below. Once chosen, Battle Info says "It's a shiny POKéMON!".
- **In battle,** a shiny POKéMON has POKéMON Emerald's shiny colours, and Battle Info says "It's a shiny POKéMON!". Stars circle it when it comes into the Battle Arena, wild or yours, as in Emerald, then stay around it.
- **In your party,** a shiny POKéMON you catch stays shiny, and so do its evolutions: a shiny Nincada gives a shiny Shedinja. The Statistics window, its details and the exported page write "(shiny)" after its name.
- **Its icon** in the Statistics window stays the original's, with a mark on it, as on a shiny starter. You choose the mark, or none, in File → Preferences..., under Graphics.
- **Mewtwo:** you can keep a shiny Mewtwo besides a normal one, but not a second of either.
- **In the POKéDEX,** the Shiny box under the sprite shows any species shiny. There the stars play with the animation only, and the still sprite has none.
- **Saves** keep shininess in lines after the original's, which the original doesn't read. They still load in the original and in v4.5.0, where every POKéMON is normal.

## Don't show this message again

The message every battle starts with, "Select a POKéMON from the Statistics Window", has a check box that hides it from then on. To bring it back, check Select a POKéMON reminder in File → Preferences..., under Messages, or click Restore Defaults there.

## Where the choices are kept

The game keeps the choices of File → Preferences... in `settings.ini`, in the system's folder for programs' settings (`~/.config/pokemon-simulator/` on Linux, `%AppData%\pokemon-simulator\` on Windows, `~/Library/Application Support/pokemon-simulator/` on macOS), or in the browser's storage.
