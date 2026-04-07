// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/runtime/store/config"
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
	"github.com/agntcy/dir/runtime/store/sql"
	"github.com/agntcy/dir/runtime/store/types"
)

// New creates a new store based on configuration.
//
//nolint:wrapcheck
func New(cfg config.Config) (types.Store, error) {
	switch cfg.Type {
	case etcd.StoreType:
		return etcd.New(cfg.Etcd)
	case crd.StoreType:
		return crd.New(cfg.CRD)
	case sql.StoreTypeSqlite:
		db, err := sql.NewSqlite()
		if err != nil {
			return nil, fmt.Errorf("failed to create sqlite connection: %w", err)
		}

		store, err := sql.New(db)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize sqlite store: %w", err)
		}

		return store, nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}
