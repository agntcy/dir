// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
)

// versionLength is how many hex digits of the content hash make a policy
// version: enough that two different policies do not share one, short enough to
// read in a metric label.
const versionLength = 16

// Registry maps operations to the operation policies that decide at them, and
// holds the evaluation policies.
type Registry struct {
	byOp        map[string][]Validator
	evaluations []evaluation
}

// evaluation is one loaded evaluation policy. Evaluators turns it into an
// Evaluator once it has a source of record content.
type evaluation struct {
	id        string
	version   string
	validator Validator
}

// contentVersion is the version of a policy made of parts, in order. Each part
// is length-prefixed so that moving a boundary between two parts changes it.
func contentVersion(parts ...string) string {
	h := sha256.New()

	for _, part := range parts {
		fmt.Fprintf(h, "%d:%s", len(part), part)
	}

	return hex.EncodeToString(h.Sum(nil))[:versionLength]
}

// NewRegistry constructs the policies of both blocks. policyDir is the
// directory named .rego files are loaded from (server/reconciler policy.dir).
// Two empty lists yield a registry with no evaluation policies and no
// operation policy for any op.
func NewRegistry(
	ctx context.Context,
	evaluations policyconfig.Evaluations,
	operations policyconfig.Operations,
	policyDir string,
) (*Registry, error) {
	if err := evaluations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := operations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := policyconfig.ValidateNames(evaluations, operations); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	var loaded []evaluation

	for i, e := range evaluations {
		if e.Provider == policyconfig.ProviderOASF {
			return nil, fmt.Errorf("evaluations[%d]: provider %q cannot define an evaluation policy", i, e.Provider)
		}

		v, version, err := newValidator(ctx, layerEvaluation, e.Provider,
			provider{file: e.File, expressions: e.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("evaluations[%d]: %w", i, err)
		}

		loaded = append(loaded, evaluation{id: e.PolicyID(), version: version, validator: v})
	}

	byOp := make(map[string][]Validator)

	for i, o := range operations {
		v, _, err := newValidator(ctx, layerOperation, o.Provider,
			provider{schemaURL: o.SchemaURL, file: o.File, expressions: o.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		for _, op := range o.Ops {
			byOp[op] = append(byOp[op], v)
		}
	}

	return &Registry{byOp: byOp, evaluations: loaded}, nil
}

// provider is the settings of whichever backend is configured. Config
// validation has already rejected the ones that do not apply.
type provider struct {
	schemaURL   string
	file        string
	expressions []string
}

// newValidator builds the validator for one configured provider, and the version
// of its content when the provider has any. OASF has none.
//
// l is the layer the policy was configured in. Only CEL needs it today,
// because it is the only provider that compiles its policy here and so the
// only one that can reject reading the other layer's inputs at load time.
func newValidator(ctx context.Context, l layer, name string, p provider, policyDir string) (Validator, string, error) {
	switch name {
	case policyconfig.ProviderOASF:
		v, err := newOASFValidator(p.schemaURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		return v, "", nil
	case policyconfig.ProviderCEL:
		v, err := newCELValidator(p.expressions, l)
		if err != nil {
			return nil, "", fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return v, v.version, nil
	case policyconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, p.file)
		if err != nil {
			return nil, "", fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return v, v.version, nil
	default:
		return nil, "", fmt.Errorf("unsupported provider %q", name)
	}
}

// For returns the operation policies configured for op, in config order.
// The slice is empty if none are configured.
func (r *Registry) For(op string) []Validator {
	if r == nil {
		return nil
	}

	return r.byOp[op]
}

// EvaluationIDs returns the evaluation policy IDs, in config order.
func (r *Registry) EvaluationIDs() []string {
	if r == nil {
		return nil
	}

	ids := make([]string, len(r.evaluations))
	for i, e := range r.evaluations {
		ids[i] = e.id
	}

	return ids
}

// Evaluators returns one evaluator per evaluation policy, in config order.
// records supplies the record content a verdict is decided on. A nil registry
// returns no evaluators.
func (r *Registry) Evaluators(records RecordSource) []Evaluator {
	if r == nil {
		return nil
	}

	out := make([]Evaluator, len(r.evaluations))
	for i, e := range r.evaluations {
		out[i] = &evaluator{
			id:        e.id,
			version:   e.version,
			validator: e.validator,
			records:   records,
		}
	}

	return out
}

// Decide applies every operation policy configured for op, in config order.
// A nil registry or an op with no policies is a no-op and reports that the
// record passes. Messages from every failing policy are collected.
func (r *Registry) Decide(ctx context.Context, op string, record *corev1.Record) (bool, []string, error) {
	var all []string

	input := operationInput(op, record)

	for _, v := range r.For(op) {
		if v == nil {
			continue
		}

		ok, msgs, err := validateOperation(ctx, record, input, v)
		if err != nil {
			return false, nil, fmt.Errorf("decide record: %w", err)
		}

		if !ok {
			all = append(all, msgs...)
		}
	}

	return len(all) == 0, all, nil
}

// With returns a registry that runs vs for op. Intended for tests.
func With(op string, vs ...Validator) *Registry {
	return &Registry{byOp: map[string][]Validator{op: vs}}
}
