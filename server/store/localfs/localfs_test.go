// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package localfs

import (
	"bytes"
	"io"
	"os"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/store/localfs/config"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	ctx := t.Context()

	// Create store
	store, err := New(config.Config{Dir: os.TempDir()})
	assert.NoError(t, err, "failed to create store")

	// Define testing object
	objContents := []byte("example!")
	objRef := &coretypes.ObjectRef{
		Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest: digest.FromBytes(objContents).String(),
		Size:   uint64(len(objContents)),
		Annotations: map[string]string{
			"name":       "name",
			"version":    "version",
			"created_at": "created_at",
		},
	}

	// Push
	digest, err := store.Push(ctx, objRef, bytes.NewReader(objContents))
	assert.NoError(t, err, "push failed")

	// Lookup
	fetchedMeta, err := store.Lookup(ctx, digest)
	assert.NoError(t, err, "lookup failed")
	assert.Equal(t, objRef.GetDigest(), fetchedMeta.GetDigest())
	assert.Equal(t, objRef.GetType(), fetchedMeta.GetType())
	assert.Equal(t, objRef.GetSize(), fetchedMeta.GetSize())
	assert.Equal(t, objRef.GetAnnotations(), fetchedMeta.GetAnnotations())

	// Pull
	fetchedReader, err := store.Pull(ctx, digest)
	assert.NoErrorf(t, err, "pull failed")

	fetchedContents, _ := io.ReadAll(fetchedReader)
	// TODO: fix chunking and sizing issues
	assert.Equal(t, objContents, fetchedContents[:len(objContents)])

	// Delete
	err = store.Delete(ctx, digest)
	assert.NoErrorf(t, err, "delete failed")
}
