// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sql

import (
	"fmt"
	"time"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// workload represents a workload data in the database.
type workloadRecord struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ID        string `gorm:"column:id;primaryKey;not null"`
	Payload   []byte `gorm:"column:payload;not null"`
}

func newWorkloadRecord(runtimeWorkload *runtimev1.Workload) (*workloadRecord, error) {
	if runtimeWorkload == nil {
		return nil, fmt.Errorf("workload is nil")
	}

	if runtimeWorkload.GetId() == "" {
		return nil, fmt.Errorf("workload ID is empty")
	}

	payload, err := protojson.Marshal(runtimeWorkload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal workload: %w", err)
	}

	return &workloadRecord{
		ID:      runtimeWorkload.GetId(),
		Payload: payload,
	}, nil
}

func (w *workloadRecord) ToRuntimeWorkload() (*runtimev1.Workload, error) {
	workload := &runtimev1.Workload{}

	err := protojson.Unmarshal(w.Payload, workload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal workload payload for %s: %w", w.ID, err)
	}

	return workload, nil
}
