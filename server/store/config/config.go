// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	oci "github.com/agntcy/dir/server/store/oci/config"
)

const (
	DefaultProvider = "oci"
)

type Config struct {
	// Provider is the type of the storage provider.
	Provider string `json:"c,omitempty" mapstructure:"provider"`

	// Config for OCI database.
	OCI oci.Config `json:"oci,omitempty" mapstructure:"oci"`
}
