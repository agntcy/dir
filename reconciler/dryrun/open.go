// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dryrun

import (
	"context"
	"errors"
	"fmt"

	"github.com/agntcy/dir/reconciler/tasks/policy"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	"github.com/agntcy/dir/server/store/oci"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
)

// ErrStoreUnreachable is returned by Open when the store does not answer. A
// record that cannot be read is not a record the policy rejects, so a dry run
// does not start without a store to read from.
var ErrStoreUnreachable = errors.New("the store is not reachable")

// Open opens the database and the store a dry run reads, as the reconciler
// opens them, and returns them with a function that closes what needs closing.
// The store is the node's registry, so the node has to be running. Opening a
// database applies the migrations it is missing, as starting the node does, so
// run a dry run with the same version the node runs.
func Open(ctx context.Context, db dbconfig.Config, store ociconfig.Config, limits policy.Config) (Source, func() error, error) {
	nodeDB, err := database.New(db)
	if err != nil {
		return Source{}, nil, fmt.Errorf("open the database: %w", err)
	}

	records, err := oci.New(store)
	if err != nil {
		_ = nodeDB.Close()

		return Source{}, nil, fmt.Errorf("open the store: %w", err)
	}

	if !records.IsReady(ctx) {
		_ = nodeDB.Close()

		address, _ := store.GetRegistryAddress()

		return Source{}, nil, fmt.Errorf("%w: nothing answers at %s; is the node running?", ErrStoreUnreachable, address)
	}

	return Source{DB: nodeDB, Records: records, Limits: limits}, nodeDB.Close, nil
}
