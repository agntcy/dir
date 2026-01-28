// Package storage provides server-side etcd storage with in-memory indices.
package storage

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/agntcy/dir/discovery/pkg/types"
)

// reader provides read-only etcd operations with in-memory indices for the server.
type reader struct {
	client          *clientv3.Client
	workloadsPrefix string

	// In-memory indices (protected by mutex)
	mu         sync.RWMutex
	workloads  map[string]*types.Workload     // id → Workload
	byHostname map[string]string              // hostname → id
	byName     map[string]string              // "namespace/name" or "name" → id
	byGroup    map[string]map[string]struct{} // group → {ids}

	// Watch control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewReader creates a new server storage with in-memory indices.
func NewReader(cfg Config) (types.StoreReader, error) {
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

	s := &reader{
		client:          client,
		workloadsPrefix: cfg.WorkloadsPrefix,
		workloads:       make(map[string]*types.Workload),
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

	// Start watches
	s.startWatches()

	return s, nil
}

// Close stops watches and closes the etcd connection.
func (s *reader) Close() error {
	s.cancel()
	s.wg.Wait()
	return s.client.Close()
}

func (s *reader) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.workloads)
}

// Get returns a workload by ID.
func (s *reader) Get(id string) (*types.Workload, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try by ID
	if workload := s.workloads[id]; workload != nil {
		return workload, nil
	}

	// Try by hostname
	if id, ok := s.byHostname[id]; ok {
		if workload := s.workloads[id]; workload != nil {
			return workload, nil
		}
	}

	// Try by name
	if id, ok := s.byName[id]; ok {
		if workload := s.workloads[id]; workload != nil {
			return workload, nil
		}
	}

	return nil, fmt.Errorf("workload not found: %s", id)
}

// List returns all workloads with optional filters.
func (s *reader) List(runtime types.RuntimeType, labelFilter map[string]string) []*types.Workload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*types.Workload
	for _, workload := range s.workloads {
		// Filter by runtime
		if runtime != "" && workload.Runtime != runtime {
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

		results = append(results, workload)
	}

	return results
}

// FindReachable finds all workloads reachable from the caller.
func (s *reader) FindReachable(callerIdentity string) (*types.ReachabilityResult, error) {
	caller, err := s.Get(callerIdentity)
	if err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	callerGroups := make(map[string]struct{})
	for _, g := range caller.IsolationGroups {
		callerGroups[g] = struct{}{}
	}

	if len(callerGroups) == 0 {
		return &types.ReachabilityResult{Caller: caller, Reachable: nil, Count: 0}, nil
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
	var reachable []*types.Workload
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
		filtered := &types.Workload{
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
			Metadata:        workload.Metadata,
		}
		reachable = append(reachable, filtered)
	}

	// Sort by name
	sort.Slice(reachable, func(i, j int) bool {
		return reachable[i].Name < reachable[j].Name
	})

	return &types.ReachabilityResult{
		Caller:    caller,
		Reachable: reachable,
		Count:     len(reachable),
	}, nil
}

// filterAddresses filters addresses to only those in shared isolation groups.
func (s *reader) filterAddresses(addresses []string, sharedGroups []string) []string {
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

// loadWorkloads loads all workloads from etcd.
func (s *reader) loadWorkloads() error {
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

		workload, err := types.FromJSON(kv.Value)
		if err != nil {
			log.Printf("[storage] Failed to parse workload %s: %v", workloadID, err)
			continue
		}

		s.updateWorkloadIndex(workloadID, workload)
	}

	log.Printf("[storage] Loaded %d workloads", len(s.workloads))
	return nil
}

// startWatches starts background watch goroutines.
func (s *reader) startWatches() {
	s.wg.Add(1)
	go s.watchWorkloads()
	log.Printf("[storage] Started watch thread")
}

// watchWorkloads watches for workload changes.
func (s *reader) watchWorkloads() {
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
					workload, err := types.FromJSON(event.Kv.Value)
					if err != nil {
						log.Printf("[storage] Watch: failed to parse workload %s: %v", workloadID, err)
						continue
					}
					s.updateWorkloadIndex(workloadID, workload)
					log.Printf("[storage] Watch: updated workload %s", workload.Name)

				case clientv3.EventTypeDelete:
					s.removeWorkloadIndex(workloadID)
					log.Printf("[storage] Watch: removed workload %s", workloadID[:12])
				}
			}
		}
	}
}

// updateWorkloadIndex updates in-memory indices for a workload.
func (s *reader) updateWorkloadIndex(workloadID string, workload *types.Workload) {
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
func (s *reader) removeWorkloadIndex(workloadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeWorkloadIndexLocked(workloadID)
}

// removeWorkloadIndexLocked removes workload from indices (must hold lock).
func (s *reader) removeWorkloadIndexLocked(workloadID string) {
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
