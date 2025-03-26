// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
)

type LabelMetric struct {
	Name  string `json:"name"`  // label name
	Total uint64 `json:"total"` // total items assigned to the label
}

type Metrics struct {
	Data map[string]LabelMetric `json:"data"`
}

func (m *Metrics) increment(label string) {
	if _, ok := m.Data[label]; !ok {
		m.Data[label] = LabelMetric{
			Name:  label,
			Total: 0,
		}
	}

	m.Data[label] = LabelMetric{
		Name:  label,
		Total: m.Data[label].Total + 1,
	}
}

func (m *Metrics) counts() map[string]uint64 {
	counts := make(map[string]uint64)
	for _, metric := range m.Data {
		counts[metric.Name] = metric.Total
	}

	return counts
}

func (m *Metrics) labels() []string {
	labels := make([]string, 0, len(m.Data))
	for _, metric := range m.Data {
		labels = append(labels, metric.Name)
	}

	return labels
}

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
