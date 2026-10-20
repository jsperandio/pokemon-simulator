// SPDX-License-Identifier: AGPL-3.0-only

package audio

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wmaStart is the start of a Windows Media Audio file, a format the player
// doesn't play.
const wmaStart string = "0&\xb2\x75\x8e\x66\xcf\x11"

// fakePlayer stands in for the sound card's player.
type fakePlayer struct {
	src     io.Reader
	playing bool
	plays   int
	pos     int64
	closed  bool
}

func (fp *fakePlayer) Play() {
	fp.playing, fp.plays = true, fp.plays+1
}

func (fp *fakePlayer) Pause() {
	fp.playing = false
}

func (fp *fakePlayer) IsPlaying() bool {
	return fp.playing
}

func (fp *fakePlayer) Close() error {
	fp.closed = true

	return nil
}

func (fp *fakePlayer) Seek(offset int64, whence int) (int64, error) {
	fp.pos = offset

	return offset, nil
}

// wavFile returns a WAV file of n silent 16-bit mono samples at 22050 Hz.
func wavFile(n uint32) []byte {
	wav := make([]byte, 44+2*n)
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:], 36+2*n)
	copy(wav[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(wav[16:], 16)
	binary.LittleEndian.PutUint16(wav[20:], 1) // PCM
	binary.LittleEndian.PutUint16(wav[22:], 1) // mono
	binary.LittleEndian.PutUint32(wav[24:], 22050)
	binary.LittleEndian.PutUint32(wav[28:], 2*22050)
	binary.LittleEndian.PutUint16(wav[32:], 2)
	binary.LittleEndian.PutUint16(wav[34:], 16)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:], 2*n)

	return wav
}

// openSong opens a short WAV file on a fake sound card, and returns the
// music and the player it plays on. It requires the song to play once.
func openSong(t *testing.T) (*Music, *fakePlayer) {
	t.Helper()

	var players []*fakePlayer
	msc := &Music{
		newPlayer: func(src io.Reader) (player, error) {
			fp := &fakePlayer{
				src: src,
			}
			players = append(players, fp)

			return fp, nil
		},
	}

	require.NoError(t, msc.Open("song.wav", wavFile(100)))
	require.Len(t, players, 1)

	fp := players[0]
	require.True(t, fp.playing, "the song isn't playing")
	require.Equal(t, 1, fp.plays, "the song's plays")
	require.True(t, msc.Playing(), "Playing() after Open")

	return msc, fp
}

func Test_Music_Open(t *testing.T) {
	t.Parallel()

	t.Run("when a WAV file is opened, should play it", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		assert.True(t, p.playing, "the song isn't playing")
		assert.Equal(t, 1, p.plays, "the song's plays")
		assert.True(t, m.Playing(), "Playing() after Open")
	})

	t.Run("when a WAV file is opened, should decode it to 16-bit stereo at the output's rate", func(t *testing.T) {
		t.Parallel()

		_, p := openSong(t)

		// The resampled stream holds 16-bit stereo at the output's rate.
		n, err := io.Copy(io.Discard, p.src)

		assert.NoError(t, err)
		assert.Equal(t, int64(4*200), n, "decoded bytes")
	})

	t.Run("when a WMA file is opened, should fail with ErrFormat", func(t *testing.T) {
		t.Parallel()

		m, _ := openSong(t)

		assert.ErrorIs(t, m.Open("song.wma", []byte(wmaStart)), ErrFormat)
	})

	// A file that can't play closes the one before.
	t.Run("when a WMA file is opened, should close the music open before it", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		assert.ErrorIs(t, m.Open("song.wma", []byte(wmaStart)), ErrFormat)
		assert.True(t, p.closed, "the music before wasn't closed")
	})

	t.Run("when an MP3 is broken, should fail with ErrFormat", func(t *testing.T) {
		t.Parallel()

		m, _ := openSong(t)

		assert.ErrorIs(t, m.Open("broken.mp3", []byte("not an mp3")), ErrFormat)
	})
}

func Test_Music_Update(t *testing.T) {
	t.Parallel()

	t.Run("when the music is still playing, should not play it again", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		m.Update(true)

		assert.Equal(t, 1, p.plays, "the song's plays")
		assert.True(t, m.Playing(), "stopped playing")
	})

	t.Run("when the music has ended and repeat is off, should not play it again", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		p.playing = false
		p.pos = 400

		m.Update(false)

		assert.False(t, p.playing, "played again with repeat off")
		assert.False(t, m.Playing(), "Playing() with repeat off")
	})

	// The original's Timer1 checks on every tick, so turning repeat on after
	// the end plays it again too.
	t.Run("when repeat is turned on after the end, should play it again from its start", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		p.playing = false
		p.pos = 400

		m.Update(false) // at its end, repeat off
		m.Update(true)  // repeat turned on after the end

		assert.True(t, p.playing, "repeat: not playing")
		assert.Equal(t, 2, p.plays, "repeat: plays")
		assert.Zero(t, p.pos, "repeat: position")
	})

	t.Run("when the music was stopped, should not play it again", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		require.NoError(t, m.Stop())

		m.Update(true)

		assert.False(t, p.playing, "stopped music played again")
		assert.False(t, m.Playing(), "stopped music is Playing")
	})
}

func Test_Music_Stop(t *testing.T) {
	t.Parallel()

	t.Run("when nothing is open, should fail with ErrNotOpen", func(t *testing.T) {
		t.Parallel()

		assert.ErrorIs(t, new(Music).Stop(), ErrNotOpen)
	})

	t.Run("when the music plays, should pause it and rewind it", func(t *testing.T) {
		t.Parallel()

		m, p := openSong(t)

		p.pos = 100

		assert.NoError(t, m.Stop())
		assert.False(t, p.playing, "still playing")
		assert.Zero(t, p.pos, "position")
	})

	t.Run("when a failed Open has closed the music, should fail with ErrNotOpen", func(t *testing.T) {
		t.Parallel()

		m, _ := openSong(t)

		require.ErrorIs(t, m.Open("song.wma", []byte(wmaStart)), ErrFormat)

		assert.ErrorIs(t, m.Stop(), ErrNotOpen)
	})
}

func Test_format(t *testing.T) {
	t.Parallel()

	t.Run("when the data starts with a known signature, should tell the format from it", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "wav", format("a.bin", []byte("RIFF\x24\x00\x00\x00WAVEfmt ")))
		assert.Equal(t, "ogg", format("a.bin", []byte("OggS\x00\x02")))
		assert.Equal(t, "mp3", format("a.bin", []byte("ID3\x03\x00")))
		assert.Equal(t, "mp3", format("a.bin", []byte("\xff\xfb\x90\x64")))
	})

	t.Run("when the data has no signature, should tell the format from the extension in any letter case", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "mp3", format("A.MP3", nil))
		assert.Equal(t, "ogg", format("a.ogg", nil))
		assert.Equal(t, "ogg", format("B.OGG", nil))
		assert.Equal(t, "wav", format("a.Wav", nil))
	})

	t.Run("when the format is not one it plays, should tell none", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, format("a.mid", []byte("MThd")))
		assert.Empty(t, format("a.wma", []byte(wmaStart)))
	})
}
