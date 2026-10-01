// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
)

// startAdvertiseTask advertises everything this node has published, once at
// startup and then on every reprovide tick.
//
// The startup pass is what makes a restart cheap: provider records expire, and
// a ticker does not fire when it is created, so without it a restarted node
// would be invisible until the first interval elapsed.
func (r *routeRemote) startAdvertiseTask() {
	r.wg.Go(func() {
		if !r.waitForPeers(r.ctx) {
			return
		}

		// Anything that asked for a pass while we waited for a peer is about to
		// get one. Dropping the request here rather than after the pass is
		// deliberate: a request raised while the pass is already running has
		// missed it, and must survive to be served by the loop below.
		r.drainReadvertise()

		r.advertisePublishedRecords(r.ctx)

		ticker := time.NewTicker(r.reprovideInterval)
		defer ticker.Stop()

		remoteLogger.Info("Started reprovide task", "interval", r.reprovideInterval)

		for {
			select {
			case <-r.ctx.Done():
				remoteLogger.Debug("Stopping reprovide task")

				return
			case <-ticker.C:
				r.advertisePublishedRecords(r.ctx)
			case <-r.readvertise:
				remoteLogger.Debug("Re-advertising ahead of the next tick")
				r.advertisePublishedRecords(r.ctx)
			}
		}
	})
}

// waitForPeers blocks until the DHT has someone to advertise to, reporting
// false if the node shut down first.
//
// Provide against an empty routing table fails outright, so there is nothing to
// gain from starting earlier. A lone bootstrap node waits here until its first
// peer arrives, which is exactly when its records become worth announcing.
func (r *routeRemote) waitForPeers(ctx context.Context) bool {
	if r.server.DHT().RoutingTable().Size() > 0 {
		return true
	}

	remoteLogger.Info("Waiting for a DHT peer before advertising held records")

	ticker := time.NewTicker(advertisePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if r.server.DHT().RoutingTable().Size() > 0 {
				return true
			}
		}
	}
}

// advertisePublishedRecords announces every record this node has published:
// each CID so the record can be fetched, and the labels, with their ancestors,
// so it can be found.
//
// Labels are advertised as one distinct set rather than per record. Records
// share ancestors heavily — every skill under "/skills/A" reprovides that key —
// so deduplicating first is the difference between one Provide per label and
// one per record that carries it.
func (r *routeRemote) advertisePublishedRecords(ctx context.Context) {
	started := time.Now()

	cids, labels, err := r.publishedRecords(ctx)
	if err != nil {
		remoteLogger.Error("Failed to enumerate published records for advertising", "error", err)

		return
	}

	if len(cids) == 0 {
		remoteLogger.Debug("Nothing published to advertise")

		return
	}

	keys := make([]cid.Cid, 0, len(cids))

	for _, cidStr := range cids {
		decoded, err := cid.Decode(cidStr)
		if err != nil {
			remoteLogger.Warn("Skipping held record with an undecodable CID", "cid", cidStr, "error", err)

			continue
		}

		keys = append(keys, decoded)
	}

	expanded := expandLabels(labels)

	failedCIDs := r.provideKeys(ctx, keys)
	failedLabels := r.provideLabels(ctx, expanded)

	remoteLogger.Info("Advertised published records",
		"records", len(cids),
		"labelKeys", len(expanded),
		"failedRecords", failedCIDs,
		"failedLabels", failedLabels,
		"took", time.Since(started))
}

// publishedRecords returns the CIDs of every published record and the labels
// carried by them, paging so a large corpus is never loaded whole.
//
// Records this node merely holds are excluded: they stay servable to anyone who
// knows the CID, but nothing announces them. Unpublishing works the same way,
// by omission rather than withdrawal, because Kademlia has no retraction — the
// provider records already out there are left to expire.
func (r *routeRemote) publishedRecords(ctx context.Context) ([]string, []types.Label, error) {
	if r.db == nil {
		return nil, nil, errNoDatabase
	}

	var (
		allCIDs   []string
		allLabels []types.Label
	)

	for offset := 0; ; offset += advertisePageSize {
		if err := ctx.Err(); err != nil {
			return nil, nil, err //nolint:wrapcheck
		}

		cids, err := r.db.GetRecordCIDs(
			types.WithPublished(true),
			types.WithLimit(advertisePageSize),
			types.WithOffset(offset),
		)
		if err != nil {
			return nil, nil, err //nolint:wrapcheck
		}

		if len(cids) == 0 {
			break
		}

		labels, err := r.db.GetRecordLabels(cids)
		if err != nil {
			return nil, nil, err //nolint:wrapcheck
		}

		allCIDs = append(allCIDs, cids...)

		for _, recordLabels := range labels {
			allLabels = append(allLabels, recordLabels...)
		}

		if len(cids) < advertisePageSize {
			break
		}
	}

	return allCIDs, allLabels, nil
}

// provideLabels advertises each label as a DHT key and returns how many failed.
//
// Failures are counted rather than returned. A record stays reachable to
// anyone who already knows its CID, and the reprovide cycle retries, so
// aborting over one unreachable custodian would be worse than a partially
// advertised record.
func (r *routeRemote) provideLabels(ctx context.Context, labels []types.Label) int {
	keys := make([]cid.Cid, 0, len(labels))
	skipped := 0

	for _, label := range labels {
		key, err := labelKey(label)
		if err != nil {
			remoteLogger.Warn("Skipping label with no derivable DHT key", "label", label, "error", err)

			skipped++

			continue
		}

		keys = append(keys, key)
	}

	return skipped + r.provideKeys(ctx, keys)
}

// provideKeys advertises keys to the DHT concurrently and returns how many
// failed.
//
// Each Provide is a full Kademlia lookup followed by K AddProvider sends, so
// they run in parallel; done one at a time, a node holding a few hundred
// records would take hours to announce itself.
func (r *routeRemote) provideKeys(ctx context.Context, keys []cid.Cid) int {
	if len(keys) == 0 {
		return 0
	}

	var (
		wg     sync.WaitGroup
		failed atomic.Int64
	)

	pending := make(chan cid.Cid)

	for range min(advertiseConcurrency, len(keys)) {
		wg.Go(func() {
			for key := range pending {
				if err := r.server.DHT().Provide(ctx, key, true); err != nil {
					remoteLogger.Warn("Failed to announce key", "key", key, "error", err)
					failed.Add(1)
				}
			}
		})
	}

feed:
	for _, key := range keys {
		select {
		case pending <- key:
		case <-ctx.Done():
			break feed
		}
	}

	close(pending)
	wg.Wait()

	return int(failed.Load())
}
