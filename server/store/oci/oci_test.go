// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"bytes"
	"context"
	"io"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	// Skip manual test that requires zot to be running
	t.SkipNow()

	ctx := context.Background()
	config := Config{
		RegistryAddress: "localhost:5000",
		RepositoryName:  "test",
	}

	store, err := New(config)
	assert.NoError(t, err, "failed to create store")

	// Define testing object
	objContents := []byte("example!")
	objMeta := coretypes.ObjectMeta{
		Type: coretypes.ObjectType_OBJECT_TYPE_CUSTOM,
		Name: "example",
		Annotations: map[string]string{
			"label": "example",
		},
	}

	// Push
	digest, err := store.Push(ctx, &objMeta, bytes.NewReader(objContents))
	assert.NoError(t, err, "push failed")

	// Lookup
	fetchedMeta, err := store.Lookup(ctx, digest)
	assert.NoError(t, err, "lookup failed")
	assert.Equal(t, objMeta.GetType(), fetchedMeta.GetType())
	assert.Equal(t, objMeta.GetName(), fetchedMeta.GetName())
	assert.Equal(t, objMeta.GetAnnotations(), fetchedMeta.GetAnnotations())

	// Pull
	fetchedReader, err := store.Pull(ctx, digest)
	assert.NoError(t, err, "pull failed")

	fetchedContents, _ := io.ReadAll(fetchedReader)
	// TODO: fix chunking and sizing issues
	assert.Equal(t, objContents, fetchedContents[:len(objContents)])

	// Delete
	err = store.Delete(ctx, digest)
	assert.NoErrorf(t, err, "delete failed")
}
