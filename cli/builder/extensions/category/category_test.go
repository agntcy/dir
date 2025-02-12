// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package category

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	builder := New([]string{"category1", "category2"})

	// build
	gotExtension, err := builder.Build(context.Background())
	assert.Nil(t, err)

	specs, ok := gotExtension.Specs.(ExtensionSpecs)
	assert.True(t, ok)
	assert.Equal(t, specs.Categories, []string{"category1", "category2"})
}
