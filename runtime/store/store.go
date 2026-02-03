// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"

	"github.com/agntcy/dir/runtime/store/config"
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
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
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}
