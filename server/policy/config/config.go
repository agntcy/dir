// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"time"

	validatorsconfig "github.com/agntcy/dir/server/validators/config"
)

// DefaultRefreshInterval is how often the server rechecks which version of
// each enforced policy is in force.
const DefaultRefreshInterval = 30 * time.Second

// Config is the node's record checks: the validators that run on ingest and
// as content policies, the directory named OPA files are loaded from, and
// how reads enforce those policies.
type Config struct {
	// Dir is the directory that holds policy files (one file per policy).
	// An opa validator's config.file is resolved against it.
	Dir string `json:"dir,omitempty" mapstructure:"dir"`

	// Validators is the ordered list of record validators (OASF, OPA, CEL).
	// YAML only — a list of objects cannot be bound to a single env var.
	Validators validatorsconfig.Config `json:"validators,omitempty" mapstructure:"validators"`

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

	// Policies are the IDs of the policies a record must comply with, as
	// their evaluators report them, e.g. "opa:require-license".
	//
	// No version is given. A policy's version comes from its content: the
	// reconciler registers the version of each policy it runs, and the server
	// follows it. A verdict reached under any other version does not count, so
	// after a policy changes a record stays hidden until it is evaluated under
	// the new rule, and nothing the new rule rejects is served meanwhile.
	Policies []string `json:"policies,omitempty" mapstructure:"policies"`

	// RefreshInterval is how often the server rechecks which version of each
	// policy is in force. Zero means DefaultRefreshInterval.
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`
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

	for i, id := range c.Policies {
		if id == "" {
			return fmt.Errorf("policy %d: the id is required", i)
		}

		if seen[id] {
			return fmt.Errorf("policy %q is listed more than once", id)
		}

		seen[id] = true
	}

	if c.Active() && len(c.Policies) == 0 {
		return errors.New("a search or fetch mode is set but no policies are enforced")
	}

	if c.RefreshInterval < 0 {
		return errors.New("refresh_interval must not be negative")
	}

	return nil
}

// GetRefreshInterval returns RefreshInterval, or DefaultRefreshInterval when
// it is not set.
func (c *EnforcementConfig) GetRefreshInterval() time.Duration {
	if c.RefreshInterval == 0 {
		return DefaultRefreshInterval
	}

	return c.RefreshInterval
}
