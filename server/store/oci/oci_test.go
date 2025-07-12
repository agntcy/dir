// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"bytes"
	"context"
	"io"
	"os"
	"strconv"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/store/testutil"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

// TODO: this should be configurable to unified Storage API test flow.
var (
	// test config.
	testConfig = ociconfig.Config{
		LocalDir:        os.TempDir(),                         // used for local test/bench
		RegistryAddress: "localhost:5000",                     // used for remote test/bench
		RepositoryName:  "test-store",                         // used for remote test/bench
		AuthConfig:      ociconfig.AuthConfig{Insecure: true}, // used for remote test/bench
	}
	runLocal = true
	// TODO: this may blow quickly when doing rapid benchmarking if not tested against fresh OCI instance.
	runRemote = false

	// common test.
	testCtx = context.Background()

	// common bench.
	benchObjectType = coretypes.ObjectType_OBJECT_TYPE_RAW // for object type to create
	benchChunk      = bytes.Repeat([]byte{1}, 4096)        // for checking chunking efficiency based on size
)

func TestStorePushLookupPullDelete(t *testing.T) {
	store := loadLocalStore(t)

	data := []byte("test")
	object, err := testutil.CreateTestObjectWithDefaults(data)
	assert.NoErrorf(t, err, "failed to create test object")

	objRef, err := store.Push(testCtx, object, bytes.NewReader(data))
	assert.NoErrorf(t, err, "push failed")
	assert.Equal(t, object.CID(), objRef.CID())
	assert.Equal(t, object.Type(), objRef.Type())
	assert.Equal(t, object.Size(), objRef.Size())
	assert.Equal(t, object.Annotations(), objRef.Annotations())

	// lookup op
	fetchedObject, err := store.Lookup(testCtx, objRef)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, object.CID(), fetchedObject.CID())
	assert.Equal(t, object.Type(), fetchedObject.Type())
	assert.Equal(t, object.Size(), fetchedObject.Size())
	assert.Equal(t, object.Annotations(), fetchedObject.Annotations())

	// pull op
	fetchedReader, err := store.Pull(testCtx, objRef)
	assert.NoErrorf(t, err, "pull failed")

	fetchedContents, _ := io.ReadAll(fetchedReader)
	assert.Equal(t, data, fetchedContents)

	// delete op
	err = store.Delete(testCtx, objRef)
	assert.NoErrorf(t, err, "delete failed")

	// lookup op
	_, err = store.Lookup(testCtx, objRef)
	assert.Error(t, err, "lookup should fail after delete")
	assert.ErrorContains(t, err, "object not found")
}

func BenchmarkLocalStore(b *testing.B) {
	if !runLocal {
		b.Skip()
	}

	store := loadLocalStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, benchObjectType, append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func BenchmarkRemoteStore(b *testing.B) {
	if !runRemote {
		b.Skip()
	}

	store := loadRemoteStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, benchObjectType, append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func benchmarkStep(store types.StoreAPI, objectType coretypes.ObjectType, objectData []byte) {
	// data to push
	objectRef := getRefForData(objectType, objectData, nil)

	// push op
	pushedRef, err := store.Push(testCtx, objectRef, bytes.NewReader(objectData))
	if err != nil {
		panic(err)
	}

	// lookup op
	fetchedRef, err := store.Lookup(testCtx, pushedRef)
	if err != nil {
		panic(err)
	}

	// assert equal
	if pushedRef.CID() != fetchedRef.CID() || pushedRef.Type() != fetchedRef.Type() || pushedRef.Size() != fetchedRef.Size() {
		panic("not equal lookup")
	}
}

func loadLocalStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create tmp storage for test artifacts
	tmpDir, err := os.MkdirTemp(testConfig.LocalDir, "test-oci-store-*") //nolint:usetesting
	assert.NoErrorf(t, err, "failed to create test dir")
	t.Cleanup(func() {
		err := os.RemoveAll(tmpDir)
		if err != nil {
			t.Fatalf("failed to cleanup: %v", err)
		}
	})

	// create local
	store, err := New(ociconfig.Config{LocalDir: tmpDir})
	assert.NoErrorf(t, err, "failed to create local store")

	return store
}

func loadRemoteStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create remote
	store, err := New(
		ociconfig.Config{
			RegistryAddress: testConfig.RegistryAddress,
			RepositoryName:  testConfig.RepositoryName,
			AuthConfig:      testConfig.AuthConfig,
		})
	assert.NoErrorf(t, err, "failed to create remote store")

	return store
}

func getRefForData(objType coretypes.ObjectType, data []byte, meta map[string]string) types.Object {
	object, err := testutil.CreateTestObject(testutil.TestObjectOptions{
		ObjectType:  objType,
		Data:        data,
		Annotations: meta,
	})
	if err != nil {
		panic(err)
	}

	return object
}
