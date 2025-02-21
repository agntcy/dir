// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"context"
	"encoding/json"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/database/types"
	ds "github.com/dep2p/libp2p/datastore"
	"github.com/dep2p/libp2p/datastore/query"
	"gorm.io/gorm"
)

const (
	agentTableName = "agents"
)

// AgentTable handles all database operations for agent data models.
//
// Following notes are true:
//   - The table is served via SQL. The constructor only needs the database instance
//     and to reserve a specific table name (use agentTableName).
//   - Keys are always constructed with the schema: /<namespace>/<agent-digest>.
//     Use AgentCID function on the received key in db ops to extract agent digest/CID.
//   - Value that is being exchanged (read/written/queried) is of type core.ObjectMeta.
//     We can create our own database object as well.
//   - We can use the StoreService to perform O(1) operations in Get, GetSize, Has
//     for optimization,but this can be done some other time.
type agentTable struct {
	db *gorm.DB
}

func NewAgentTable(db *gorm.DB) ds.Datastore {
	return &agentTable{
		db: db,
	}
}

func (s *agentTable) Get(ctx context.Context, key ds.Key) (value []byte, err error) {
	//TODO implement me
	panic("implement me")
}

func (s *agentTable) Has(ctx context.Context, key ds.Key) (exists bool, err error) {
	//TODO implement me
	panic("implement me")
}

// GetSize we can fake return back here
func (s *agentTable) GetSize(ctx context.Context, key ds.Key) (size int, err error) {
	//TODO implement me
	panic("implement me")
}

// Query dont implement now
func (s *agentTable) Query(ctx context.Context, q query.Query) (query.Results, error) {
	//TODO implement me
	panic("implement me")
}

func (s *agentTable) Put(ctx context.Context, key ds.Key, value []byte) error {
	var objectMeta coretypes.ObjectMeta
	if err := json.Unmarshal(value, &objectMeta); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	agent := types.Agent{
		Name:    objectMeta.Name,
		Version: objectMeta.Version,
		Digest:  objectMeta.Digest.Encode(),
	}

	return s.db.WithContext(ctx).Table(agentTableName).Save(&agent).Error
}

func (s *agentTable) Delete(ctx context.Context, key ds.Key) error {
	//TODO implement me
	panic("implement me")
}

// Sync dont implement now
func (s *agentTable) Sync(ctx context.Context, prefix ds.Key) error {
	//TODO implement me
	panic("implement me")
}

// Close should only close the actual table, not the database itself
func (s *agentTable) Close() error {
	//TODO implement me
	panic("implement me")
}

// AgentCID extracts the agent digest (UID) from the key.
// For example, for key=/<namespace>/<agent-digest>, we return digest=<agent-digest>
func AgentCID(key ds.Key) (*coretypes.Digest, error) {
	var digest coretypes.Digest
	if err := digest.Decode(key.BaseNamespace()); err != nil {
		return nil, err
	}
	return &digest, nil
}
