// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content/oci"
)

func TestDeleteFromOCIStore_NotFound(t *testing.T) {
	// Create an empty OCI store which will return errdef.ErrNotFound for any non-existent CID
	tmpDir := t.TempDir()
	ociStore, err := oci.New(tmpDir)
	require.NoError(t, err)

	s := &store{
		repo: ociStore,
	}

	// Attempting to delete a non-existent CID should return nil (success)
	err = s.deleteFromOCIStore(context.Background(), "bafybeib76j4m6pfr253a652y5z2ndifewfhztdv73cyb25yhtv4b3445nm")
	assert.NoError(t, err)
}
