// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import "time"

// DefaultWellKnownTimeout is the default timeout for .well-known
// HTTP requests.
const DefaultWellKnownTimeout = 10 * time.Second

// WellKnown holds configuration for the .well-known/dir-naming HTTP
// verification provider used by the naming subsystem.
type WellKnown struct {
	// Timeout is the maximum time to wait for HTTP requests.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`
}

// DefaultWellKnown returns the default well-known configuration.
func DefaultWellKnown() *WellKnown {
	return &WellKnown{Timeout: DefaultWellKnownTimeout}
}
