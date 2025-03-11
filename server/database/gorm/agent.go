// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"context"
	"encoding/json"
	"errors"
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

type agentTable struct {
	db *gorm.DB
}

func NewAgentTable(db *gorm.DB) ds.Datastore {
	return &agentTable{
		db: db,
	}
}

func (s *agentTable) Get(ctx context.Context, key ds.Key) ([]byte, error) {
	keyDigest, err := AgentCID(key)
	if err != nil {
		return nil, fmt.Errorf("failed to extract agent digest: %w", err)
	}

	var agent types.Agent

	err = s.db.WithContext(ctx).
		Table(agentTableName).
		Where("c_id = ?", keyDigest.Encode()).
		Where("deleted_at IS NULL").
		First(&agent).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ds.ErrNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// Create ObjectMeta from the agent data
	objectMeta := &coretypes.ObjectMeta{
		Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT,
		Digest: keyDigest,
		Name:   agent.Name,
	}

	// Marshal the ObjectMeta to JSON
	marshalledObjectMeta, err := json.Marshal(objectMeta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object meta: %w", err)
	}

	return marshalledObjectMeta, nil
}

func (s *agentTable) Has(ctx context.Context, key ds.Key) (bool, error) {
	keyDigest, err := AgentCID(key)
	if err != nil {
		return false, fmt.Errorf("failed to extract agent digest: %w", err)
	}

	var count int64

	err = s.db.WithContext(ctx).
		Table(agentTableName).
		Where("c_id = ?", keyDigest.Encode()).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check agent existence: %w", err)
	}

	return count > 0, nil
}

func (s *agentTable) GetSize(ctx context.Context, key ds.Key) (int, error) {
	// Get the actual data
	data, err := s.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	// Return the size of the serialized data
	return len(data), nil
}

func (s *agentTable) Query(_ context.Context, _ query.Query) (query.Results, error) {
	// TODO implement me
	panic("implement me")
}

func (s *agentTable) Put(ctx context.Context, key ds.Key, value []byte) error {
	var objectMeta coretypes.ObjectMeta
	if err := json.Unmarshal(value, &objectMeta); err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	if objectMeta.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT {
		return fmt.Errorf("invalid object type: %v", objectMeta.GetType())
	}

	if objectMeta.GetDigest() == nil {
		return errors.New("missing digest in object meta")
	}

	keyDigest, err := AgentCID(key)
	if err != nil {
		return fmt.Errorf("failed to extract agent digest: %w", err)
	}

	if keyDigest.String() != objectMeta.GetDigest().String() {
		return fmt.Errorf("digest mismatch: %v != %v", keyDigest, objectMeta.GetDigest())
	}

	// Check if the agent already exists
	exists, err := s.Has(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check if agent exists: %w", err)
	}

	if exists {
		return fmt.Errorf("agent already exists with this key: %v", key.BaseNamespace())
	}

	agent := types.Agent{
		Model: types.Model{
			CID: objectMeta.GetDigest().Encode(),
		},
		Name: objectMeta.GetName(),
	}

	return s.db.WithContext(ctx).Table(agentTableName).Create(&agent).Error
}

func (s *agentTable) Delete(ctx context.Context, key ds.Key) error {
	keyDigest, err := AgentCID(key)
	if err != nil {
		return fmt.Errorf("failed to extract agent digest: %w", err)
	}

	result := s.db.WithContext(ctx).
		Table(agentTableName).
		Where("c_id = ?", keyDigest.Encode()).
		Delete(&types.Agent{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete agent: %w", result.Error)
	}

	// If no rows were affected, the record didn't exist
	if result.RowsAffected == 0 {
		return ds.ErrNotFound
	}

	return nil
}

// Sync dont implement now.
func (s *agentTable) Sync(_ context.Context, _ ds.Key) error {
	// TODO implement me
	panic("implement me")
}

func (s *agentTable) Close() error {
	// Since we're using GORM and the database connection is managed externally,
	// we don't need to do any specific cleanup for just the table.
	// The actual database connection closure should be handled by the main application.
	return nil
}

// For example, for key=/<namespace>/<agent-digest>, we return digest=<agent-digest>.
func AgentCID(key ds.Key) (*coretypes.Digest, error) {
	var digest coretypes.Digest
	if err := digest.Decode(key.BaseNamespace()); err != nil {
		return nil, fmt.Errorf("failed to decode agent digest: %w", err)
	}

	return &digest, nil
}
