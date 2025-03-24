// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

type LabelMetric struct {
	Name  string // label name
	Total uint64 // total items assigned to the label
}

type Metrics map[string]LabelMetric

func (m *Metrics) increment(label string) {
	if _, ok := (*m)[label]; !ok {
		(*m)[label] = LabelMetric{
			Name:  label,
			Total: 0,
		}
	}

	(*m)[label] = LabelMetric{
		Name:  label,
		Total: (*m)[label].Total + 1,
	}
}

func (m *Metrics) load(ctx context.Context, dstore types.Datastore) error {
	res, err := dstore.Query(ctx, query.Query{
		Prefix: "/metrics",
	})
	if err != nil {
		return fmt.Errorf("failed to query datastore: %w", err)
	}

	entries, err := res.Rest()
	if err != nil {
		return fmt.Errorf("failed to parse metrics data: %w", err)
	}

	if len(entries) > 1 {
		return fmt.Errorf("unexpected number of metrics entries: %d", len(entries))
	}

	if len(entries) == 0 {
		metrics := make(Metrics)
		*m = metrics

		return nil
	}

	// Parse existing metrics
	metrics := make(Metrics)

	err = json.Unmarshal(entries[0].Value, &metrics)
	if err != nil {
		return fmt.Errorf("failed to unmarshal metrics data: %w", err)
	}

	*m = metrics

	return nil
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
