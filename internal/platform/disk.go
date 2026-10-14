// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package platform

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// disk is the file system.
type disk struct {
	home string
}

// Default returns the platform's storage: here, the disk.
func Default() Files {
	return disk{}
}

// Disk returns the disk with the dialogs starting in home.
func Disk(home string) Files {
	return disk{
		home,
	}
}

// Home is the current folder, as the original's dialogs opened there, or
// the user's home folder when the program runs from the file system's
// root (a desktop launcher, an application bundle).
func (d disk) Home() string {
	if d.home != "" {
		return d.home
	}

	if wd, err := os.Getwd(); err == nil && filepath.Dir(wd) != wd {
		return wd
	}

	if h, err := os.UserHomeDir(); err == nil {
		return h
	}

	return string(filepath.Separator)
}

func (disk) Roots() []string {
	if runtime.GOOS != "windows" {
		return []string{"/"}
	}

	var out []string

	for c := 'A'; c <= 'Z'; c++ {
		d := string(c) + `:\`
		if _, err := os.Stat(d); err == nil {
			out = append(out, d)
		}
	}

	return out
}

func (disk) List(dir string) ([]Entry, error) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(des))
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}

		isDir := info.IsDir()
		if info.Mode()&fs.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(dir, de.Name())); err == nil {
				isDir = st.IsDir()
			}
		}

		out = append(out, Entry{
			Name:    de.Name(),
			Dir:     isDir,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	return out, nil
}

func (disk) Stat(path string) (Entry, error) {
	st, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Entry{}, ErrNotExist
	}

	if err != nil {
		return Entry{}, err
	}

	return Entry{
		Name:    st.Name(),
		Dir:     st.IsDir(),
		Size:    st.Size(),
		ModTime: st.ModTime(),
	}, nil
}

func (disk) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // the file the player chose
}

func (disk) WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o600)
}

func (disk) Mkdir(path string) error {
	return os.Mkdir(path, 0o750)
}

func (disk) Join(dir, name string) string {
	return filepath.Join(dir, name)
}

func (disk) Dir(path string) string {
	return filepath.Dir(path)
}

func (disk) Base(path string) string {
	return filepath.Base(path)
}

func (disk) IsAbs(path string) bool {
	return filepath.IsAbs(path)
}

// OpenPage shows a page, written to the file at path, in the system's
// browser.
func OpenPage(path string, _ []byte) error {
	return start(path)
}

// OpenURL opens a web address in the system's browser, or a mailto: one in
// its e-mail program.
func OpenURL(url string) error {
	return start(url)
}

// start opens a file or an address with the program the system has for it.
func start(target string) error {
	var cmd *exec.Cmd

	//nolint:gosec // the programs are fixed, and target is the game's page or address
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		cmd = exec.Command("open", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go cmd.Wait() //nolint:errcheck // the browser outlives the command

	return nil
}

// Args returns the command line's arguments.
func Args() []string {
	return os.Args[1:]
}

// Transfers is false: the game's files are the player's own, on the disk.
const Transfers bool = false

// Upload does nothing on the disk; see Transfers.
func Upload(Files, string) <-chan []string {
	return nil
}

// Download does nothing on the disk; see Transfers.
func Download(string, []byte) error {
	return nil
}

// WarnOnLeave does nothing on the desktop, where closing the window saves.
func WarnOnLeave(bool) {}
