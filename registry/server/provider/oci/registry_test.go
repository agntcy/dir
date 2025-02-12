package oci

import (
	"bytes"
	"context"
	"io"
	"testing"

	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
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

	r, err := NewRegistryProvider(config)
	assert.NoErrorf(t, err, "failed to create registry provider")

	// Define testing object
	objContents := []byte("example!")
	objMeta := registrytypes.ObjectMeta{
		Type: registrytypes.ObjectType_OBJECT_TYPE_CUSTOM,
		Name: "example",
		Annotations: map[string]string{
			"label": "example",
		},
	}

	// Push
	digest, err := r.Store().Push(ctx, &objMeta, bytes.NewReader(objContents))
	assert.NoErrorf(t, err, "push failed")

	// Lookup
	fetchedMeta, err := r.Store().Lookup(ctx, digest)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, objMeta.Type, fetchedMeta.Type)
	assert.Equal(t, objMeta.Name, fetchedMeta.Name)
	assert.Equal(t, objMeta.Annotations, fetchedMeta.Annotations)

	// Pull
	fetchedReader, err := r.Store().Pull(ctx, digest)
	assert.NoErrorf(t, err, "pull failed")
	fetchedContents, _ := io.ReadAll(fetchedReader)
	// TODO: fix chunking and sizing issues
	assert.Equal(t, objContents, fetchedContents[:len(objContents)])

	// Delete
	err = r.Store().Delete(ctx, digest)
	assert.NoErrorf(t, err, "delete failed")
}
