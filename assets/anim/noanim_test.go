// SPDX-License-Identifier: AGPL-3.0-only

//go:build noanim

package anim

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Load(t *testing.T) {
	t.Parallel()

	t.Run("when built without the animations, should find none", func(t *testing.T) {
		t.Parallel()

		_, err := Load(25)
		assert.Error(t, err)
	})
}
