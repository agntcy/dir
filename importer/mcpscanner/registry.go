// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcpscanner

import (
	"fmt"
	"sort"
	"strings"

	mcpscannerconfig "github.com/agntcy/dir/importer/mcpscanner/config"
)

// RunnerFactory creates a Runner from scanner config.
type RunnerFactory func(cfg mcpscannerconfig.Config) Runner

// registry maps scan mode names to their factory functions.
// To add a new scan mode, implement the Runner interface and add one entry here.
var registry = map[string]RunnerFactory{
	"behavioral": func(cfg mcpscannerconfig.Config) Runner { return NewBehavioralRunner(cfg) },
}

// NewRunners creates Runner instances for each mode specified in the config.
func NewRunners(cfg mcpscannerconfig.Config) ([]Runner, error) {
	var runners []Runner

	for _, mode := range cfg.Modes {
		factory, ok := registry[mode]
		if !ok {
			return nil, fmt.Errorf("unknown scanner mode %q (available: %s)", mode, AvailableModes())
		}

		runners = append(runners, factory(cfg))
	}

	return runners, nil
}

// AvailableModes returns a sorted, comma-separated list of registered scan mode names.
func AvailableModes() string {
	modes := make([]string, 0, len(registry))
	for name := range registry {
		modes = append(modes, name)
	}

	sort.Strings(modes)

	return strings.Join(modes, ", ")
}
