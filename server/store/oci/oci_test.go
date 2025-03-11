package oci

import (
	"bytes"
	"context"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"io"
	"testing"

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
	assert.NoErrorf(t, err, "failed to create store")

	// Define testing object
	objContents := []byte("example!")
	objRef := storetypes.ObjectRef{
		Type: ptrTo[string]("example-type"),
		Name: ptrTo[string]("example-name"),
		Size: ptrTo[uint64](123),
	}

	// Push
	digest, err := store.Push(ctx, &objRef, bytes.NewReader(objContents))
	assert.NoErrorf(t, err, "push failed")

	// Lookup
	fetchedRef, err := store.Lookup(ctx, digest)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, objRef.Type, fetchedRef.Type)
	assert.Equal(t, objRef.Name, fetchedRef.Name)
	// assert.Equal(t, objRef.Annotations, fetchedRef.Annotations)

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
