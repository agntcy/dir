// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
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
	benchChunk = bytes.Repeat([]byte{1}, 4096) // for checking chunking efficiency based on size
)

func TestStorePushLookupPullDelete(t *testing.T) {
	store := loadLocalStore(t)

	// Create test record
	agent := &objectsv1.Agent{
		Name:          "test-agent",
		SchemaVersion: "v1",
		Description:   "A test agent",
	}

	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: agent,
		},
	}

	// Calculate CID for the record
	recordCID := record.GetCid()
	assert.NotEmpty(t, recordCID, "failed to calculate CID")

	// Push operation
	recordRef, err := store.Push(testCtx, record)
	assert.NoErrorf(t, err, "push failed")
	assert.Equal(t, recordCID, recordRef.GetCid())

	// Lookup operation
	recordMeta, err := store.Lookup(testCtx, recordRef)
	assert.NoErrorf(t, err, "lookup failed")
	assert.Equal(t, recordCID, recordMeta.GetCid())

	// Pull operation
	pulledRecord, err := store.Pull(testCtx, recordRef)
	assert.NoErrorf(t, err, "pull failed")

	pulledCID := pulledRecord.GetCid()
	assert.NotEmpty(t, pulledCID, "failed to get pulled record CID")
	assert.Equal(t, recordCID, pulledCID)

	// Verify the pulled agent data
	pulledAgent := pulledRecord.GetV1()
	assert.NotNil(t, pulledAgent, "pulled agent should not be nil")
	assert.Equal(t, agent.GetName(), pulledAgent.GetName())
	assert.Equal(t, agent.GetSchemaVersion(), pulledAgent.GetSchemaVersion())
	assert.Equal(t, agent.GetDescription(), pulledAgent.GetDescription())

	// Delete operation
	err = store.Delete(testCtx, recordRef)
	assert.NoErrorf(t, err, "delete failed")

	// Lookup should fail after delete
	_, err = store.Lookup(testCtx, recordRef)
	assert.Error(t, err, "lookup should fail after delete")
	assert.ErrorContains(t, err, "not found")
}

func BenchmarkLocalStore(b *testing.B) {
	if !runLocal {
		b.Skip()
	}

	store := loadLocalStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func BenchmarkRemoteStore(b *testing.B) {
	if !runRemote {
		b.Skip()
	}

	store := loadRemoteStore(&testing.T{})
	for step := range b.N {
		benchmarkStep(store, append(benchChunk, []byte(strconv.Itoa(step))...))
	}
}

func benchmarkStep(store types.StoreAPI, testData []byte) {
	// Create test record
	agent := &objectsv1.Agent{
		Name:          "bench-agent",
		SchemaVersion: "v1",
		Description:   "A benchmark agent",
	}

	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: agent,
		},
	}

	// Record is ready for push operation

	// Push operation
	pushedRef, err := store.Push(testCtx, record)
	if err != nil {
		panic(err)
	}

	// Lookup operation
	fetchedMeta, err := store.Lookup(testCtx, pushedRef)
	if err != nil {
		panic(err)
	}

	// Assert equal
	if pushedRef.GetCid() != fetchedMeta.GetCid() {
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
