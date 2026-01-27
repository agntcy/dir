// Package storage provides etcd storage implementations.
package storage

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

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

// =============================================================================
// WatcherStorage - Write-only storage for the watcher
// =============================================================================

// WatcherStorage provides write-only etcd operations for the watcher.
type WatcherStorage struct {
	client          *clientv3.Client
	workloadsPrefix string
}

// NewWatcherStorage creates a new watcher storage.
func NewWatcherStorage(cfg Config) (*WatcherStorage, error) {
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

	return &WatcherStorage{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
	}, nil
}

// Close closes the etcd connection.
func (s *WatcherStorage) Close() error {
	return s.client.Close()
}

// Register writes a workload to etcd.
func (s *WatcherStorage) Register(ctx context.Context, workload *models.Workload) error {
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
func (s *WatcherStorage) Deregister(ctx context.Context, workloadID string) error {
	key := s.workloadsPrefix + workloadID
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("[storage] Deregistered workload %s", workloadID[:12])
	return nil
}

// ListWorkloadIDs returns all workload IDs from etcd (keys only).
func (s *WatcherStorage) ListWorkloadIDs(ctx context.Context) (map[string]struct{}, error) {
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

// =============================================================================
// InspectorStorage - Read/Write storage for the inspector
// =============================================================================

// InspectorStorage provides read/write etcd operations for the inspector.
type InspectorStorage struct {
	client          *clientv3.Client
	workloadsPrefix string
	metadataPrefix  string
}

// NewInspectorStorage creates a new inspector storage.
func NewInspectorStorage(cfg Config) (*InspectorStorage, error) {
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

	return &InspectorStorage{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
		metadataPrefix:  cfg.MetadataPrefix,
	}, nil
}

// Close closes the etcd connection.
func (s *InspectorStorage) Close() error {
	return s.client.Close()
}

// ListWorkloads returns all workloads from storage.
func (s *InspectorStorage) ListWorkloads(ctx context.Context) ([]*models.Workload, error) {
	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	var workloads []*models.Workload
	for _, kv := range resp.Kvs {
		workload, err := models.FromJSON(kv.Value)
		if err != nil {
			log.Printf("[storage] Failed to parse workload: %v", err)
			continue
		}
		workloads = append(workloads, workload)
	}

	return workloads, nil
}

// ListWorkloadIDs returns all workload IDs from storage (keys only).
func (s *InspectorStorage) ListWorkloadIDs(ctx context.Context) ([]string, error) {
	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	var ids []string
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		id := strings.TrimPrefix(key, s.workloadsPrefix)
		if id != "" {
			ids = append(ids, id)
		}
	}

	log.Printf("[storage] Listed %d workload IDs from etcd", len(ids))
	return ids, nil
}

// GetWorkload returns a single workload by ID.
func (s *InspectorStorage) GetWorkload(ctx context.Context, workloadID string) (*models.Workload, error) {
	key := s.workloadsPrefix + workloadID
	resp, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	return models.FromJSON(resp.Kvs[0].Value)
}

// WatchWorkloads watches for workload changes and sends events to the channel.
func (s *InspectorStorage) WatchWorkloads(ctx context.Context, events chan<- *WorkloadEvent) error {
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
func (s *InspectorStorage) SetMetadata(ctx context.Context, workloadID, processorKey string, data interface{}) error {
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

// DeleteMetadata removes metadata for a workload.
func (s *InspectorStorage) DeleteMetadata(ctx context.Context, workloadID, processorKey string) error {
	key := s.metadataPrefix + workloadID + "/" + processorKey
	_, err := s.client.Delete(ctx, key)
	if err != nil {
		return err
	}

	log.Printf("[storage] Deleted metadata %s for %s", processorKey, workloadID[:12])
	return nil
}

// DeleteAllMetadata removes all metadata for a workload.
func (s *InspectorStorage) DeleteAllMetadata(ctx context.Context, workloadID string) error {
	prefix := s.metadataPrefix + workloadID + "/"
	_, err := s.client.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	log.Printf("[storage] Deleted all metadata for %s", workloadID[:12])
	return nil
}
