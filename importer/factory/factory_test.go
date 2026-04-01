// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package factory

import (
	"context"
	"testing"

	"github.com/agntcy/dir/importer/config"
)

func TestCreate_UnsupportedRegistry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	_, err := Create(ctx, nil, config.Config{Type: "unknown"})
	if err == nil {
		t.Fatal("expected error for unsupported registry type")
	}
}
