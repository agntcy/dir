// Package storage provides server-side etcd storage with in-memory indices.
package storage

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	models "github.com/agntcy/dir/discovery/pkg/types"
)

// ServerStorage provides read-only etcd operations with in-memory indices for the server.
type ServerStorage struct {
	client          *clientv3.Client
	workloadsPrefix string
	metadataPrefix  string

	// In-memory indices (protected by mutex)
	mu         sync.RWMutex
	workloads  map[string]*models.Workload       // id → Workload
	metadata   map[string]map[string]interface{} // id → {processor: data}
	byHostname map[string]string                 // hostname → id
	byName     map[string]string                 // "namespace/name" or "name" → id
	byGroup    map[string]map[string]struct{}    // group → {ids}

	// Watch control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewServerStorage creates a new server storage with in-memory indices.
func NewServerStorage(cfg Config) (*ServerStorage, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints(),
		DialTimeout: cfg.DialTimeout,
		Username:    cfg.Username,
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, err
	}

	connCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = client.Status(connCtx, cfg.Endpoints()[0])
	if err != nil {
		client.Close()
		return nil, err
	}

	ctx, ctxCancel := context.WithCancel(context.Background())

	s := &ServerStorage{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
		metadataPrefix:  cfg.MetadataPrefix,
		workloads:       make(map[string]*models.Workload),
		metadata:        make(map[string]map[string]interface{}),
		byHostname:      make(map[string]string),
		byName:          make(map[string]string),
		byGroup:         make(map[string]map[string]struct{}),
		ctx:             ctx,
		cancel:          ctxCancel,
	}

	log.Printf("[storage] Connected to etcd at %s", cfg.Endpoints()[0])

	// Load initial data
	if err := s.loadWorkloads(); err != nil {
		client.Close()
		return nil, err
	}
	if err := s.loadMetadata(); err != nil {
		log.Printf("[storage] Warning: failed to load metadata: %v", err)
	}

	// Start watches
	s.startWatches()

	return s, nil
}

// Close stops watches and closes the etcd connection.
func (s *ServerStorage) Close() error {
	s.cancel()
	s.wg.Wait()
	return s.client.Close()
}

// Get returns a workload by ID.
func (s *ServerStorage) Get(id string) *models.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workload := s.workloads[id]
	if workload != nil {
		return s.workloadWithMetadata(workload)
	}
	return nil
}

// GetByHostname returns a workload by hostname.
func (s *ServerStorage) GetByHostname(hostname string) *models.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if id, ok := s.byHostname[hostname]; ok {
		if workload := s.workloads[id]; workload != nil {
			return s.workloadWithMetadata(workload)
		}
	}
	return nil
}

// GetByName returns a workload by name (and optional namespace).
func (s *ServerStorage) GetByName(name, namespace string) *models.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var id string
	if namespace != "" {
		id = s.byName[namespace+"/"+name]
	}
	if id == "" {
		id = s.byName[name]
	}

	if id != "" {
		if workload := s.workloads[id]; workload != nil {
			return s.workloadWithMetadata(workload)
		}
	}
	return nil
}

// ListAll returns all workloads with optional filters.
func (s *ServerStorage) ListAll(runtime string, labelFilter map[string]string) []*models.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*models.Workload
	for _, workload := range s.workloads {
		// Filter by runtime
		if runtime != "" && string(workload.Runtime) != runtime {
			continue
		}

		// Filter by labels
		if labelFilter != nil {
			match := true
			for k, v := range labelFilter {
				if workload.Labels[k] != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}

		results = append(results, s.workloadWithMetadata(workload))
	}

	return results
}

// IdentifyWorkload finds a workload by hostname, name, or ID.
func (s *ServerStorage) IdentifyWorkload(identity string) *models.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try hostname first
	if id, ok := s.byHostname[identity]; ok {
		return s.workloadWithMetadata(s.workloads[id])
	}

	// Try name
	if id, ok := s.byName[identity]; ok {
		return s.workloadWithMetadata(s.workloads[id])
	}

	// Try ID directly
	if workload, ok := s.workloads[identity]; ok {
		return s.workloadWithMetadata(workload)
	}

	// Try ID prefix
	for id, workload := range s.workloads {
		if strings.HasPrefix(id, identity) {
			return s.workloadWithMetadata(workload)
		}
	}

	return nil
}

// FindReachable finds all workloads reachable from the caller.
func (s *ServerStorage) FindReachable(callerIdentity string) (*models.ReachabilityResult, error) {
	caller := s.IdentifyWorkload(callerIdentity)
	if caller == nil {
		return nil, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	callerGroups := make(map[string]struct{})
	for _, g := range caller.IsolationGroups {
		callerGroups[g] = struct{}{}
	}

	if len(callerGroups) == 0 {
		return &models.ReachabilityResult{Caller: caller, Reachable: nil, Count: 0}, nil
	}

	// Find all workloads sharing at least one group
	reachableIDs := make(map[string]struct{})
	for group := range callerGroups {
		if ids, ok := s.byGroup[group]; ok {
			for id := range ids {
				reachableIDs[id] = struct{}{}
			}
		}
	}
	delete(reachableIDs, caller.ID)

	// Build result list with filtered addresses
	var reachable []*models.Workload
	for id := range reachableIDs {
		workload := s.workloads[id]
		if workload == nil {
			continue
		}

		workloadGroups := make(map[string]struct{})
		for _, g := range workload.IsolationGroups {
			workloadGroups[g] = struct{}{}
		}

		// Find shared groups
		var sharedGroups []string
		for g := range callerGroups {
			if _, ok := workloadGroups[g]; ok {
				sharedGroups = append(sharedGroups, g)
			}
		}

		// Filter addresses
		filteredAddrs := s.filterAddresses(workload.Addresses, sharedGroups)

		// Create filtered workload
		filtered := &models.Workload{
			ID:           workload.ID,
			Name:         workload.Name,
			Hostname:     workload.Hostname,
			Runtime:      workload.Runtime,
			WorkloadType: workload.WorkloadType,
			// Node:            workload.Node,
			// Namespace:       workload.Namespace,
			Addresses:       filteredAddrs,
			IsolationGroups: sharedGroups,
			Ports:           workload.Ports,
			Labels:          workload.Labels,
			Annotations:     workload.Annotations,
			Metadata:        s.mergeMetadata(workload),
		}
		reachable = append(reachable, filtered)
	}

	// Sort by name
	sort.Slice(reachable, func(i, j int) bool {
		return reachable[i].Name < reachable[j].Name
	})

	return &models.ReachabilityResult{
		Caller:    caller,
		Reachable: reachable,
		Count:     len(reachable),
	}, nil
}

// filterAddresses filters addresses to only those in shared isolation groups.
func (s *ServerStorage) filterAddresses(addresses []string, sharedGroups []string) []string {
	sharedSet := make(map[string]struct{})
	for _, g := range sharedGroups {
		sharedSet[g] = struct{}{}
	}

	var filtered []string
	for _, addr := range addresses {
		parts := strings.Split(addr, ".")
		if len(parts) >= 3 && (parts[len(parts)-1] == "pod" || parts[len(parts)-1] == "svc") {
			// Kubernetes format
			namespace := parts[len(parts)-2]
			if _, ok := sharedSet[namespace]; ok {
				filtered = append(filtered, addr)
			}
		} else if len(parts) == 2 {
			// Docker format: {name}.{network}
			network := parts[1]
			if _, ok := sharedSet[network]; ok {
				filtered = append(filtered, addr)
			}
		} else {
			filtered = append(filtered, addr)
		}
	}
	return filtered
}

// workloadWithMetadata returns a copy of workload with merged metadata.
func (s *ServerStorage) workloadWithMetadata(workload *models.Workload) *models.Workload {
	result := *workload
	result.Metadata = s.mergeMetadata(workload)
	return &result
}

// mergeMetadata merges stored metadata with workload metadata.
func (s *ServerStorage) mergeMetadata(workload *models.Workload) map[string]interface{} {
	stored := s.metadata[workload.ID]
	if stored == nil && workload.Metadata == nil {
		return nil
	}

	merged := make(map[string]interface{})
	for k, v := range workload.Metadata {
		merged[k] = v
	}
	for k, v := range stored {
		merged[k] = v
	}
	return merged
}

// loadWorkloads loads all workloads from etcd.
func (s *ServerStorage) loadWorkloads() error {
	log.Printf("[storage] Loading workloads from %s", s.workloadsPrefix)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.client.Get(ctx, s.workloadsPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		workloadID := strings.TrimPrefix(key, s.workloadsPrefix)

		workload, err := models.FromJSON(kv.Value)
		if err != nil {
			log.Printf("[storage] Failed to parse workload %s: %v", workloadID, err)
			continue
		}

		s.updateWorkloadIndex(workloadID, workload)
	}

	log.Printf("[storage] Loaded %d workloads", len(s.workloads))
	return nil
}

// loadMetadata loads all metadata from etcd.
func (s *ServerStorage) loadMetadata() error {
	log.Printf("[storage] Loading metadata from %s", s.metadataPrefix)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.client.Get(ctx, s.metadataPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		relativeKey := strings.TrimPrefix(key, s.metadataPrefix)
		parts := strings.SplitN(relativeKey, "/", 2)

		if len(parts) >= 2 {
			workloadID := parts[0]
			processor := parts[1]

			var data interface{}
			if err := json.Unmarshal(kv.Value, &data); err != nil {
				log.Printf("[storage] Failed to parse metadata %s/%s: %v", workloadID, processor, err)
				continue
			}

			s.mu.Lock()
			if s.metadata[workloadID] == nil {
				s.metadata[workloadID] = make(map[string]interface{})
			}
			s.metadata[workloadID][processor] = data
			s.mu.Unlock()
		}
	}

	log.Printf("[storage] Loaded metadata for %d workloads", len(s.metadata))
	return nil
}

// startWatches starts background watch goroutines.
func (s *ServerStorage) startWatches() {
	s.wg.Add(2)
	go s.watchWorkloads()
	go s.watchMetadata()
	log.Printf("[storage] Started watch threads")
}

// watchWorkloads watches for workload changes.
func (s *ServerStorage) watchWorkloads() {
	defer s.wg.Done()

	watchChan := s.client.Watch(s.ctx, s.workloadsPrefix, clientv3.WithPrefix())
	log.Printf("[storage] Watching workloads at %s", s.workloadsPrefix)

	for {
		select {
		case <-s.ctx.Done():
			return
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				log.Printf("[storage] Workload watch error: %v", watchResp.Err())
				continue
			}

			for _, event := range watchResp.Events {
				key := string(event.Kv.Key)
				workloadID := strings.TrimPrefix(key, s.workloadsPrefix)

				switch event.Type {
				case clientv3.EventTypePut:
					workload, err := models.FromJSON(event.Kv.Value)
					if err != nil {
						log.Printf("[storage] Watch: failed to parse workload %s: %v", workloadID, err)
						continue
					}
					s.updateWorkloadIndex(workloadID, workload)
					log.Printf("[storage] Watch: updated workload %s", workload.Name)

				case clientv3.EventTypeDelete:
					s.removeWorkloadIndex(workloadID)
					s.mu.Lock()
					delete(s.metadata, workloadID)
					s.mu.Unlock()
					log.Printf("[storage] Watch: removed workload %s", workloadID[:12])
				}
			}
		}
	}
}

// watchMetadata watches for metadata changes.
func (s *ServerStorage) watchMetadata() {
	defer s.wg.Done()

	watchChan := s.client.Watch(s.ctx, s.metadataPrefix, clientv3.WithPrefix())
	log.Printf("[storage] Watching metadata at %s", s.metadataPrefix)

	for {
		select {
		case <-s.ctx.Done():
			return
		case watchResp := <-watchChan:
			if watchResp.Err() != nil {
				log.Printf("[storage] Metadata watch error: %v", watchResp.Err())
				continue
			}

			for _, event := range watchResp.Events {
				key := string(event.Kv.Key)
				relativeKey := strings.TrimPrefix(key, s.metadataPrefix)
				parts := strings.SplitN(relativeKey, "/", 2)

				if len(parts) < 2 {
					continue
				}

				workloadID := parts[0]
				processor := parts[1]

				switch event.Type {
				case clientv3.EventTypePut:
					var data interface{}
					if err := json.Unmarshal(event.Kv.Value, &data); err != nil {
						log.Printf("[storage] Watch: failed to parse metadata: %v", err)
						continue
					}
					s.mu.Lock()
					if s.metadata[workloadID] == nil {
						s.metadata[workloadID] = make(map[string]interface{})
					}
					s.metadata[workloadID][processor] = data
					s.mu.Unlock()

				case clientv3.EventTypeDelete:
					s.mu.Lock()
					if s.metadata[workloadID] != nil {
						delete(s.metadata[workloadID], processor)
					}
					s.mu.Unlock()
				}
			}
		}
	}
}

// updateWorkloadIndex updates in-memory indices for a workload.
func (s *ServerStorage) updateWorkloadIndex(workloadID string, workload *models.Workload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old entries
	s.removeWorkloadIndexLocked(workloadID)

	// Add new entries
	s.workloads[workloadID] = workload

	if workload.Hostname != "" {
		s.byHostname[workload.Hostname] = workloadID
	}

	// if workload.Namespace != "" {
	// 	s.byName[workload.Namespace+"/"+workload.Name] = workloadID
	// }
	s.byName[workload.Name] = workloadID

	for _, group := range workload.IsolationGroups {
		if s.byGroup[group] == nil {
			s.byGroup[group] = make(map[string]struct{})
		}
		s.byGroup[group][workloadID] = struct{}{}
	}
}

// removeWorkloadIndex removes workload from all indices.
func (s *ServerStorage) removeWorkloadIndex(workloadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeWorkloadIndexLocked(workloadID)
}

// removeWorkloadIndexLocked removes workload from indices (must hold lock).
func (s *ServerStorage) removeWorkloadIndexLocked(workloadID string) {
	workload := s.workloads[workloadID]
	if workload == nil {
		return
	}

	if workload.Hostname != "" && s.byHostname[workload.Hostname] == workloadID {
		delete(s.byHostname, workload.Hostname)
	}

	nameKey := workload.Name
	// if workload.Namespace != "" {
	// 	nameKey = workload.Namespace + "/" + workload.Name
	// }
	if s.byName[nameKey] == workloadID {
		delete(s.byName, nameKey)
	}
	if s.byName[workload.Name] == workloadID {
		delete(s.byName, workload.Name)
	}

	for _, group := range workload.IsolationGroups {
		if s.byGroup[group] != nil {
			delete(s.byGroup[group], workloadID)
			if len(s.byGroup[group]) == 0 {
				delete(s.byGroup, group)
			}
		}
	}

	delete(s.workloads, workloadID)
}
