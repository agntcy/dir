package oci

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
)

func TestOCI(t *testing.T) {
	// Create local OCI store for testing
	store, err := New(config.Config{
		RegistryAddress: "localhost:5000",
		RepositoryName:  "dir",
		AuthConfig: config.AuthConfig{
			Insecure: true,
		},
	})
	assert.Nil(t, err)

	// Push some example data
	baseIn := "Hello, world!"
	baseRef, err := store.PushData(t.Context(), io.NopCloser(strings.NewReader(baseIn)))
	assert.Nil(t, err)

	// Lookup the data back
	lookBase, err := store.Lookup(t.Context(), baseRef)
	fmt.Println(lookBase)
	assert.Nil(t, err)

	// Pull the data back
	pulledData, err := store.Pull(t.Context(), baseRef)
	assert.Nil(t, err)
	dataBytes, err := io.ReadAll(pulledData)
	assert.Nil(t, err)
	assert.Equal(t, baseIn, string(dataBytes))

	// Create test object
	obj := &storev1.Object{
		Annotations: map[string]string{
			"key": "value",
		},
		Cid:  baseRef.GetCid(),
		Size: lookBase.GetSize(),
		Links: []*storev1.ObjectLink{
			{
				Cid:  baseRef.GetCid(),
				Size: lookBase.GetSize(),
				Annotations: map[string]string{
					"link-key": "link-value",
				},
			},
		},
	}

	// Push object as OCI manifest
	objRef, err := store.PushObject(t.Context(), obj)
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

	// Create nested object
	nestedObj := &storev1.Object{
		Annotations: map[string]string{
			"nested-key": "nested-value",
		},
		Cid:  objRef.GetCid(),
		Size: lookObj.GetSize(),
		Links: []*storev1.ObjectLink{
			{
				Cid:  objRef.GetCid(),
				Size: lookObj.GetSize(),
				Annotations: map[string]string{
					"obj-link-key": "obj-link-value",
				},
			},
		},
	}

	// Push nested object
	nestedRef, err := store.PushObject(t.Context(), nestedObj)
	assert.Nil(t, err)

	// Lookup nested object back
	lookNested, err := store.Lookup(t.Context(), nestedRef)
	assert.Nil(t, err)
	assert.True(t, reflect.DeepEqual(nestedObj, lookNested))

	// Pull nested object back
	pulledNested, err := store.Pull(t.Context(), nestedRef)
	assert.Nil(t, err)
	dataBytes, err = io.ReadAll(pulledNested)
	assert.Nil(t, err)
	assert.Equal(t, baseIn, string(dataBytes))
}
