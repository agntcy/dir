// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config declares the two kinds of content policy, because there are
// two kinds of thing.
//
// An evaluation policy produces information: the reconciler runs it over every
// record and stores a verdict. It has no operation and no mode, because it
// never acts; its scheduling comes from the task that runs it.
//
// An operation policy decides on information: it answers one question at one
// operation and persists nothing. It has an operation and a mode, and no
// scheduling.
//
// They were previously one list distinguished by an op whose values were three
// lifecycle points and "evaluate", which is not one. Configuring them
// separately is what lets each carry only the fields that apply to it.
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	// ProviderOASF is the OASF schema validator. It cannot define a policy:
	// its content lives behind an HTTP endpoint, so there is nothing to hash
	// into a version.
	ProviderOASF = "oasf"

	// ProviderCEL evaluates inline CEL expressions.
	ProviderCEL = "cel"

	// ProviderOPA evaluates a Rego policy file.
	ProviderOPA = "opa"

	// OpPush runs on StoreService.Push (including dirctl push/import).
	OpPush = "push"

	// OpAutosync runs on DHT autosync ingest.
	OpAutosync = "autosync"

	// OpIndex runs on the reconciler indexer.
	OpIndex = "index"
)

// policyNamePattern is what a policy name may contain. The name ends up in the
// policy's ID, which is listed in a comma-separated setting and used as a
// metric label.
var policyNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

var knownOps = map[string]struct{}{
	OpPush:     {},
	OpAutosync: {},
	OpIndex:    {},
}

// KnownOps returns every op a policy may be configured at, in no order.
//
// Exported so that what the rest of the server derives per op can be checked
// against this list rather than restating it.
func KnownOps() []string {
	ops := make([]string, 0, len(knownOps))
	for op := range knownOps {
		ops = append(ops, op)
	}

	return ops
}

// Mode is how a policy's decision applies.
type Mode string

const (
	// ModeOff applies no policy. For a read gate the empty mode means the
	// same; for an operation policy it means enforce, because a policy
	// configured at an operation was always acted on.
	ModeOff Mode = "off"

	// ModeShadow checks and reports what it would have done, but does
	// nothing.
	ModeShadow Mode = "shadow"

	// ModeEnforce acts on the decision: a read excludes the record, and an
	// operation on it is refused.
	ModeEnforce Mode = "enforce"
)

// Active reports whether the mode checks anything at all.
func (m Mode) Active() bool {
	return m == ModeShadow || m == ModeEnforce
}

// Validate checks that the mode is one of the known values.
func (m Mode) Validate() error {
	switch m {
	case "", ModeOff, ModeShadow, ModeEnforce:
		return nil
	default:
		return fmt.Errorf("unknown mode %q, expected %q, %q or %q", m, ModeOff, ModeShadow, ModeEnforce)
	}
}

// Evaluation is one evaluation policy: a producer of verdicts.
//
// Only the providers whose content is known when they are loaded can define
// one, because a verdict is stored against the version of the policy that
// reached it. That rules out oasf, which is why there is no schema_url here.
type Evaluation struct {
	// Provider selects the backend: opa or cel.
	Provider string `json:"provider" mapstructure:"provider"`

	// Name is what this policy is called. With the provider it forms the
	// policy ID that verdicts are stored under and that enforcement lists.
	Name string `json:"name" mapstructure:"name"`

	// File is the .rego file in policy.dir, for provider opa.
	File string `json:"file,omitempty" mapstructure:"file"`

	// Expressions are the CEL expressions, for provider cel. Every one of
	// them must hold.
	Expressions []string `json:"expressions,omitempty" mapstructure:"expressions"`
}

// Operation is one operation policy: a decider at one or more operations.
type Operation struct {
	// Provider selects the backend: oasf, opa or cel.
	Provider string `json:"provider" mapstructure:"provider"`

	// Ops is the set of operations this policy decides at.
	// Known values: push, autosync, index.
	Ops []string `json:"op,omitempty" mapstructure:"op"`

	// Mode is how the decision applies. An empty mode means enforce, which
	// is what every operation policy did before there was a mode.
	//
	// off does not run the policy. shadow runs it and reports the decision
	// without refusing the record. enforce refuses it.
	Mode Mode `json:"mode,omitempty" mapstructure:"mode"`

	// Name is what this policy is called, and what a decision about it is
	// reported under.
	Name string `json:"name" mapstructure:"name"`

	// SchemaURL is the schema endpoint, for provider oasf.
	SchemaURL string `json:"schema_url,omitempty" mapstructure:"schema_url"`

	// File is the .rego file in policy.dir, for provider opa.
	File string `json:"file,omitempty" mapstructure:"file"`

	// Expressions are the CEL expressions, for provider cel. Every one of
	// them must hold.
	Expressions []string `json:"expressions,omitempty" mapstructure:"expressions"`
}

// Evaluations is the ordered list of evaluation policies.
type Evaluations []Evaluation

// Operations is the ordered list of operation policies.
type Operations []Operation

// PolicyName returns what this policy is called: Name, or for opa the policy
// file's name without .rego. It is empty when there is neither.
func (e Evaluation) PolicyName() string {
	return policyName(e.Provider, e.Name, e.File)
}

// PolicyName returns what this policy is called: Name, or for opa the policy
// file's name without .rego. It is empty when there is neither.
func (o Operation) PolicyName() string {
	return policyName(o.Provider, o.Name, o.File)
}

// PolicyID returns the ID of this policy, such as "cel:has-provenance". It is
// what verdicts are stored under and what enforcement lists, and it stays the
// same when the policy's content changes.
func (e Evaluation) PolicyID() string {
	return e.Provider + ":" + e.PolicyName()
}

// PolicyID returns the ID of this policy, such as "cel:require-description".
func (o Operation) PolicyID() string {
	return o.Provider + ":" + o.PolicyName()
}

func policyName(provider, name, file string) string {
	if name = strings.TrimSpace(name); name != "" {
		return name
	}

	if provider == ProviderOPA {
		return strings.TrimSuffix(strings.TrimSpace(file), ".rego")
	}

	return ""
}

// HasOp reports whether this policy is configured for op.
func (o Operation) HasOp(op string) bool {
	return slices.Contains(o.Ops, op)
}

// Validate checks each entry and that no two define the same policy.
// An empty list is valid.
func (c Evaluations) Validate() error {
	seen := make(map[string]int, len(c))

	for i, e := range c {
		if err := e.validate(i); err != nil {
			return err
		}

		id := e.PolicyID()
		if first, dup := seen[id]; dup {
			return fmt.Errorf("evaluations[%d]: policy %q is already defined by evaluations[%d]", i, id, first)
		}

		seen[id] = i
	}

	return nil
}

// Validate checks each entry and that no two define the same policy.
// An empty list is valid.
func (c Operations) Validate() error {
	seen := make(map[string]int, len(c))

	for i, o := range c {
		if err := o.validate(i); err != nil {
			return err
		}

		id := o.PolicyID()
		if first, dup := seen[id]; dup {
			return fmt.Errorf("operations[%d]: policy %q is already defined by operations[%d]", i, id, first)
		}

		seen[id] = i
	}

	return nil
}

// ValidateNames rejects a policy ID used in both blocks.
//
// Nothing would resolve it wrongly, because only evaluation IDs are ever
// referenced. The reason is that a log line naming cel:something would be
// ambiguous to whoever reads it, and those logs are the audit surface.
func ValidateNames(evaluations Evaluations, operations Operations) error {
	inEvaluations := make(map[string]int, len(evaluations))
	for i, e := range evaluations {
		inEvaluations[e.PolicyID()] = i
	}

	for i, o := range operations {
		if first, both := inEvaluations[o.PolicyID()]; both {
			return fmt.Errorf("operations[%d]: policy %q is also defined by evaluations[%d]: "+
				"a name must identify one policy", i, o.PolicyID(), first)
		}
	}

	return nil
}

func (e Evaluation) validate(index int) error {
	where := fmt.Sprintf("evaluations[%d]", index)

	if e.Provider == ProviderOASF {
		return fmt.Errorf("%s: provider %q cannot define an evaluation policy, because its content "+
			"is not known when it is loaded: only %q and %q can", where, ProviderOASF, ProviderOPA, ProviderCEL)
	}

	// Before the name, because for opa the name comes from the file.
	if err := validateProviderFields(where, e.Provider, providerFields{
		file:        e.File,
		expressions: e.Expressions,
	}); err != nil {
		return err
	}

	return errorAt(where, validateName(e.PolicyName()))
}

func (o Operation) validate(index int) error {
	where := fmt.Sprintf("operations[%d]", index)

	// Before the name, because for opa the name comes from the file.
	if err := validateProviderFields(where, o.Provider, providerFields{
		schemaURL:   o.SchemaURL,
		file:        o.File,
		expressions: o.Expressions,
	}); err != nil {
		return err
	}

	if err := validateName(o.PolicyName()); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}

	if err := o.Mode.Validate(); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}

	if len(o.Ops) == 0 {
		return fmt.Errorf("%s: at least one op is required", where)
	}

	seen := make(map[string]bool, len(o.Ops))

	for _, op := range o.Ops {
		if _, ok := knownOps[op]; !ok {
			return fmt.Errorf("%s: unsupported op %q, expected %q, %q or %q",
				where, op, OpPush, OpAutosync, OpIndex)
		}

		if seen[op] {
			return fmt.Errorf("%s: op %q is listed more than once", where, op)
		}

		seen[op] = true
	}

	return nil
}

func errorAt(where string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("%s: %w", where, err)
}

func validateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}

	if !policyNamePattern.MatchString(name) {
		return fmt.Errorf("policy name %q must start with a letter or digit and contain only "+
			"letters, digits, '.', '_' and '-'", name)
	}

	return nil
}

// providerFields are the settings a provider may carry. Exactly the ones its
// provider uses must be set, so that a setting meant for another provider is
// rejected rather than silently ignored.
type providerFields struct {
	schemaURL   string
	file        string
	expressions []string
}

func validateProviderFields(where, provider string, f providerFields) error {
	var wanted string

	switch provider {
	case "":
		return fmt.Errorf("%s: provider is required", where)
	case ProviderOASF:
		wanted = "schema_url"

		if strings.TrimSpace(f.schemaURL) == "" {
			return fmt.Errorf("%s: schema_url is required for provider %q", where, provider)
		}
	case ProviderOPA:
		wanted = "file"

		if err := ValidatePolicyFile(f.file); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	case ProviderCEL:
		wanted = "expressions"

		if err := validateExpressions(f.expressions); err != nil {
			return fmt.Errorf("%s: %w", where, err)
		}
	default:
		return fmt.Errorf("%s: unsupported provider %q, expected %q, %q or %q",
			where, provider, ProviderOASF, ProviderOPA, ProviderCEL)
	}

	// Ordered, so that an entry setting two fields that do not apply always
	// reports the same one.
	for _, other := range []struct {
		field string
		set   bool
	}{
		{"schema_url", strings.TrimSpace(f.schemaURL) != ""},
		{"file", strings.TrimSpace(f.file) != ""},
		{"expressions", len(f.expressions) > 0},
	} {
		field, set := other.field, other.set
		if set && field != wanted {
			return fmt.Errorf("%s: %s does not apply to provider %q, which takes %s",
				where, field, provider, wanted)
		}
	}

	return nil
}

func validateExpressions(exprs []string) error {
	if len(exprs) == 0 {
		return fmt.Errorf("expressions is required for provider %q", ProviderCEL)
	}

	for i, expr := range exprs {
		if strings.TrimSpace(expr) == "" {
			return fmt.Errorf("expressions[%d]: value is empty", i)
		}
	}

	return nil
}

// ValidatePolicyFile checks that name is a bare .rego filename, not a path.
func ValidatePolicyFile(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("file is required for provider %q", ProviderOPA)
	}

	if name != filepath.Base(name) || name == "." || name == ".." {
		return fmt.Errorf("file %q must be a filename, not a path", name)
	}

	if !strings.HasSuffix(name, ".rego") {
		return fmt.Errorf("file %q must have a .rego extension", name)
	}

	return nil
}
