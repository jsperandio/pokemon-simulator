// SPDX-License-Identifier: AGPL-3.0-only

package delphi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_DecodeANSI(t *testing.T) {
	t.Parallel()

	t.Run("when a byte is in Latin-1's range, should decode it as Latin-1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "POKéMON", DecodeANSI([]byte("POK\xe9MON")))
		assert.Equal(t, "2500¥", DecodeANSI([]byte("2500\xa5")))
	})

	t.Run("when a byte is one of cp1252's own, should decode it as cp1252", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "€™", DecodeANSI([]byte("\x80\x99")))
	})

	t.Run("when a byte is undefined in cp1252, should decode it to the same code point", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "\u0081", DecodeANSI([]byte("\x81")))
	})

	t.Run("when there are no bytes, should return an empty string", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, DecodeANSI(nil))
	})
}

func Test_EncodeANSI(t *testing.T) {
	t.Parallel()

	t.Run("when a character is in Latin-1's range, should encode it as Latin-1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []byte("POK\xe9MON"), EncodeANSI("POKéMON"))
		assert.Equal(t, []byte("2500\xa5"), EncodeANSI("2500¥"))
	})

	t.Run("when a character is one of cp1252's own, should encode it as cp1252", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []byte("\x80\x99"), EncodeANSI("€™"))
	})

	t.Run("when a character is undefined in cp1252, should encode it to the same byte", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []byte("\x81"), EncodeANSI("\u0081"))
	})

	t.Run("when cp1252 can't hold a character, should write a question mark", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []byte("??"), EncodeANSI("日本"))
	})

	t.Run("when the text is empty, should return no bytes", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, EncodeANSI(""))
	})
}
