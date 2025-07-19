// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package localfs

import (
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	"github.com/agntcy/dir/server/store/localfs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	ctx := context.Background()

	// Create store
	store, err := New(config.Config{Dir: t.TempDir()})
	require.NoError(t, err, "failed to create store")

	// Create test record
	testAgent := &objectsv1.Agent{
		Name:        "test-agent-123",
		Description: "A test agent for unit testing",
		Version:     "1.0.0",
	}

	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: testAgent,
		},
	}

	// Calculate CID (as the controller would do)
	recordCID := record.GetCid()
	require.NotEmpty(t, recordCID, "failed to calculate CID")

	// Push
	pushedRef, err := store.Push(ctx, record)
	require.NoError(t, err, "push failed")
	assert.Equal(t, recordCID, pushedRef.GetCid(), "pushed CID should match calculated CID")

	// Lookup
	fetchedMeta, err := store.Lookup(ctx, pushedRef)
	require.NoError(t, err, "lookup failed")
	assert.Equal(t, recordCID, fetchedMeta.GetCid(), "fetched CID should match")
	assert.Equal(t, "v1", fetchedMeta.GetSchemaVersion(), "schema version should be v1")
	// TODO: where the annotations are?
	// assert.NotNil(t, fetchedMeta.GetAnnotations(), "annotations should not be nil")

	// Pull
	fetchedRecord, err := store.Pull(ctx, pushedRef)
	require.NoError(t, err, "pull failed")

	fetchedCID := fetchedRecord.GetCid()
	require.NotEmpty(t, fetchedCID, "failed to get fetched record CID")
	assert.Equal(t, recordCID, fetchedCID, "pulled record CID should match")

	// Verify record data
	assert.NotNil(t, fetchedRecord.GetV1(), "should have v1 data")
	fetchedAgent := fetchedRecord.GetV1()
	assert.Equal(t, testAgent.GetName(), fetchedAgent.GetName(), "agent name should match")
	assert.Equal(t, testAgent.GetDescription(), fetchedAgent.GetDescription(), "agent description should match")
	assert.Equal(t, testAgent.GetVersion(), fetchedAgent.GetVersion(), "agent version should match")

	// Delete
	err = store.Delete(ctx, pushedRef)
	require.NoError(t, err, "delete failed")

	// Verify deletion - lookup should fail
	_, err = store.Lookup(ctx, pushedRef)
	assert.Error(t, err, "lookup should fail after deletion")
}
