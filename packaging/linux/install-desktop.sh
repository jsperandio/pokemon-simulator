#!/bin/sh
# install-desktop.sh adds POKéMON Simulator to the desktop's menu, and gives
# its windows the game's icon in the taskbar. Run it once from the folder
# the game is in; it points the entry at that folder.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
data="${XDG_DATA_HOME:-$HOME/.local/share}"
icons="$data/icons/hicolor/32x32/apps"

mkdir -p "$data/applications" "$icons"
cp "$here/pokemon-simulator.png" "$icons/pokemon-simulator.png"
# The Exec key holds the path in double quotes, which the Desktop Entry
# Specification says to escape: a backslash is written as four, a double
# quote, a backtick and a dollar sign get two before them, and a percent
# sign is doubled. awk takes the result from the environment, so it changes
# nothing in it, as sed would with an & or a | in the folder's name.
exe=$(printf '%s' "$here/pokemon-simulator" |
	sed -e 's/\\/\\\\\\\\/g' -e 's/["`$]/\\\\&/g' -e 's/%/%%/g')
EXE=$exe awk '/^Exec=/ { print "Exec=\"" ENVIRON["EXE"] "\""; next } { print }' \
	"$here/pokemon-simulator.desktop" > "$data/applications/pokemon-simulator.desktop"

if command -v update-desktop-database >/dev/null; then
	update-desktop-database "$data/applications" || true
fi
if command -v gtk-update-icon-cache >/dev/null; then
	gtk-update-icon-cache -q -t "$data/icons/hicolor" 2>/dev/null || true
fi

echo "Installed POKéMON Simulator in $data/applications."
