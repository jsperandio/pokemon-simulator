#!/bin/sh
# uninstall-desktop.sh removes what install-desktop.sh added to the desktop's
# menu and icons. It leaves the game's folder alone.
set -eu

data="${XDG_DATA_HOME:-$HOME/.local/share}"

rm -f "$data/applications/pokemon-simulator.desktop" \
	"$data/icons/hicolor/32x32/apps/pokemon-simulator.png"

if command -v update-desktop-database >/dev/null && [ -d "$data/applications" ]; then
	update-desktop-database "$data/applications" || true
fi
if command -v gtk-update-icon-cache >/dev/null && [ -d "$data/icons/hicolor" ]; then
	gtk-update-icon-cache -q -t "$data/icons/hicolor" 2>/dev/null || true
fi

echo "Removed POKéMON Simulator from $data/applications."
