// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	"github.com/agntcy/dir/runtime/store/types"
	"github.com/agntcy/dir/runtime/utils"
	clientv3 "go.etcd.io/etcd/client/v3"
)

const StoreType types.StoreType = "etcd"

var logger = utils.NewLogger("store", "etcd")

// store provides storage operations for etcd.
type store struct {
	client          *clientv3.Client
	workloadsPrefix string
}

// New creates a new etcd store.
func New(cfg Config) (types.Store, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints(),
		DialTimeout: cfg.DialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	// Verify connection
	_, err = client.Status(context.Background(), cfg.Endpoints()[0])
	if err != nil {
		client.Close()

		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	logger.Info("connected to etcd", "endpoint", cfg.Endpoints()[0])

	return &store{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
	}, nil
}

// Close closes the etcd connection.
func (s *store) Close() error {
	if err := s.client.Close(); err != nil {
		return fmt.Errorf("failed to close etcd client: %w", err)
	}

	return nil
}

// RegisterWorkload stores a workload in etcd.
func (s *store) RegisterWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	key := s.workloadsPrefix + workload.GetId()

	data, err := json.Marshal(workload)
	if err != nil {
		return fmt.Errorf("failed to marshal workload: %w", err)
	}

	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put workload in etcd: %w", err)
	}

	logger.Info("registered workload", "workload", workload.GetId())

	return nil
}

// DeregisterWorkload removes a workload from etcd.
func (s *store) DeregisterWorkload(ctx context.Context, workloadID string) error {
	key := s.workloadsPrefix + workloadID

	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to delete workload from etcd: %w", err)
	}

	logger.Info("deregistered workload", "workload", workloadID)

	return nil
}

// PatchWorkload performs a full update of an existing workload in storage.
func (s *store) PatchWorkload(ctx context.Context, workload *runtimev1.Workload) error {
	key := s.workloadsPrefix + workload.GetId()

	// Get existing workload
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get workload from etcd: %w", err)
	}

	if len(resp.Kvs) == 0 {
		logger.Warn("workload not found for patch", "workload", workload.GetId())

		return nil
	}

	// Marshal and store workload
	data, err := json.Marshal(workload)
	if err != nil {
		return fmt.Errorf("failed to marshal workload: %w", err)
	}

	_, err = s.client.Put(ctx, key, string(data))
	if err != nil {
		return fmt.Errorf("failed to put workload in etcd: %w", err)
	}

	logger.Info("patched workload", "workload", workload.GetId())

	return nil
}

// ListWorkloadIDs returns all workload IDs in the store.
func (s *store) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, fmt.Errorf("failed to list workload IDs from etcd: %w", err)
	}

	ids := make(map[string]struct{})

	for _, kv := range resp.Kvs {
		id := strings.TrimPrefix(string(kv.Key), s.workloadsPrefix)
		ids[id] = struct{}{}
	}

	return ids, nil
}

// ListWorkloads returns all workloads in the store.
func (s *store) ListWorkloads(ctx context.Context) ([]*runtimev1.Workload, error) {
	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to get workloads from etcd: %w", err)
	}

	var workloads []*runtimev1.Workload

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		workloadID := strings.TrimPrefix(key, s.workloadsPrefix)

		var workload *runtimev1.Workload

		err := json.Unmarshal(kv.Value, &workload)
		if err != nil {
			logger.Error("failed to parse workload", "workload", workloadID, "error", err)

			continue
		}

		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// WatchWorkloads watches for workload changes.
func (s *store) WatchWorkloads(ctx context.Context, handler func(workload *runtimev1.Workload, deleted bool)) error {
	watchChan := s.client.Watch(ctx, s.workloadsPrefix, clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			//nolint:wrapcheck
			return ctx.Err()
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				logger.Error("watch error", "error", watchResp.Err())

				continue
			}

			for _, event := range watchResp.Events {
				switch event.Type {
				case clientv3.EventTypePut:
					var workload *runtimev1.Workload

					err := json.Unmarshal(event.Kv.Value, &workload)
					if err != nil {
						continue
					}

					handler(workload, false)
				case clientv3.EventTypeDelete:
					workloadID := strings.TrimPrefix(string(event.Kv.Key), s.workloadsPrefix)
					handler(&runtimev1.Workload{Id: workloadID}, true)
				}
			}
		}
	}
}
