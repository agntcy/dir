// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"slices"
	"strings"
)

const (
	// ProviderOASF is the OASF schema validator.
	ProviderOASF = "oasf"

	// OpPush runs on StoreService.Push (including dirctl push/import).
	OpPush = "push"

	// OpAutosync runs on DHT autosync ingest.
	OpAutosync = "autosync"

	// OpIndex runs on the reconciler indexer.
	OpIndex = "index"
)

// Config is the ordered list of record validators.
type Config []Validator

// Validator is one record-validation backend and the operations it applies to.
type Validator struct {
	// Provider selects the backend. Currently only "oasf" is supported.
	Provider string `json:"provider" mapstructure:"provider"`

	// SchemaURL is the OASF schema endpoint (required for provider "oasf").
	SchemaURL string `json:"schema_url,omitempty" mapstructure:"schema_url"`

	// Ops is the set of operations this validator runs on.
	// Known values: push, autosync, index.
	Ops []string `json:"op,omitempty" mapstructure:"op"`
}

var knownOps = map[string]struct{}{
	OpPush:     {},
	OpAutosync: {},
	OpIndex:    {},
}

// Validate checks each entry. An empty list is valid (no record validation).
func (c Config) Validate() error {
	for i, v := range c {
		if err := v.validate(i); err != nil {
			return err
		}
	}

	return nil
}

func (v Validator) validate(index int) error {
	if v.Provider == "" {
		return fmt.Errorf("validators[%d]: provider is required", index)
	}

	if v.Provider != ProviderOASF {
		return fmt.Errorf("validators[%d]: unsupported provider %q", index, v.Provider)
	}

	if strings.TrimSpace(v.SchemaURL) == "" {
		return fmt.Errorf("validators[%d]: schema_url is required for provider %q", index, v.Provider)
	}

	if len(v.Ops) == 0 {
		return fmt.Errorf("validators[%d]: at least one op is required", index)
	}

	for _, op := range v.Ops {
		if _, ok := knownOps[op]; !ok {
			return fmt.Errorf("validators[%d]: unsupported op %q", index, op)
		}
	}

	return nil
}

// HasOp reports whether this validator is configured for op.
func (v Validator) HasOp(op string) bool {
	return slices.Contains(v.Ops, op)
}
