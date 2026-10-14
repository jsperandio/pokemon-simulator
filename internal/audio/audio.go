// SPDX-License-Identifier: AGPL-3.0-only

// Package audio plays the music opened from the Music menu, as the
// original's TMediaPlayer did: one file at a time, from its start, which
// the menu stops, or which plays again from its end while "Repeat Music"
// is on. It decodes MP3, Ogg Vorbis and WAV in Go and plays them through
// Oto, without cgo. The original played whatever Windows Media Player
// could, WMA, MIDI and video too; those aren't played here.
package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"sync"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// sampleRate is the output's, to which the decoders convert: most MP3
// files'.
const sampleRate int = 44100

var (
	// ErrNotOpen is Stop's error when no music is open, where the
	// original's media player raised "No MCI device open".
	ErrNotOpen = errors.New("audio: no music open")
	// ErrFormat is Open's error for a file it can't play.
	ErrFormat = errors.New("audio: not an MP3, Ogg Vorbis or WAV file")
)

// Music is the music player. Its zero value plays on the sound card.
type Music struct {
	newPlayer func(io.Reader) (player, error) // nil for the sound card's
	p         player
	stopped   bool // Stop stopped it
}

// player plays decoded sound: Oto's player, or a test's.
type player interface {
	Play()
	Pause()
	IsPlaying() bool
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

// Open closes the music open before, then plays a file's data from its
// start. The data's first bytes tell its format, or else the extension of
// name.
func (m *Music) Open(name string, data []byte) error {
	m.close()

	src, err := decode(name, data)
	if err != nil {
		return err
	}

	newPlayer := m.newPlayer
	if newPlayer == nil {
		newPlayer = soundCard
	}

	p, err := newPlayer(src)
	if err != nil {
		return err
	}

	m.p, m.stopped = p, false
	p.Play()

	return nil
}

// Stop stops the music and rewinds it: Music → Stop Music.
func (m *Music) Stop() error {
	if m.p == nil {
		return ErrNotOpen
	}

	m.p.Pause()

	m.stopped = true
	_, err := m.p.Seek(0, io.SeekStart)

	return err
}

// Update plays the music again from its start once it has ended, while
// repeat is on. The original's Timer1 checks on every tick, so turning
// repeat on after the end plays it again too.
func (m *Music) Update(repeat bool) {
	if !repeat || !m.ended() {
		return
	}

	if _, err := m.p.Seek(0, io.SeekStart); err == nil {
		m.p.Play()
	}
}

// ended reports whether the music has played to its end: open, not
// stopped, and no longer playing.
func (m *Music) ended() bool {
	return m.p != nil && !m.stopped && !m.p.IsPlaying()
}

// Playing reports whether the music plays: open, not stopped, and not at
// its end.
func (m *Music) Playing() bool {
	return m.p != nil && !m.stopped && m.p.IsPlaying()
}

func (m *Music) close() {
	if m.p != nil {
		_ = m.p.Close()
		m.p = nil
	}
}

// The sound card's context: Oto allows one.
var (
	card     *oto.Context
	errCard  error
	cardOnce sync.Once
)

// soundCard returns a player on the sound card, whose context starts with
// the first music opened.
func soundCard(src io.Reader) (player, error) {
	cardOnce.Do(func() {
		card, _, errCard = oto.NewContext(&oto.NewContextOptions{
			SampleRate:      sampleRate,
			ChannelCount:    2,
			Format:          oto.FormatSignedInt16LE,
			ApplicationName: "POKéMON Simulator",
		})
	})

	if errCard != nil {
		return nil, errCard
	}

	if err := card.Err(); err != nil {
		return nil, err
	}

	return card.NewPlayer(src), nil
}

// decode returns sound data as 16-bit stereo at the output's rate.
func decode(name string, data []byte) (io.ReadSeeker, error) {
	r := bytes.NewReader(data)

	var (
		s   io.ReadSeeker
		err error
	)

	switch format(name, data) {
	case "mp3":
		s, err = mp3Stream(r)
	case "ogg":
		s, err = oggStream(r)
	case "wav":
		s, err = wavStream(r)
	default:
		return nil, ErrFormat
	}

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}

	return s, nil
}

func mp3Stream(r io.Reader) (io.ReadSeeker, error) {
	s, err := mp3.DecodeWithSampleRate(sampleRate, r)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func oggStream(r io.Reader) (io.ReadSeeker, error) {
	s, err := vorbis.DecodeWithSampleRate(sampleRate, r)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func wavStream(r io.Reader) (io.ReadSeeker, error) {
	s, err := wav.DecodeWithSampleRate(sampleRate, r)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// format tells data's format from its first bytes, or else from name's
// extension: "mp3", "ogg", "wav", or "".
func format(name string, data []byte) string {
	switch {
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return "wav"
	case bytes.HasPrefix(data, []byte("OggS")):
		return "ogg"
	case bytes.HasPrefix(data, []byte("ID3")), len(data) >= 2 && data[0] == 0xFF && data[1]&0xE0 == 0xE0:
		return "mp3"
	}

	switch strings.ToLower(path.Ext(name)) {
	case ".mp3":
		return "mp3"
	case ".ogg", ".oga":
		return "ogg"
	case ".wav":
		return "wav"
	}

	return ""
}
