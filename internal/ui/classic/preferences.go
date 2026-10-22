// SPDX-License-Identifier: AGPL-3.0-only

package classic

import (
	"strconv"
	"strings"
)

// settingsStore keeps the preferences' bytes between runs: a file on the
// desktop, the page's local storage on the web.
type settingsStore interface {
	Read() ([]byte, error)
	Write(data []byte) error
}

// preferences are the player's choices in the Preferences window, which
// the original doesn't have.
type preferences struct {
	animatedSprites bool // Graphics: Enable animated sprites
}

// defaultPreferences are the choices before the player makes any.
func defaultPreferences() preferences {
	return preferences{
		animatedSprites: true,
	}
}

// loadPreferences reads the preferences from dat, an INI file; what is
// missing or unreadable keeps its default.
func loadPreferences(dat []byte) preferences {
	prf := defaultPreferences()
	ini := parseINI(dat)
	prf.animatedSprites = ini.boolean("Graphics", "AnimatedSprites", prf.animatedSprites)

	return prf
}

// iniFile is a settings file as Delphi's TIniFile reads and writes it:
// [Section] lines, each followed by its Key=Value lines. Sections and keys
// match whatever their case, and the lines it doesn't know are kept.
type iniFile struct {
	lines []string
}

func parseINI(dat []byte) *iniFile {
	txt := strings.ReplaceAll(string(dat), "\r\n", "\n")
	ini := &iniFile{}

	if txt = strings.TrimSuffix(txt, "\n"); txt != "" {
		ini.lines = strings.Split(txt, "\n")
	}

	return ini
}

// iniSection returns the name of the section that line starts, if it
// starts one.
func iniSection(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if len(line) < 2 || line[0] != '[' || line[len(line)-1] != ']' {
		return "", false
	}

	return strings.TrimSpace(line[1 : len(line)-1]), true
}

// iniKey returns the key that line sets, if it sets one.
func iniKey(line string) (string, bool) {
	key, _, ok := strings.Cut(line, "=")

	return strings.TrimSpace(key), ok
}

// value returns section's key, and whether it is there.
func (ini *iniFile) value(section, key string) (string, bool) {
	cur := ""

	for _, line := range ini.lines {
		if nam, ok := iniSection(line); ok {
			cur = nam

			continue
		}

		if !strings.EqualFold(cur, section) {
			continue
		}

		if k, ok := iniKey(line); ok && strings.EqualFold(k, key) {
			_, val, _ := strings.Cut(line, "=")

			return strings.TrimSpace(val), true
		}
	}

	return "", false
}

// boolean is ReadBool: section's key as an integer, true when not 0, or
// def when it is missing or isn't a number.
func (ini *iniFile) boolean(section, key string, def bool) bool {
	val, ok := ini.value(section, key)
	if !ok {
		return def
	}

	num, err := strconv.Atoi(val)
	if err != nil {
		return def
	}

	return num != 0
}
