// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	builder := New([]string{"author1", "author2"})

	// build
	gotExtension, err := builder.Build(context.Background())
	assert.Nil(t, err)

	specs, ok := gotExtension.Specs.(ExtensionSpecs)
	assert.True(t, ok)
	assert.Equal(t, specs.Authors, []string{"author1", "author2"})
}
