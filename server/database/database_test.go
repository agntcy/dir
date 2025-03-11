// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"encoding/json"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/config"
	ds "github.com/dep2p/libp2p/datastore"
)

func TestDatabase(t *testing.T) {
	db, err := NewDatabase(&config.Config{
		DBDriver:    "gorm",
		DatabaseDSN: "/tmp/sqlite/dir.db",
	})
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Create a context
	ctx := context.Background()

	// Create a key
	key := ds.NewKey("/agents/blobs/sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	digest := &coretypes.Digest{
		Type:  coretypes.DigestType_DIGEST_TYPE_SHA256,
		Value: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	objectMeta := &coretypes.ObjectMeta{
		Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT,
		Name:   "example-agent:1.0.0",
		Digest: digest,
	}

	// Marshal the ObjectMeta to a byte slice
	value, err := json.Marshal(objectMeta)
	if err != nil {
		t.Fatalf("failed to marshal object meta: %v", err)
	}

	// Call the Has method
	exists, err := db.Agent().Has(ctx, key)
	if err != nil {
		t.Fatalf("failed to check if key exists: %v", err)
	}

	if !exists {
		t.Log("key does not exist")

		// Call the Put method
		err = db.Agent().Put(ctx, key, value)
		if err != nil {
			t.Fatalf("failed to put data: %v", err)
		}

		t.Log("Data successfully stored in the database")
	} else {
		t.Logf("Key %s exists in the database", key.BaseNamespace())
	}

	// Call the Get method
	value, err = db.Agent().Get(ctx, key)
	if err != nil {
		t.Fatalf("failed to get data: %v", err)
	}

	t.Logf("Data successfully retrieved from the database: %s", string(value))

	// Call the GetSize method
	size, err := db.Agent().GetSize(ctx, key)
	if err != nil {
		t.Fatalf("failed to get size of data: %v", err)
	}

	t.Logf("Size of data: %d bytes", size)

	// Call the Delete method
	err = db.Agent().Delete(ctx, key)
	if err != nil {
		t.Fatalf("failed to delete data: %v", err)
	}

	t.Log("Data successfully deleted from the database")
}
