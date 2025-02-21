// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	key := ds.NewKey("/namespace/sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	digest := &coretypes.Digest{
		Type: coretypes.DigestType_DIGEST_TYPE_SHA256,
		Value: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}

	// Create an Agent object
	agent := &coretypes.Agent{
		Name:    "example-agent",
		Version: "1.0.0",
		Digest:  digest,
	}

	// Convert the Agent object to ObjectMeta
	objectMeta, err := agent.ObjectMeta()
	if err != nil {
		log.Fatalf("failed to convert agent to object meta: %v", err)
	}

	// Marshal the ObjectMeta to a byte slice
	value, err := json.Marshal(objectMeta)
	if err != nil {
		log.Fatalf("failed to marshal object meta: %v", err)
	}

	// Call the Put method
	err = db.Agent().Put(ctx, key, value)
	if err != nil {
		log.Fatalf("failed to put data: %v", err)
	}

	fmt.Println("Data successfully stored in the database")
}
