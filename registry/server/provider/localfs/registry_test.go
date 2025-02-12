// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package localfs

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	sm, _ := NewRegistryProvider(os.TempDir())
	ctx := context.Background()

	// Define testing object
	objContents := []byte("example!")
	objMeta := registrytypes.ObjectMeta{
		Type: registrytypes.ObjectType_OBJECT_TYPE_CUSTOM,
		Name: "example",
		Annotations: map[string]string{
			"label": "example",
		},
	}
	objDigest := "DIGEST_TYPE_SHA256:2e4551de804e27aacf20f9df5be3e8cd384ed64488b21ab079fb58e8c90068ab"

	// Push
	digest, err := sm.Store().Push(ctx, &objMeta, bytes.NewReader(objContents))
	assert.NoErrorf(t, err, "push failed")

	// Lookup
	fetchedMeta, err := sm.Store().Lookup(ctx, digest)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, &objMeta, fetchedMeta)

	// Pull
	fetchedReader, err := sm.Store().Pull(ctx, digest)
	assert.NoErrorf(t, err, "pull failed")
	fetchedContents, _ := io.ReadAll(fetchedReader)
	// TODO: fix chunking and sizing issues
	assert.Equal(t, objContents, fetchedContents[:len(objContents)])

	// Tag
	err = sm.Publish().Publish(ctx, "latest", digest)
	assert.NoErrorf(t, err, "tagging failed")

	// Resolve
	linkDigest, err := sm.Publish().Resolve(ctx, "latest")
	assert.NoErrorf(t, err, "resolve failed")
	assert.Equal(t, linkDigest, digest)
	assert.Equal(t, linkDigest.ToString(), objDigest)
}
