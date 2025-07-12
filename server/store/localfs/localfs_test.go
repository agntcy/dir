// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package localfs

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/agntcy/dir/server/store/localfs/config"
	"github.com/agntcy/dir/server/store/testutil"
	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	ctx := t.Context()

	// Create store
	store, err := New(config.Config{Dir: os.TempDir()})
	assert.NoError(t, err, "failed to create store")

	// Define testing object
	data := []byte("test")
	object, err := testutil.CreateTestObjectWithDefaults(data)
	assert.NoError(t, err, "failed to create test object")

	// Push
	pushedObject, err := store.Push(ctx, object, bytes.NewReader(data))
	assert.NoError(t, err, "push failed")

	// Lookup
	fetchedMeta, err := store.Lookup(ctx, pushedObject)
	assert.NoError(t, err, "lookup failed")
	assert.Equal(t, pushedObject.CID(), fetchedMeta.CID())
	assert.Equal(t, pushedObject.Type(), fetchedMeta.Type())
	assert.Equal(t, pushedObject.Size(), fetchedMeta.Size())
	assert.Equal(t, pushedObject.Annotations(), fetchedMeta.Annotations())

	// Pull
	fetchedReader, err := store.Pull(ctx, pushedObject)
	assert.NoErrorf(t, err, "pull failed")

	fetchedContents, _ := io.ReadAll(fetchedReader)
	// TODO: fix chunking and sizing issues
	assert.Equal(t, data, fetchedContents[:len(data)])

	// Delete
	err = store.Delete(ctx, pushedObject)
	assert.NoErrorf(t, err, "delete failed")
}
