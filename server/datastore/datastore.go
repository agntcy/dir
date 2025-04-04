// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package datastore

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger"
)

// New is shortcut to creating specific datastore.
// For now, we use memory store.
//
// We should only create a proper datastore from options,
// as we do not implement this interface.
func New(opts ...Option) (types.Datastore, error) {
	// read options
	options := &options{}
	for _, opt := range opts {
		if err := opt(options); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// create local datastore if requested
	if localDir := options.localDir; localDir != "" {
		return badger.NewDatastore(localDir, &badger.DefaultOptions) //nolint:wrapcheck
	}

	// create in-memory datastore
	return datastore.NewMapDatastore(), nil
}
