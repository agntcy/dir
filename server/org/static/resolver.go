// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package static provides a file-based OrgResolver that reads a YAML org chart.
package static

import (
	"context"
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// Config holds configuration for the static org resolver.
type Config struct {
	// File is the path to the YAML org chart file.
	File string `json:"file" mapstructure:"file"`
}

// Resolver is a file-based org resolver. It reads a YAML file mapping each
// manager alias to a list of their direct reports, then recursively expands
// any alias into the full subtree.
//
// Example org-chart.yaml:
//
//	alice@example.com:
//	  - bob@example.com
//	  - carol@example.com
//	bob@example.com:
//	  - dave@example.com
type Resolver struct {
	// reports maps a manager alias to their direct reports.
	reports map[string][]string
}

// New loads the org chart from the given config and returns a Resolver.
func New(cfg Config) (*Resolver, error) {
	data, err := os.ReadFile(cfg.File)
	if err != nil {
		return nil, fmt.Errorf("reading org chart file %q: %w", cfg.File, err)
	}

	var reports map[string][]string
	if err := yaml.Unmarshal(data, &reports); err != nil {
		return nil, fmt.Errorf("parsing org chart file %q: %w", cfg.File, err)
	}

	return &Resolver{reports: reports}, nil
}

// Expand returns the manager alias and all direct and indirect reports under them.
// The returned slice always includes the manager alias itself.
// Cycles in the org chart are silently broken.
func (r *Resolver) Expand(_ context.Context, managerAlias string) ([]string, error) {
	visited := make(map[string]bool)

	return r.expand(managerAlias, visited), nil
}

func (r *Resolver) expand(alias string, visited map[string]bool) []string {
	if visited[alias] {
		return nil
	}

	visited[alias] = true

	result := []string{alias}

	for _, report := range r.reports[alias] {
		result = append(result, r.expand(report, visited)...)
	}

	return result
}
