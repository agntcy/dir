// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
)

// Config points the server at a directory of file-based OPA policies.
// Named .rego files are loaded by validators with provider "opa" via
// config.file.
type Config struct {
	// Dir is the directory that holds policy files (one file per policy).
	Dir string `json:"dir,omitempty" mapstructure:"dir"`

	// Enforcement decides which policies the records this server returns
	// must comply with, and how each kind of read applies them.
	Enforcement EnforcementConfig `json:"enforcement" mapstructure:"enforcement"`
}

// Mode is how the enforced policies apply to one kind of read.
type Mode string

const (
	// ModeOff applies no policy. The empty mode means the same.
	ModeOff Mode = "off"

	// ModeShadow checks each read and reports what it would exclude, but
	// excludes nothing.
	ModeShadow Mode = "shadow"

	// ModeEnforce excludes every record that does not comply with all the
	// enforced policies: searches leave it out and a read of it by CID is
	// refused.
	ModeEnforce Mode = "enforce"
)

// Active reports whether the mode checks reads at all.
func (m Mode) Active() bool {
	return m == ModeShadow || m == ModeEnforce
}

func (m Mode) validate() error {
	switch m {
	case "", ModeOff, ModeShadow, ModeEnforce:
		return nil
	default:
		return fmt.Errorf("unknown mode %q, expected %q, %q or %q", m, ModeOff, ModeShadow, ModeEnforce)
	}
}

// EnforcementConfig is the enforced policies and one mode per kind of read,
// so enforcement can be rolled out one kind at a time.
type EnforcementConfig struct {
	// Search applies to reads that query the index: search, the catalog,
	// filter values, and publication by query or of every record.
	Search Mode `json:"search,omitempty" mapstructure:"search"`

	// Fetch applies to reads of one record by CID: pull, lookup, referrers,
	// export, the peer RPC, publication of explicit CIDs and verification
	// lookups.
	Fetch Mode `json:"fetch,omitempty" mapstructure:"fetch"`

	// Policies are the policies a record must comply with, each at the
	// version in force. YAML only: a list of objects cannot be bound to a
	// single environment variable.
	Policies []EnforcedPolicy `json:"policies,omitempty" mapstructure:"policies"`
}

// EnforcedPolicy names a policy and the version of it in force. Only a
// verdict under that version counts.
type EnforcedPolicy struct {
	// ID is the policy's identifier, as its evaluator reports it.
	ID string `json:"id" mapstructure:"id"`

	// Version is the policy version in force.
	Version string `json:"version" mapstructure:"version"`
}

// Active reports whether any kind of read checks the enforced policies.
func (c *EnforcementConfig) Active() bool {
	return c.Search.Active() || c.Fetch.Active()
}

func (c *EnforcementConfig) Validate() error {
	if err := c.Search.validate(); err != nil {
		return fmt.Errorf("search: %w", err)
	}

	if err := c.Fetch.validate(); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	seen := make(map[string]bool, len(c.Policies))

	for i, policy := range c.Policies {
		if policy.ID == "" || policy.Version == "" {
			return fmt.Errorf("policy %d: id and version are required", i)
		}

		if seen[policy.ID] {
			return fmt.Errorf("policy %q is listed more than once", policy.ID)
		}

		seen[policy.ID] = true
	}

	if c.Active() && len(c.Policies) == 0 {
		return errors.New("a search or fetch mode is set but no policies are enforced")
	}

	return nil
}
