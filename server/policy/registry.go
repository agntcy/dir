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
	byOp        map[string][]operationPolicy
	evaluations []evaluation
}

// operationPolicy is one loaded operation policy: what decides, what it is
// called, and how its decision applies.
//
// The mode is kept here because it is a property of the configured policy, not
// of the decision: the same validator in two entries can enforce at one
// operation and only report at another.
type operationPolicy struct {
	id        string
	mode      policyconfig.Mode
	validator Validator
}

// evaluation is one loaded evaluation policy. Evaluators turns it into an
// Evaluator once it has a source of record content.
type evaluation struct {
	id         string
	version    string
	validator  Validator
	factGroups []string
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

		v, version, groups, err := newValidator(ctx, layerEvaluation, e.Provider,
			provider{file: e.File, expressions: e.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("evaluations[%d]: %w", i, err)
		}

		loaded = append(loaded, evaluation{
			id:         e.PolicyID(),
			version:    version,
			validator:  v,
			factGroups: groups,
		})
	}

	byOp := make(map[string][]operationPolicy)

	for i, o := range operations {
		// An operation policy stores no verdict, so nothing caches what it
		// decided and nothing has to be invalidated: its dependencies are not
		// asked for.
		v, _, _, err := newValidator(ctx, layerOperation, o.Provider,
			provider{schemaURL: o.SchemaURL, file: o.File, expressions: o.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		// Resolved once, here, so that a decision and the log of it report the
		// mode the policy runs in rather than the mode the operator omitted.
		mode := o.Mode
		if mode == "" {
			mode = policyconfig.ModeEnforce
		}

		for _, op := range o.Ops {
			byOp[op] = append(byOp[op], operationPolicy{id: o.PolicyID(), mode: mode, validator: v})
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

// newValidator builds the validator for one configured provider, the version
// of its content when the provider has any, and the metadata groups its policy
// reads. OASF has neither: it validates a record against a schema, which says
// nothing about what this node observed.
//
// The groups come from here because this is where the policy's source is read.
// Nothing downstream has it: a verdict names a policy and a version, and the
// expressions that reached it are gone by then.
//
// l is the layer the policy was configured in. Only CEL needs it today,
// because it is the only provider that compiles its policy here and so the
// only one that can reject reading the other layer's inputs at load time.
func newValidator(ctx context.Context, l layer, name string, p provider, policyDir string) (Validator, string, []string, error) {
	switch name {
	case policyconfig.ProviderOASF:
		v, err := newOASFValidator(p.schemaURL)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		return v, "", nil, nil
	case policyconfig.ProviderCEL:
		v, err := newCELValidator(p.expressions, l)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return v, v.version, v.factGroups, nil
	case policyconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, p.file)
		if err != nil {
			return nil, "", nil, fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return v, v.version, v.factGroups, nil
	default:
		return nil, "", nil, fmt.Errorf("unsupported provider %q", name)
	}
}

// OperationIDs returns the IDs of the policies configured at op, in config
// order, whatever mode each runs in. The slice is empty if none are.
func (r *Registry) OperationIDs(op string) []string {
	if r == nil {
		return nil
	}

	ids := make([]string, 0, len(r.byOp[op]))
	for _, p := range r.byOp[op] {
		ids = append(ids, p.id)
	}

	return ids
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
// records supplies the record content a verdict is decided on. facts supplies
// the stored observations projected into meta; a nil source leaves meta unset.
// A nil registry returns no evaluators.
func (r *Registry) Evaluators(records RecordSource, facts FactSource) []Evaluator {
	if r == nil {
		return nil
	}

	out := make([]Evaluator, len(r.evaluations))
	for i, e := range r.evaluations {
		out[i] = &evaluator{
			id:         e.id,
			version:    e.version,
			validator:  e.validator,
			factGroups: e.factGroups,
			records:    records,
			facts:      facts,
		}
	}

	return out
}

// Decide applies every operation policy configured for op, in config order.
// A nil registry or an op with no policies is a no-op and reports that the
// record passes. Messages from every failing enforcing policy are collected.
//
// Each policy applies in its own mode. A policy in off is not run at all. A
// policy in shadow is run and its decision reported, but it does not reach the
// caller: that is the whole of the difference between shadow and enforce, and
// it is why the decision has to be reached per policy rather than once.
//
// Every decision is logged here rather than by the caller, because what was
// decided and by which policy in which mode is known here and nowhere else —
// a shadow decision in particular is never returned at all.
func (r *Registry) Decide(
	ctx context.Context,
	op string,
	record *corev1.Record,
	opts ...DecideOption,
) (bool, []string, error) {
	if r == nil {
		return true, nil, nil
	}

	var all []string

	input := operationInput(op, record, opts...)

	for _, p := range r.byOp[op] {
		if p.validator == nil || p.mode == policyconfig.ModeOff {
			continue
		}

		ok, msgs, err := validateOperation(ctx, record, input, p.validator)
		if err != nil {
			return false, nil, fmt.Errorf("decide record with policy %q: %w", p.id, err)
		}

		logDecision(ctx, p, input, ok, msgs)

		if !ok && p.mode == policyconfig.ModeEnforce {
			all = append(all, msgs...)
		}
	}

	return len(all) == 0, all, nil
}

// With returns a registry that runs vs for op, each enforcing, which is what
// an operation policy does when its mode is unset. Intended for tests.
func With(op string, vs ...Validator) *Registry {
	policies := make([]operationPolicy, 0, len(vs))
	for i, v := range vs {
		policies = append(policies, operationPolicy{
			id:        fmt.Sprintf("test:%d", i),
			mode:      policyconfig.ModeEnforce,
			validator: v,
		})
	}

	return &Registry{byOp: map[string][]operationPolicy{op: policies}}
}
