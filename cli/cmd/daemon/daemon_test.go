// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestEmbeddedZot tests the embedded Zot server.
func TestEmbeddedZot(t *testing.T) {
	address := "localhost:5000"
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
