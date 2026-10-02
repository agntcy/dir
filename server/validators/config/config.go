// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	// ProviderOASF is the OASF schema validator.
	ProviderOASF = "oasf"

	// ProviderCEL is the CEL expression validator.
	ProviderCEL = "cel"

	// ProviderOPA is the file-based OPA/Rego validator.
	ProviderOPA = "opa"

	// OpPush runs on StoreService.Push (including dirctl push/import).
	OpPush = "push"

	// OpAutosync runs on DHT autosync ingest.
	OpAutosync = "autosync"

	// OpIndex runs on the reconciler indexer.
	OpIndex = "index"

	// OpEvaluate makes the validator a content policy: the reconciler's policy
	// evaluation task runs it on every record and stores a verdict, which the
	// server's read gate enforces. Only the opa and cel providers accept it.
	OpEvaluate = "evaluate"

	// ConfigKeySchemaURL is the OASF config key for the schema endpoint.
	ConfigKeySchemaURL = "schema_url"

	// ConfigKeyFile is the OPA config key for the policy filename.
	ConfigKeyFile = "file"

	// ConfigKeyExpressions is the CEL config key for the inline expressions.
	ConfigKeyExpressions = "expressions"

	// ConfigKeyName names the policy a validator with op evaluate defines. It
	// is optional for OPA, where it defaults to the file name without .rego,
	// and required for CEL.
	ConfigKeyName = "name"
)

// policyNamePattern is what a policy name may contain. The name ends up in the
// policy's ID, which is listed in a comma-separated setting and used as a
// metric label.
var policyNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Config is the ordered list of record validators.
type Config []Validator

// Validator is one record-validation backend and the operations it applies to.
type Validator struct {
	// Provider selects the backend. Supported values: oasf, opa, cel.
	Provider string `json:"provider" mapstructure:"provider"`

	// Ops is the set of operations this validator runs on.
	// Known values: push, autosync, index, evaluate.
	Ops []string `json:"op,omitempty" mapstructure:"op"`

	// Config is provider-specific.
	Config map[string]any `json:"config,omitempty" mapstructure:"config"`
}

var knownOps = map[string]struct{}{
	OpPush:     {},
	OpAutosync: {},
	OpIndex:    {},
	OpEvaluate: {},
}

// Validate checks each entry, and that no two entries define the same policy.
// An empty list is valid (no record validation).
func (c Config) Validate() error {
	policies := make(map[string]int)

	for i, v := range c {
		if err := v.validate(i); err != nil {
			return err
		}

		if !v.HasOp(OpEvaluate) {
			continue
		}

		id := v.PolicyID()
		if first, dup := policies[id]; dup {
			return fmt.Errorf("validators[%d]: policy %q is already defined by validators[%d]", i, id, first)
		}

		policies[id] = i
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

	if v.HasOp(OpEvaluate) {
		return v.validatePolicy(index)
	}

	return nil
}

// validatePolicy checks what a validator needs to define a content policy: a
// provider whose content is known when it is loaded, and a name the policy can
// be listed by.
func (v Validator) validatePolicy(index int) error {
	if v.Provider != ProviderOPA && v.Provider != ProviderCEL {
		return fmt.Errorf("validators[%d]: provider %q cannot define a policy: op %q supports only %q and %q",
			index, v.Provider, OpEvaluate, ProviderOPA, ProviderCEL)
	}

	name := v.PolicyName()
	if name == "" {
		return fmt.Errorf("validators[%d]: config.name is required for provider %q with op %q", index, v.Provider, OpEvaluate)
	}

	if !policyNamePattern.MatchString(name) {
		return fmt.Errorf("validators[%d]: policy name %q must start with a letter or digit and contain only letters, digits, '.', '_' and '-'", index, name)
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
	case ProviderOPA:
		if err := ValidatePolicyFile(v.ConfigString(ConfigKeyFile)); err != nil {
			return fmt.Errorf("validators[%d]: %w", index, err)
		}
	default:
		return fmt.Errorf("validators[%d]: unsupported provider %q", index, v.Provider)
	}

	return nil
}

// PolicyName returns the name of the policy this validator defines: config.name,
// or for OPA the policy file's name without .rego. It is empty when there is
// none.
func (v Validator) PolicyName() string {
	if name := strings.TrimSpace(v.ConfigString(ConfigKeyName)); name != "" {
		return name
	}

	if v.Provider == ProviderOPA {
		return strings.TrimSuffix(strings.TrimSpace(v.ConfigString(ConfigKeyFile)), ".rego")
	}

	return ""
}

// PolicyID returns the ID of the policy this validator defines, such as
// "opa:require-license". It is what the server's enforcement setting lists,
// and stays the same when the policy's content changes.
func (v Validator) PolicyID() string {
	return v.Provider + ":" + v.PolicyName()
}

// HasOp reports whether this validator is configured for op.
func (v Validator) HasOp(op string) bool {
	return slices.Contains(v.Ops, op)
}

// ValidatePolicyFile checks that name is a bare .rego filename, not a path.
func ValidatePolicyFile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("config.file is required for provider %q", ProviderOPA)
	}

	if name != filepath.Base(name) || name == "." || name == ".." {
		return fmt.Errorf("config.file %q must be a filename, not a path", name)
	}

	if !strings.HasSuffix(name, ".rego") {
		return fmt.Errorf("config.file %q must have a .rego extension", name)
	}

	return nil
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
