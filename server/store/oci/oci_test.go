package oci

import (
	"io"
	"reflect"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
)

func TestOCI(t *testing.T) {
	// Create local OCI store for testing
	store, err := New(config.Config{LocalDir: "./testdata/oci"})
	assert.Nil(t, err)

	// Push some example data
	baseIn := "Hello, world!"
	base := io.NopCloser(strings.NewReader(baseIn))
	baseRef, err := store.PushData(t.Context(), base)
	assert.Nil(t, err)

	// Lookup the data back
	_, err = store.Lookup(t.Context(), baseRef)
	assert.Nil(t, err)

	// Pull the data back
	pulledData, err := store.Pull(t.Context(), baseRef)
	assert.Nil(t, err)
	dataBytes, err := io.ReadAll(pulledData)
	assert.Nil(t, err)
	assert.Equal(t, baseIn, string(dataBytes))

	// Create test object
	obj := &corev1.Object{
		Schema: &corev1.ObjectSchema{
			Type:    "text",
			Version: "1.0",
			Format:  "plain",
		},
		Annotations: map[string]string{
			"key": "value",
		},
		Data: &corev1.ObjectRef{
			Cid: baseRef.GetCid(),
		},
		Links: []*corev1.Object{
			{
				Data: &corev1.ObjectRef{
					Cid: baseRef.GetCid(),
				},
				Schema: &corev1.ObjectSchema{
					Type:    "link",
					Version: "1.0.0.0.0",
					Format:  "plain2",
				},
				Annotations: map[string]string{
					"link-key": "link-value",
				},
			},
		},
	}

	// Push object as OCI manifest
	objRef, err := store.Push(t.Context(), obj)
	assert.Nil(t, err)

	// Lookup object back
	lookObj, err := store.Lookup(t.Context(), objRef)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(obj, lookObj))

	// Pull object back
	pulledObj, err := store.Pull(t.Context(), objRef)
	assert.Nil(t, err)
	dataBytes, err = io.ReadAll(pulledObj)
	assert.Nil(t, err)
	assert.Equal(t, baseIn, string(dataBytes))
}
