// SPDX-License-Identifier: AGPL-3.0-only

package audio

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

// fakePlayer stands in for the sound card's player.
type fakePlayer struct {
	src     io.Reader
	playing bool
	plays   int
	pos     int64
	closed  bool
}

func (f *fakePlayer) Play()           { f.playing, f.plays = true, f.plays+1 }
func (f *fakePlayer) Pause()          { f.playing = false }
func (f *fakePlayer) IsPlaying() bool { return f.playing }
func (f *fakePlayer) Close() error {
	f.closed = true

	return nil
}

func (f *fakePlayer) Seek(offset int64, whence int) (int64, error) {
	f.pos = offset

	return offset, nil
}

// wavFile returns a WAV file of n silent 16-bit mono samples at 22050 Hz.
func wavFile(n int) []byte {
	b := make([]byte, 44+2*n)
	copy(b, "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+2*n))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // mono
	binary.LittleEndian.PutUint32(b[24:], 22050)
	binary.LittleEndian.PutUint32(b[28:], 2*22050)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], uint32(2*n))

	return b
}

func TestMusic(t *testing.T) {
	var players []*fakePlayer
	m := &Music{newPlayer: func(src io.Reader) (player, error) {
		p := &fakePlayer{src: src}
		players = append(players, p)

		return p, nil
	}}

	if err := m.Stop(); !errors.Is(err, ErrNotOpen) {
		t.Errorf("Stop with nothing open: %v", err)
	}

	if err := m.Open("song.wav", wavFile(100)); err != nil {
		t.Fatal(err)
	}

	p := players[0]
	if !p.playing || p.plays != 1 || !m.Playing() {
		t.Fatal("not playing")
	}

	// The resampled stream holds 16-bit stereo at the output's rate.
	if n, _ := io.Copy(io.Discard, p.src); n != 4*200 {
		t.Errorf("decoded %d bytes", n)
	}

	m.Update(true) // still playing

	p.playing = false
	p.pos = 400

	m.Update(false) // at its end, repeat off

	if p.playing || m.Playing() {
		t.Error("played again with repeat off")
	}

	m.Update(true) // repeat turned on after the end

	if !p.playing || p.plays != 2 || p.pos != 0 {
		t.Errorf("repeat: playing %v, %d plays, at %d", p.playing, p.plays, p.pos)
	}

	p.pos = 100
	if err := m.Stop(); err != nil || p.playing || p.pos != 0 {
		t.Errorf("Stop: %v, playing %v, at %d", err, p.playing, p.pos)
	}

	m.Update(true)

	if p.playing || m.Playing() {
		t.Error("stopped music played again")
	}

	// A file that can't play closes the one before.
	if err := m.Open("song.wma", []byte("0&\xb2\x75\x8e\x66\xcf\x11")); !errors.Is(err, ErrFormat) {
		t.Errorf("Open WMA: %v", err)
	}

	if !p.closed {
		t.Error("the music before wasn't closed")
	}

	if err := m.Stop(); !errors.Is(err, ErrNotOpen) {
		t.Errorf("Stop after a failed Open: %v", err)
	}

	if err := m.Open("broken.mp3", []byte("not an mp3")); !errors.Is(err, ErrFormat) {
		t.Errorf("Open a broken MP3: %v", err)
	}
}

func TestFormat(t *testing.T) {
	for _, c := range []struct {
		name string
		data string
		want string
	}{
		{"a.bin", "RIFF\x24\x00\x00\x00WAVEfmt ", "wav"},
		{"a.bin", "OggS\x00\x02", "ogg"},
		{"a.bin", "ID3\x03\x00", "mp3"},
		{"a.bin", "\xff\xfb\x90\x64", "mp3"},
		{"A.MP3", "", "mp3"},
		{"a.ogg", "", "ogg"},
		{"a.Wav", "", "wav"},
		{"a.mid", "MThd", ""},
		{"a.wma", "0&\xb2\x75", ""},
	} {
		if got := format(c.name, []byte(c.data)); got != c.want {
			t.Errorf("format(%q, %q) = %q, want %q", c.name, c.data, got, c.want)
		}
	}
}
