// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/agntcy/oasf-sdk/pkg/validator"
)

// versionLength is how many hex digits of the content hash make a policy
// version: enough that two different policies do not share one, short enough to
// read in a metric label.
const versionLength = 16

// Registry maps operations to the operation policies that decide at them, and
// holds the evaluation policies.
type Registry struct {
	byOp     map[string][]corev1.Validator
	policies []Policy
}

// Policy is an evaluation policy. The reconciler evaluates every record
// against it and stores the verdict, and the server enforces the verdicts of
// the policies it is told to.
type Policy struct {
	// ID names the policy, such as "opa:require-license". It stays the same
	// when the policy's content changes.
	ID string

	// Version identifies the policy's content: an edit gives a new one, and
	// nothing else does. Records are evaluated again when it changes.
	Version string

	// Validator reaches a verdict on a record.
	Validator corev1.Validator
}

// built is a validator made from its config, with the version of its content
// when its provider can define a policy. Config validation lets only such
// providers appear as evaluations; the task refuses a policy with no version.
type built struct {
	validator corev1.Validator
	version   string
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
// Two empty lists yield a registry with no policies and no validators for
// every op.
func NewRegistry(
	ctx context.Context,
	evaluations validatorsconfig.Evaluations,
	operations validatorsconfig.Operations,
	policyDir string,
) (*Registry, error) {
	if err := evaluations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := operations.Validate(); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	if err := validatorsconfig.ValidateNames(evaluations, operations); err != nil {
		return nil, fmt.Errorf("invalid policy config: %w", err)
	}

	var policies []Policy

	for i, e := range evaluations {
		b, err := buildProvider(ctx, e.Provider, provider{file: e.File, expressions: e.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("evaluations[%d]: %w", i, err)
		}

		policies = append(policies, Policy{ID: e.PolicyID(), Version: b.version, Validator: b.validator})
	}

	byOp := make(map[string][]corev1.Validator)

	for i, o := range operations {
		b, err := buildProvider(ctx, o.Provider,
			provider{schemaURL: o.SchemaURL, file: o.File, expressions: o.Expressions}, policyDir)
		if err != nil {
			return nil, fmt.Errorf("operations[%d]: %w", i, err)
		}

		for _, op := range o.Ops {
			byOp[op] = append(byOp[op], b.validator)
		}
	}

	return &Registry{byOp: byOp, policies: policies}, nil
}

// provider is the settings of whichever backend is configured. Config
// validation has already rejected the ones that do not apply.
type provider struct {
	schemaURL   string
	file        string
	expressions []string
}

func buildProvider(ctx context.Context, name string, p provider, policyDir string) (built, error) {
	switch name {
	case validatorsconfig.ProviderOASF:
		v, err := validator.New(p.schemaURL)
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		return built{validator: v}, nil
	case validatorsconfig.ProviderCEL:
		v, err := newCELValidator(p.expressions)
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return built{validator: v, version: v.version}, nil
	case validatorsconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, p.file)
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return built{validator: v, version: v.version}, nil
	default:
		return built{}, fmt.Errorf("unsupported provider %q", name)
	}
}

// For returns the validators configured for op, in config order.
// The slice is empty if none are configured.
func (r *Registry) For(op string) []corev1.Validator {
	if r == nil {
		return nil
	}

	return r.byOp[op]
}

// Policies returns the content policies, in config order. The slice is empty if
// none are configured.
func (r *Registry) Policies() []Policy {
	if r == nil {
		return nil
	}

	return slices.Clone(r.policies)
}

// Run applies every validator configured for op, in config order.
// A nil registry or an op with no validators is a no-op and reports valid.
// Messages from every failing validator are collected.
func (r *Registry) Run(ctx context.Context, op string, record *corev1.Record) (bool, []string, error) {
	var all []string

	for _, v := range r.For(op) {
		if v == nil {
			continue
		}

		ok, msgs, err := record.ValidateWith(ctx, v)
		if err != nil {
			return false, nil, fmt.Errorf("validate record: %w", err)
		}

		if !ok {
			all = append(all, msgs...)
		}
	}

	return len(all) == 0, all, nil
}

// With returns a registry that runs vs for op. Intended for tests.
func With(op string, vs ...corev1.Validator) *Registry {
	return &Registry{byOp: map[string][]corev1.Validator{op: vs}}
}
