package localfs

import (
	"bytes"
	"context"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore(t *testing.T) {
	ctx := context.Background()

	// Create store
	store, err := New(os.TempDir())
	assert.NoErrorf(t, err, "failed to create store")

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
	assert.NoErrorf(t, err, "push failed")

	// Lookup
	fetchedMeta, err := store.Lookup(ctx, digest)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, objMeta.Type, fetchedMeta.Type)
	assert.Equal(t, objMeta.Name, fetchedMeta.Name)
	assert.Equal(t, objMeta.Annotations, fetchedMeta.Annotations)

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
