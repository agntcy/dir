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

	// ProviderCEL is the CEL expression validator.
	ProviderCEL = "cel"

	// OpPush runs on StoreService.Push (including dirctl push/import).
	OpPush = "push"

	// OpAutosync runs on DHT autosync ingest.
	OpAutosync = "autosync"

	// OpIndex runs on the reconciler indexer.
	OpIndex = "index"

	// ConfigKeySchemaURL is the OASF config key for the schema endpoint.
	ConfigKeySchemaURL = "schema_url"

	// ConfigKeyExpressions is the CEL config key for the inline expressions.
	ConfigKeyExpressions = "expressions"
)

// Config is the ordered list of record validators.
type Config []Validator

// Validator is one record-validation backend and the operations it applies to.
type Validator struct {
	// Provider selects the backend. Supported values: "oasf", "cel".
	Provider string `json:"provider" mapstructure:"provider"`

	// Ops is the set of operations this validator runs on.
	// Known values: push, autosync, index.
	Ops []string `json:"op,omitempty" mapstructure:"op"`

	// Config is provider-specific. Keys vary by provider:
	//   oasf: schema_url (required)
	//   cel:  expressions (required list of CEL expressions; all must be true)
	Config map[string]any `json:"config,omitempty" mapstructure:"config"`
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

	if err := v.validateProviderConfig(index); err != nil {
		return err
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

func (v Validator) validateProviderConfig(index int) error {
	switch v.Provider {
	case ProviderOASF:
		if strings.TrimSpace(v.ConfigString(ConfigKeySchemaURL)) == "" {
			return fmt.Errorf("validators[%d]: config.schema_url is required for provider %q", index, v.Provider)
		}
	case ProviderCEL:
		if _, err := v.ConfigStrings(ConfigKeyExpressions); err != nil {
			return fmt.Errorf("validators[%d]: config.expressions: %w", index, err)
		}
	default:
		return fmt.Errorf("validators[%d]: unsupported provider %q", index, v.Provider)
	}

	return nil
}

// HasOp reports whether this validator is configured for op.
func (v Validator) HasOp(op string) bool {
	return slices.Contains(v.Ops, op)
}

// ConfigString returns the string value of a config key, or "" if missing or not a string.
func (v Validator) ConfigString(key string) string {
	if v.Config == nil {
		return ""
	}

	s, ok := v.Config[key].(string)
	if !ok {
		return ""
	}

	return s
}

// ConfigStrings returns the string values of a list config key.
// It accepts []string or []any of strings (as YAML unmarshals). Empty items
// and a missing or empty list are errors.
func (v Validator) ConfigStrings(key string) ([]string, error) {
	if v.Config == nil {
		return nil, fmt.Errorf("%s is required", key)
	}

	raw, ok := v.Config[key]
	if !ok || raw == nil {
		return nil, fmt.Errorf("%s is required", key)
	}

	items, err := asStringSlice(raw)
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("%s must contain at least one value", key)
	}

	return items, nil
}

func asStringSlice(raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for i, s := range v {
			s = strings.TrimSpace(s)
			if s == "" {
				return nil, fmt.Errorf("[%d]: value is empty", i)
			}

			out = append(out, s)
		}

		return out, nil
	case []any:
		out := make([]string, 0, len(v))
		for i, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("[%d]: must be a string", i)
			}

			s = strings.TrimSpace(s)
			if s == "" {
				return nil, fmt.Errorf("[%d]: value is empty", i)
			}

			out = append(out, s)
		}

		return out, nil
	default:
		return nil, fmt.Errorf("must be a list of strings")
	}
}
