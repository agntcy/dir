// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithSource(t *testing.T) {
	t.Parallel()

	ctx := WithSource(t.Context(), SourcePush)
	assert.Equal(t, SourcePush, SourceFrom(ctx))
	assert.Empty(t, SourceFrom(t.Context()))
}
