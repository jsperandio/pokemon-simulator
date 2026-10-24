// SPDX-License-Identifier: AGPL-3.0-only

//go:build !js

package classic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_memo_resized(t *testing.T) {
	t.Parallel()

	t.Run("when the Battle Info log grows wider, should wrap its lines again and move the scroll bar to its edge", func(t *testing.T) {
		t.Parallel()

		mem := newTestApp(t).info.memo
		mem.setLines([]string{
			strings.Repeat("Charmander used Scratch! ", 4),
		})

		require.Len(t, mem.wraps, 2, "the line before")

		rect := mem.rect
		rect.Max.X += 189
		mem.setBounds(rect)

		assert.Len(t, mem.wraps, 1, "the line after")
		assert.Equal(t, rect.Dx()-2, mem.scroll.rect.Max.X, "the scroll bar's right edge")
	})
}
