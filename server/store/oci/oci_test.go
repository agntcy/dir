// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

// TODO: this should be configurable to unified Storage API test flow.
var (
	// test config.
	testAgentPath = "./testdata/agent.json"
	testConfig    = ociconfig.Config{
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
	benchObjectType = coretypes.ObjectType_OBJECT_TYPE_AGENT // for object type to create
	benchChunk      = bytes.Repeat([]byte{1}, 4096)          // for checking chunking efficiency based on size
)

func TestStore(t *testing.T) {
	store := loadLocalStore(t)

	// load agent
	agentRaw, err := os.ReadFile(testAgentPath)
	assert.NoErrorf(t, err, "failed to load test agent")

	agent := &coretypes.Agent{}
	err = json.Unmarshal(agentRaw, &agent)
	assert.NoErrorf(t, err, "failed to parse test agent")

	objRef := getRefForData(coretypes.ObjectType_OBJECT_TYPE_AGENT.String(), agentRaw, map[string]string{
		"name":       agent.GetName(),
		"version":    agent.GetVersion(),
		"created_at": agent.GetCreatedAt(),
	})

	// push op
	dgst, err := store.Push(testCtx, objRef, bytes.NewReader(agentRaw))
	assert.NoErrorf(t, err, "push failed")

	// lookup op
	fetchedRef, err := store.Lookup(testCtx, dgst)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, *objRef, *fetchedRef) //nolint:govet

	// pull op
	fetchedReader, err := store.Pull(testCtx, dgst)
	assert.NoErrorf(t, err, "pull failed")

	fetchedContents, _ := io.ReadAll(fetchedReader)
	assert.Equal(t, agentRaw, fetchedContents)

	// delete op
	// todo: verify delete op via lookup
	err = store.Delete(testCtx, dgst)
	assert.NoErrorf(t, err, "delete failed")
}

func BenchmarkLocalStore(b *testing.B) {
	if !runLocal {
		b.Skip()
	}

	store := loadLocalStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, benchObjectType.String(), append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func BenchmarkRemoteStore(b *testing.B) {
	if !runRemote {
		b.Skip()
	}

	store := loadRemoteStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, benchObjectType.String(), append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func benchmarkStep(store types.StoreAPI, objectType string, objectData []byte) {
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
	if pushedRef.GetDigest() != fetchedRef.GetDigest() || pushedRef.GetType() != fetchedRef.GetType() || pushedRef.GetSize() != fetchedRef.GetSize() {
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
	store, err := New(ociconfig.Config{
		RegistryAddress: testConfig.RegistryAddress,
		RepositoryName:  testConfig.RepositoryName,
		AuthConfig:      testConfig.AuthConfig,
	})
	assert.NoErrorf(t, err, "failed to create remote store")

	return store
}

func getRefForData(objType string, data []byte, meta map[string]string) *coretypes.ObjectRef {
	return &coretypes.ObjectRef{
		Type:        objType,
		Digest:      digest.FromBytes(data).String(),
		Size:        uint64(len(data)),
		Annotations: meta,
	}
}
