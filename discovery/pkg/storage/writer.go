// Package storage provides etcd storage implementations.
package storage

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/agntcy/dir/discovery/pkg/types"
	models "github.com/agntcy/dir/discovery/pkg/types"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EventType represents the type of workload event.
type EventType string

const (
	EventPut    EventType = "PUT"
	EventDelete EventType = "DELETE"
)

// WorkloadEvent represents a workload change event.
type WorkloadEvent struct {
	Type       EventType
	WorkloadID string
	Workload   *models.Workload
}

// writer provides write-only etcd operations for the discovery.
type writer struct {
	client          *clientv3.Client
	workloadsPrefix string
	metadataPrefix  string
}

// NewWriter creates a new writer storage for discovery.
func NewWriter(cfg Config) (types.StoreWriter, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints(),
		DialTimeout: cfg.DialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, err
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Status(ctx, cfg.Endpoints()[0])
	if err != nil {
		client.Close()
		return nil, err
	}

	log.Printf("[storage] Connected to etcd at %s", cfg.Endpoints()[0])

	return &writer{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
		metadataPrefix:  cfg.MetadataPrefix,
	}, nil
}

// Close closes the etcd connection.
func (s *writer) Close() error {
	return s.client.Close()
}

// Register writes a workload to etcd.
func (s *writer) RegisterWorkload(ctx context.Context, workload *models.Workload) error {
	key := s.workloadsPrefix + workload.ID
	value, err := workload.ToJSON()
	if err != nil {
		return err
	}

	_, err = s.client.Put(ctx, key, string(value))
	if err != nil {
		return err
	}

	log.Printf("[storage] Registered workload %s", workload.Name)
	return nil
}

// Deregister removes a workload from etcd.
func (s *writer) DeregisterWorkload(ctx context.Context, workloadID string) error {
	// Delete workload data
	key := s.workloadsPrefix + workloadID
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("[storage] Deregistered workload %s", workloadID[:12])

	// Delete all metadata for this workload
	prefix := s.metadataPrefix + workloadID + "/"
	_, err = s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	log.Printf("[storage] Deleted all metadata for %s", workloadID[:12])

	return nil
}

// ListWorkloadIDs returns all workload IDs from etcd (keys only).
func (s *writer) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	ids := make(map[string]struct{})
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		id := strings.TrimPrefix(key, s.workloadsPrefix)
		if id != "" {
			ids[id] = struct{}{}
		}
	}

	log.Printf("[storage] Listed %d workload IDs from etcd", len(ids))
	return ids, nil
}

// WatchWorkloads watches for workload changes and sends events to the channel.
func (s *writer) WatchWorkloads(ctx context.Context, events chan<- *WorkloadEvent) error {
	watchChan := s.client.Watch(ctx, s.workloadsPrefix, clientv3.WithPrefix())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				log.Printf("[storage] Watch error: %v", watchResp.Err())
				continue
			}

			for _, event := range watchResp.Events {
				key := string(event.Kv.Key)
				workloadID := strings.TrimPrefix(key, s.workloadsPrefix)

				switch event.Type {
				case clientv3.EventTypePut:
					workload, err := models.FromJSON(event.Kv.Value)
					if err != nil {
						log.Printf("[storage] Failed to parse workload %s: %v", workloadID, err)
						continue
					}
					events <- &WorkloadEvent{
						Type:       EventPut,
						WorkloadID: workloadID,
						Workload:   workload,
					}
					log.Printf("[storage] Watch: workload updated: %s", workload.Name)

				case clientv3.EventTypeDelete:
					events <- &WorkloadEvent{
						Type:       EventDelete,
						WorkloadID: workloadID,
					}
					log.Printf("[storage] Watch: workload deleted: %s", workloadID[:12])
				}
			}
		}
	}
}

// SetMetadata writes metadata for a workload.
func (s *writer) SetMetadata(ctx context.Context, workloadID, processorKey string, data interface{}) error {
	key := s.metadataPrefix + workloadID + "/" + processorKey

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.client.Put(ctx, key, string(jsonData))
	if err != nil {
		return err
	}

	log.Printf("[storage] Set metadata %s for %s", processorKey, workloadID[:12])
	return nil
}
