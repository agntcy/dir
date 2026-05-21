// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/agntcy/dir/config"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigUsesMacOSFriendlyLocalRegistryPort(t *testing.T) {
	originalOpts := opts
	opts = &Options{DataDir: t.TempDir()}
	t.Cleanup(func() {
		opts = originalOpts
	})

	cfg, err := loadConfig()

	require.NoError(t, err)
	// The registry is hoisted to the canonical top-level Config struct;
	// both apiserver and reconciler now read it from cfg.Registry.
	require.Equal(t, "localhost:5555", cfg.Registry.RegistryAddress)
}

// TestEmbeddedZot tests the embedded Zot server.
func TestEmbeddedZot(t *testing.T) {
	address := config.DefaultRegistryAddress
	rootDirectory := "/tmp/agntcy/dir/oci/"

	go func() {
		ctx := runEmbeddedZot(context.Background(), address, rootDirectory)

		defer ctx.Done()
	}()

	var (
		zotIsReady bool
		err        error
	)

	for range 10 {
		zotIsReady, err = isZotReady(address)
		if err == nil && zotIsReady {
			break
		}

		time.Sleep(1 * time.Second)
	}

	require.NoError(t, err)
	require.True(t, zotIsReady)
}
