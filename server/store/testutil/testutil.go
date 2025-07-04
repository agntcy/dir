// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package testutil

import (
	"bytes"
	"context"
	"io"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	cid "github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestObjectOptions provides options for creating test objects
type TestObjectOptions struct {
	ObjectType  coretypes.ObjectType
	Data        []byte
	Annotations map[string]string
}

// CreateTestObject creates a test object with the given data and options
func CreateTestObject(opts TestObjectOptions) (types.Object, error) {
	if opts.Data == nil {
		opts.Data = []byte("test")
	}

	if opts.ObjectType == coretypes.ObjectType_OBJECT_TYPE_UNSPECIFIED {
		opts.ObjectType = coretypes.ObjectType_OBJECT_TYPE_RAW
	}

	if opts.Annotations == nil {
		opts.Annotations = map[string]string{
			"name": "test",
		}
	}

	coreObject := &coretypes.Object{
		Type:        opts.ObjectType,
		Data:        opts.Data,
		Annotations: opts.Annotations,
		CreatedAt:   timestamppb.Now().AsTime().Format(time.RFC3339),
		Size:        uint64(len(opts.Data)),
	}

	pref := cid.Prefix{
		Version:  1, // CIDv1
		Codec:    uint64(coreObject.Type),
		MhType:   mh.SHA2_256, // SHA2-256 hash function
		MhLength: -1,          // default length (32 bytes for SHA2-256)
	}

	c, err := pref.Sum(opts.Data)
	if err != nil {
		return nil, err
	}

	coreObject.Cid = c.String()
	object := adapters.NewObjectV1(coreObject)

	return object, nil
}

// CreateTestObjectWithDefaults creates a test object with default values
func CreateTestObjectWithDefaults(data []byte) (types.Object, error) {
	return CreateTestObject(TestObjectOptions{
		Data: data,
	})
}

// TestStoreOperations performs a complete test of Push -> Lookup -> Pull -> Delete operations
func TestStoreOperations(t assert.TestingT, store types.StoreAPI, ctx context.Context, testData []byte) {
	// Create test object
	object, err := CreateTestObjectWithDefaults(testData)
	assert.NoError(t, err, "failed to create test object")

	// Push
	pushedObject, err := store.Push(ctx, object, bytes.NewReader(testData))
	assert.NoError(t, err, "push failed")
	assert.Equal(t, object.CID(), pushedObject.CID())
	assert.Equal(t, object.Type(), pushedObject.Type())
	assert.Equal(t, object.Size(), pushedObject.Size())
	assert.Equal(t, object.Annotations(), pushedObject.Annotations())

	// Lookup
	fetchedObject, err := store.Lookup(ctx, pushedObject)
	assert.NoError(t, err, "lookup failed")
	assert.Equal(t, object.CID(), fetchedObject.CID())
	assert.Equal(t, object.Type(), fetchedObject.Type())
	assert.Equal(t, object.Size(), fetchedObject.Size())
	assert.Equal(t, object.Annotations(), fetchedObject.Annotations())

	// Pull
	fetchedReader, err := store.Pull(ctx, pushedObject)
	assert.NoError(t, err, "pull failed")
	defer fetchedReader.Close()

	fetchedContents, err := io.ReadAll(fetchedReader)
	assert.NoError(t, err, "failed to read pulled contents")
	assert.Equal(t, testData, fetchedContents)

	// Delete
	err = store.Delete(ctx, pushedObject)
	assert.NoError(t, err, "delete failed")

	// Verify deletion
	_, err = store.Lookup(ctx, pushedObject)
	assert.Error(t, err, "lookup should fail after delete")
}
