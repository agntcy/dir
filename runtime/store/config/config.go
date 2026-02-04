// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
	"github.com/agntcy/dir/runtime/store/types"
)

// Config holds storage configuration.
type Config struct {
	// Type is the storage backend type ("etcd" or "crd").
	Type types.StoreType `json:"type" mapstructure:"type"`

	// Etcd configuration.
	Etcd etcd.Config `json:"etcd" mapstructure:"etcd"`

	// CRD configuration.
	CRD crd.Config `json:"crd" mapstructure:"crd"`
}
