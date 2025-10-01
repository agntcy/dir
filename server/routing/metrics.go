// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package routing provides label frequency metrics for operational monitoring.
//
// The Metrics system tracks how many records are associated with each label
// (skills, domains, features) on the local peer. This data is persisted to
// the datastore and can be used for:
//
// - Operational monitoring and dashboards
// - Debugging label distribution issues
// - Future query optimization features
// - Administrative APIs and tooling
//
// Metrics are automatically maintained during Publish/Unpublish operations
// and stored at the "/metrics" datastore key in JSON format.
package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
)

// LabelMetric represents the frequency count for a specific label.
type LabelMetric struct {
	Name  string `json:"name"`  // Full label name (e.g., "/skills/AI/ML", "/domains/research")
	Total uint64 `json:"total"` // Number of local records that have this label
}

// Metrics tracks label frequency distribution for operational monitoring.
// This provides visibility into what types of records this peer is providing
// and can be used for debugging, monitoring, and future optimization features.
type Metrics struct {
	Data map[string]LabelMetric `json:"data"` // Map of label name → frequency count
}

func (m *Metrics) increment(label types.Label) {
	labelStr := label.String()
	if _, ok := m.Data[labelStr]; !ok {
		m.Data[labelStr] = LabelMetric{
			Name:  labelStr,
			Total: 0,
		}
	}

	m.Data[labelStr] = LabelMetric{
		Name:  labelStr,
		Total: m.Data[labelStr].Total + 1,
	}
}

func (m *Metrics) decrement(label types.Label) {
	labelStr := label.String()
	if _, ok := m.Data[labelStr]; !ok {
		return
	}

	currentTotal := m.Data[labelStr].Total
	if currentTotal > 0 {
		m.Data[labelStr] = LabelMetric{
			Name:  labelStr,
			Total: currentTotal - 1,
		}
	}

	// Remove the label from the map if the total is zero.
	if m.Data[labelStr].Total == 0 {
		delete(m.Data, labelStr)
	}
}

// NOTE: counts() method removed as it's no longer used in the new List API
// The new ListResponse doesn't include label_counts field for simplicity

// NOTE: labels() method removed as it's no longer used in the new List API
// The new List API doesn't return peer statistics for empty requests

func (m *Metrics) update(ctx context.Context, dstore types.Datastore) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics data: %w", err)
	}

	err = dstore.Put(ctx, datastore.NewKey("/metrics"), data)
	if err != nil {
		return fmt.Errorf("failed to update metrics data: %w", err)
	}

	return nil
}

func loadMetrics(ctx context.Context, dstore types.Datastore) (*Metrics, error) {
	// Fetch metrics data
	data, err := dstore.Get(ctx, datastore.NewKey("/metrics"))
	if err != nil {
		if errors.Is(err, datastore.ErrNotFound) {
			return &Metrics{
				Data: make(map[string]LabelMetric),
			}, nil
		}

		return nil, fmt.Errorf("failed to update metrics data: %w", err)
	}

	// Parse existing metrics data
	var metrics Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics data: %w", err)
	}

	return &metrics, nil
}
