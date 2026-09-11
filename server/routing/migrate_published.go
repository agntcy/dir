// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
)

var migrateLogger = logging.Logger("routing/migrate")

const (
	// v1RecordsPrefix is where v1 recorded which records this node published.
	// v2 keeps that on the record row instead, so an upgraded node has to be
	// told what it used to advertise or it comes back holding everything and
	// announcing nothing.
	v1RecordsPrefix = "/records/"

	// badgerManifest exists in any directory Badger has opened. Its absence
	// means there is no v1 datastore to migrate, and checking for it is what
	// keeps a fresh install from having an empty one created underneath it.
	badgerManifest = "MANIFEST"
)

// startPublishedMigrationTask recovers v1's publish state in the background.
//
// Backgrounded rather than done before the node serves, because opening the v1
// datastore replays its value log, and that database holds the whole label
// cache rather than just the handful of keys wanted here. A legacy datastore
// that is slow or damaged should cost a log line, not a failed start.
//
// The cost of not blocking is a race with the startup advertise pass, which may
// run before the flags are written and miss the records this recovers. Rather
// than gate startup on the migration, it asks for another pass afterwards:
// advertising is idempotent, so a second pass is cheaper than the coupling.
func (r *routeRemote) startPublishedMigrationTask(baseDir string) {
	r.wg.Go(func() {
		migrated, err := migratePublishedFromV1(r.ctx, baseDir, r.db)
		if err != nil {
			migrateLogger.Error("Failed to migrate v1 publish state", "error", err)
		}

		if migrated > 0 {
			r.requestReadvertise()
		}
	})
}

// requestReadvertise asks for an advertise pass without waiting for one.
func (r *routeRemote) requestReadvertise() {
	select {
	case r.readvertise <- struct{}{}:
	default: // one is already pending, which covers this request too
	}
}

// drainReadvertise discards a pending request, for a caller about to advertise
// anyway.
func (r *routeRemote) drainReadvertise() {
	select {
	case <-r.readvertise:
	default:
	}
}

// migratePublishedFromV1 moves v1's publish state onto the record rows.
//
// v1 recorded it as keys in its routing datastore; v2 uses records.published,
// which defaults to false. Nothing bridges the two, so without this an upgrade
// silently unpublishes the node: every record still held, none discoverable.
//
// Records are handed over one at a time and each v1 key is dropped once its row
// is flagged. The order matters. Flagging first means a crash leaves the CID in
// both places, which the next run resolves idempotently; dropping first would
// lose it outright. So at every instant, including mid-crash, a published CID is
// present in at least one of the two stores.
//
// Returns how many rows it flagged. Safe to run on every start: once the keys
// are gone there is nothing left to do, and once the operator removes the v1
// files it stops opening anything at all.
func migratePublishedFromV1(ctx context.Context, baseDir string, db types.DatabaseAPI) (int, error) {
	if baseDir == "" || db == nil {
		return 0, nil
	}

	if !hasV1Datastore(baseDir) {
		return 0, nil
	}

	v1, err := datastore.New(datastore.WithFsProvider(baseDir))
	if err != nil {
		return 0, fmt.Errorf("failed to open the v1 routing datastore at %s: %w", baseDir, err)
	}

	defer func() {
		if err := v1.Close(); err != nil {
			migrateLogger.Error("Failed to close the v1 routing datastore", "error", err)
		}
	}()

	results, err := v1.Query(ctx, query.Query{Prefix: v1RecordsPrefix, KeysOnly: true})
	if err != nil {
		return 0, fmt.Errorf("failed to read v1 publish state: %w", err)
	}

	defer results.Close()

	migrated := 0

	for result := range results.Next() {
		if err := ctx.Err(); err != nil {
			return migrated, err //nolint:wrapcheck // context error, nothing to add
		}

		if result.Error != nil {
			migrateLogger.Warn("Skipping unreadable v1 publish entry", "key", result.Key, "error", result.Error)

			continue
		}

		cid := strings.TrimPrefix(result.Key, v1RecordsPrefix)
		if cid == "" {
			continue
		}

		flagged, err := handOverRecord(ctx, v1, db, cid, result.Key)
		if err != nil {
			return migrated, err
		}

		if flagged {
			migrated++
		}
	}

	if migrated > 0 {
		migrateLogger.Info("Migrated v1 publish state onto record rows", "records", migrated)
	}

	return migrated, nil
}

// handOverRecord transfers one record's publish state to v2 and drops v1's copy,
// reporting whether a row was flagged.
func handOverRecord(ctx context.Context, v1 types.Datastore, db types.DatabaseAPI, cid, key string) (bool, error) {
	flagged := true

	if err := db.SetRecordPublished(cid, true); err != nil {
		// The datastore can name a record the database no longer has. There is
		// nothing to hand over, and keeping the key would retry the same
		// failure on every start, so drop it and move on.
		migrateLogger.Warn("Dropping v1 publish entry for an unknown record", "cid", cid, "error", err)

		flagged = false
	}

	if err := v1.Delete(ctx, ds.NewKey(key)); err != nil {
		// Stop rather than continue. The row is already flagged, so the record
		// is safe, but a datastore refusing writes will refuse the next delete
		// too and the log would fill with identical failures.
		return flagged, fmt.Errorf("failed to remove migrated v1 publish entry %s: %w", key, err)
	}

	return flagged, nil
}

// hasV1Datastore reports whether a Badger database exists at the given path.
//
// Opening one read-write creates it, so an unguarded migration would fabricate
// an empty v1 datastore in the volume root of every fresh install.
func hasV1Datastore(baseDir string) bool {
	_, err := os.Stat(filepath.Join(baseDir, badgerManifest))
	if err == nil {
		return true
	}

	if !errors.Is(err, os.ErrNotExist) {
		migrateLogger.Warn("Could not check for a v1 routing datastore", "dir", baseDir, "error", err)
	}

	return false
}
