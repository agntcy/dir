// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/agntcy/dir/runtime/discovery/resolver/a2a"
	"github.com/agntcy/dir/runtime/discovery/resolver/oasf"
)

// Config holds all resolver-specific configuration.
type Config struct {
	// A2A resolver configuration.
	A2A a2a.Config `json:"a2a" mapstructure:"a2a"`

	// OASF resolver configuration.
	OASF oasf.Config `json:"oasf" mapstructure:"oasf"`
}
