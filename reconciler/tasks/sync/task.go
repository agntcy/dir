// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package sync implements the sync reconciler task.
// It searches the routing network for records matching Criteria and creates
// one sync per announcing peer. Defaults match the dirctl sync-netsec CronJob.
package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/sync")

var errNoPeerAddress = errors.New("peer has no /dir/ or /oci/ address")

// Searcher looks up remote records on the routing network.
// Satisfied by types.RoutingAPI (daemon) and reconciler/routing.Client
// (standalone).
type Searcher interface {
	Search(ctx context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error)
}

// SyncCreator persists a sync row for the server sync worker.
type SyncCreator interface {
	CreateSync(remoteURL string, cids []string, remoteRegistryURL, repositoryName string) (string, error)
}

// Task implements the sync reconciler task.
type Task struct {
	config   Config
	searcher Searcher
	db       SyncCreator
}

// NewTask creates a new sync task.
func NewTask(config Config, searcher Searcher, db SyncCreator) (*Task, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Task{
		config:   config,
		searcher: searcher,
		db:       db,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "sync"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run searches routing for Criteria and creates one sync per peer.
func (t *Task) Run(ctx context.Context) error {
	domain := t.config.GetDomain()
	limit := uint32(t.config.GetLimit()) //nolint:gosec // GetLimit is a small configured cap.
	minScore := uint32(1)

	logger.Debug("Running sync", "domain", domain, "limit", limit)

	results, err := t.searcher.Search(ctx, &routingv1.SearchRequest{
		Queries: []*routingv1.RecordQuery{{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: domain,
		}},
		MinMatchScore: &minScore,
		Limit:         &limit,
	})
	if err != nil {
		return fmt.Errorf("search routing: %w", err)
	}

	hits, err := collectSearch(ctx, results)
	if err != nil {
		return err
	}

	peers := groupByPeer(hits)
	if len(peers) == 0 {
		logger.Info("No remote records to sync", "domain", domain)

		return nil
	}

	if t.config.DryRun {
		for _, peer := range peers {
			logger.Info("Dry run: not creating sync",
				"peer", peer.id,
				"dir", peer.dirAddr,
				"oci_registry", peer.ociRegistry,
				"oci_repository", peer.ociRepo,
				"cids", peer.cids,
			)
		}

		logger.Info("Dry run: sync complete", "peers", len(peers), "domain", domain)

		return nil
	}

	var succeeded, failed int

	for _, peer := range peers {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("sync canceled: %w", err)
		}

		if err := t.createPeerSync(peer); err != nil {
			logger.Warn("Failed to create sync", "peer", peer.id, "error", err)

			failed++

			continue
		}

		succeeded++
	}

	logger.Info("Sync complete", "succeeded", succeeded, "failed", failed, "domain", domain)

	return nil
}

func (t *Task) createPeerSync(peer *peerGroup) error {
	switch {
	case peer.dirAddr != "":
		id, err := t.db.CreateSync(peer.dirAddr, peer.cids, "", "")
		if err != nil {
			return fmt.Errorf("create dir sync: %w", err)
		}

		logger.Info("Sync created", "peer", peer.id, "remote", peer.dirAddr, "cids", len(peer.cids), "sync_id", id)
	case peer.ociRegistry != "":
		id, err := t.db.CreateSync("", peer.cids, peer.ociRegistry, peer.ociRepo)
		if err != nil {
			return fmt.Errorf("create oci sync: %w", err)
		}

		logger.Info("Sync created",
			"peer", peer.id,
			"registry", peer.ociRegistry,
			"repository", peer.ociRepo,
			"cids", len(peer.cids),
			"sync_id", id,
		)
	default:
		return errNoPeerAddress
	}

	return nil
}

func collectSearch(ctx context.Context, results <-chan *routingv1.SearchResponse) ([]*routingv1.SearchResponse, error) {
	var hits []*routingv1.SearchResponse

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("collect search: %w", ctx.Err())
		case hit, ok := <-results:
			if !ok {
				return hits, nil
			}

			if hit != nil {
				hits = append(hits, hit)
			}
		}
	}
}
