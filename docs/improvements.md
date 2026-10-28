# Improvements

The original POKéMON Simulator V4.5 doesn't have these. The port's v4.5 releases are the original as it was; the versions from 4.6 on add them.

- [Animated sprites](#animated-sprites)
- [Shiny POKéMON](#shiny-pokémon)
- [Don't show this message again](#dont-show-this-message-again)
- [Fill the screen](#fill-the-screen)
- [Search Again](#search-again)
- [Coloured Battle Info](#coloured-battle-info)
- [Small interface changes](#small-interface-changes)
- [Where the choices are kept](#where-the-choices-are-kept)

## Animated sprites

A POKéMON coming into the Battle Arena plays its animation from POKéMON Emerald: the wild POKéMON when it appears, and yours when you choose it. The POKéDEX plays it too, for the POKéMON you choose there. Then the original's still sprite shows.

To turn them off, clear Enable animated sprites in File → Preferences..., under Graphics.

## Shiny POKéMON

A POKéMON found with Actions → Fight a wild POKéMON → Search for a POKéMON, or the Mewtwo fought from the same menu, is shiny 1 time in 4,096, as in the games since POKéMON X and Y. So is each starter of a new game, rolled on its own: all three can be shiny. The other POKéMON, such as trainers', never are.

- **In the starter choice,** a shiny starter has its shiny colours, with the Statistics window's mark (below) at the top left of its sprite. Once chosen, Battle Info says "It's a shiny POKéMON!".
- **In battle,** a shiny POKéMON has POKéMON Emerald's shiny colours, and Battle Info says "It's a shiny POKéMON!". Stars circle it when it comes into the Battle Arena, wild or yours, as in Emerald, then stay around it.
- **In your party,** a shiny POKéMON you catch stays shiny, and so do its evolutions: a shiny Nincada gives a shiny Shedinja. The Statistics window, its details and the exported page write "(shiny)" after its name.
- **Its icon** in the Statistics window stays the original's, with a mark on it, as on a shiny starter. You choose the mark, or none, in File → Preferences..., under Graphics.
- **Mewtwo:** you can keep a shiny Mewtwo besides a normal one, but not a second of either.
- **In the POKéDEX,** the Shiny box under the sprite shows any species shiny. There the stars play with the animation only, and the still sprite has none.
- **Saves** keep shininess in lines after the original's, which the original doesn't read. They still load in the original and in the port's v4.5 releases, where every POKéMON is normal.

## Don't show this message again

The message every battle starts with, "Select a POKéMON from the Statistics Window", has a check box that hides it from then on. To bring it back, check Select a POKéMON reminder in File → Preferences..., under Messages, or click Restore Defaults there.

## Fill the screen

Fill the screen, in File → Preferences..., under Layout, docks the game's windows edge to edge instead of side by side:

- **Statistics** goes down the right edge, at the full height, so more of your party shows at once.
- **Battle Info** goes along the bottom, against Statistics, with room for longer lines.
- **The game window** takes the rest, with the Battle Arena in its middle.

The three windows stay in place. The other windows, such as the POKéDEX, open over them as usual. Separate windows (classic), the default, puts them back side by side. The game's screen is 800×600, so in a wider window the bars at its sides stay.

## Search Again

When a battle against a POKéMON you searched for is over, Search Again shows where Go! was, and End Fight shows as usual. The search can be Actions → Fight a wild POKéMON → Search for a POKéMON, or the Go! or Random of the Mewtwo-power picker. Search Again is End Fight and the same search again, however the battle ended: a win, Run!, a loss or a catch.

- **Your POKéMON comes out again,** at full HP, with no "Select a POKéMON" message, details or "Choose X?". Its attack stays selected, unless it was Run!.
- **With Mewtwo in your party,** the same species comes again at the same level, or a new random one if you clicked Random, with no psychic powers message. Each counts as an ordinary wild battle. As in the original, the picker's own Go! counts its battle as the same kind as the battle before it, so repeating that could give a Badge or a champion's bonus at every win.
- **To change POKéMON,** double-click another one in the Statistics window before you attack, run or throw a ball: it comes out at full HP, with no attack selected. The battle isn't counted again, and the next Search Again sends the new one.
- **A POKéMON you catch** goes to the end of your party, so the next Search Again sends out the same POKéMON as before. A new species opens the POKéDEX, as usual.
- **F3** is Search for a POKéMON, as the menu shows, and Search Again once the battle is over, while the game window is active.

Search Again doesn't show after Actions → Fight a wild POKéMON → Mewtwo, or after a trainer's battle.

## Coloured Battle Info

Battle Info colours the lines that tell how the battle is going:

- **Green:** the damage your POKéMON deals and its critical hits, the enemy fainting, and a catch.
- **Red:** the damage your POKéMON takes and the enemy's critical hits, your POKéMON fainting, and the money, experience or level you lose.
- **Blue:** the experience your POKéMON gains, and a level up.
- **Gold:** the money you win, and "It's a shiny POKéMON!".
- **Teal:** the HP a Potion gives back.
- **Purple:** a POKéMON added to the POKéDEX.

A level up, a catch, either POKéMON fainting, a shiny and a critical hit are in bold too. The other lines, such as the attacks used and how effective they are, stay black, as in the original.

In File → Preferences..., under Battle Info, clear Colour the messages to turn the colours off, and choose White under Background to make Battle Info white instead of the original's grey. The colours are a little darker on the grey, where pure red and green are hard to read. The page's preview shows your choice before you keep it.

## Small interface changes

- **A new game** asks for your name first. Save As then suggests a file named after you: `Gary.pgs`, or `Gary2.pgs` if the folder already has a `Gary.pgs`. The original asked for the file first, with an empty name box.
- **The starter choice** opens in the middle of the screen. Once you choose, the game window moves beside the Statistics and Battle Info windows, unless you moved it yourself.
- **The HP bars** in the Battle Arena are the same width for both POKéMON, and longer: the game window and Battle Info are 42 pixels wider to make room. The original started each bar after its POKéMON's name, so the enemy's, with "Enemy" before the name, was usually the shorter.
- **Go!** is 22 pixels high instead of 15, with room above and below it, and lines up with the HP bars. Search Again, in its place, is as high. To make room, the Battle Arena's frame reaches further into the game window's margins, so the window keeps its size.
- **The focus** goes to Go! whenever your POKéMON comes out, and the game window becomes active. Go! is also the default button, so Enter attacks, even after you click an attack. In the original, the Statistics window stayed active, and Enter pressed Go! only after a click on it.

## Where the choices are kept

The game keeps the choices of File → Preferences... in `settings.ini`, in the system's folder for programs' settings (`~/.config/pokemon-simulator/` on Linux, `%AppData%\pokemon-simulator\` on Windows, `~/Library/Application Support/pokemon-simulator/` on macOS), or in the browser's storage.
