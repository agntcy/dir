// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"errors"
	"strings"
)

// ProcessSelector defines a function type for selecting processes based on custom criteria.
type ProcessSelector func(process *Process) (bool, error)

// ProcessSelectorNonSystem is the default process selector that selects non-system processes.
var ProcessSelectorNonSystem ProcessSelector = func(process *Process) (bool, error) {
	// Exclude system processes by checking if the username is "root" or "SYSTEM".
	if process.Username == "root" || process.Username == "SYSTEM" {
		return false, nil
	}

	return true, nil
}

// ProcessSelectorDiscoveryFunc is the active process selector used during discovery.
//
// TODO: Add more selectors for agentic runtimes (e.g., based on process name, annotations, etc.)
// and allow users to configure this.
var ProcessSelectorDiscoveryFunc = func(cfg Config) ProcessSelector {
	return func(process *Process) (bool, error) {
		if cfg.EnvSelector == "" {
			return true, nil
		}

		if strings.Contains(process.Cmdline, cfg.EnvSelector) {
			return true, nil
		}

		return false, nil
	}
}

// Apply applies a sanity checks and readability of ProcessSelector when performing checks.
func (ps ProcessSelector) Apply(process *Process) (bool, error) {
	if ps == nil {
		return false, errors.New("process selector is nil")
	}

	return ps(process)
}
