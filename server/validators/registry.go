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

// Registry maps operations to the record validators that should run for them,
// and holds the content policies the validators with op evaluate define.
type Registry struct {
	byOp     map[string][]corev1.Validator
	policies []Policy
}

// Policy is a content policy: a validator with op evaluate. The reconciler
// evaluates every record against it and stores the verdict, and the server
// enforces the verdicts of the policies it is told to.
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
// providers have op evaluate; the task refuses a policy with no version.
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

// NewRegistry constructs validators from config. policyDir is the directory
// named .rego files are loaded from (server/reconciler policy.dir). An empty
// list yields a registry that returns no validators for every op.
func NewRegistry(ctx context.Context, entries validatorsconfig.Config, policyDir string) (*Registry, error) {
	if err := entries.Validate(); err != nil {
		return nil, fmt.Errorf("invalid validators config: %w", err)
	}

	byOp := make(map[string][]corev1.Validator)

	var policies []Policy

	for i, entry := range entries {
		b, err := validatorFor(ctx, entry, policyDir)
		if err != nil {
			return nil, fmt.Errorf("validators[%d]: %w", i, err)
		}

		for _, op := range entry.Ops {
			byOp[op] = append(byOp[op], b.validator)
		}

		if entry.HasOp(validatorsconfig.OpEvaluate) {
			policies = append(policies, Policy{ID: entry.PolicyID(), Version: b.version, Validator: b.validator})
		}
	}

	return &Registry{byOp: byOp, policies: policies}, nil
}

func validatorFor(ctx context.Context, entry validatorsconfig.Validator, policyDir string) (built, error) {
	switch entry.Provider {
	case validatorsconfig.ProviderOASF:
		v, err := validator.New(entry.ConfigString(validatorsconfig.ConfigKeySchemaURL))
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		return built{validator: v}, nil
	case validatorsconfig.ProviderCEL:
		v, err := newCELValidator(entry)
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize CEL validator: %w", err)
		}

		return built{validator: v, version: v.version}, nil
	case validatorsconfig.ProviderOPA:
		v, err := newOPAValidator(ctx, policyDir, entry.ConfigString(validatorsconfig.ConfigKeyFile))
		if err != nil {
			return built{}, fmt.Errorf("failed to initialize OPA validator: %w", err)
		}

		return built{validator: v, version: v.version}, nil
	default:
		return built{}, fmt.Errorf("unsupported provider %q", entry.Provider)
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
