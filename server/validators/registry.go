// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"fmt"
	"slices"

	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/oasf-sdk/pkg/validator"
)

var logger = logging.Logger("validators")

// Registry maps operations to the record validators that should run for them.
type Registry struct {
	byOp map[string][]corev1.Validator
}

// NewRegistry constructs validators from config. Entries that share a schema URL
// reuse one OASF-SDK instance. An empty list yields a registry that returns no
// validators for every op.
func NewRegistry(entries validatorsconfig.Config) (*Registry, error) {
	if err := entries.Validate(); err != nil {
		return nil, fmt.Errorf("invalid validators config: %w", err)
	}

	byURL := make(map[string]corev1.Validator)
	byOp := make(map[string][]corev1.Validator)

	for i, entry := range entries {
		v, err := validatorFor(byURL, entry)
		if err != nil {
			return nil, fmt.Errorf("validators[%d]: %w", i, err)
		}

		for _, op := range entry.Ops {
			byOp[op] = appendUnique(byOp[op], v)
		}

		logger.Info("Record validator configured",
			"provider", entry.Provider,
			"schema_url", entry.SchemaURL,
			"op", entry.Ops)
	}

	return &Registry{byOp: byOp}, nil
}

func validatorFor(byURL map[string]corev1.Validator, entry validatorsconfig.Validator) (corev1.Validator, error) {
	if v, ok := byURL[entry.SchemaURL]; ok {
		return v, nil
	}

	v, err := validator.New(entry.SchemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OASF validator: %w", err)
	}

	byURL[entry.SchemaURL] = v

	return v, nil
}

func appendUnique(dst []corev1.Validator, v corev1.Validator) []corev1.Validator {
	if slices.Contains(dst, v) {
		return dst
	}

	return append(dst, v)
}

// For returns the validators configured for op, in config order.
// The slice is empty if none are configured.
func (r *Registry) For(op string) []corev1.Validator {
	if r == nil {
		return nil
	}

	return r.byOp[op]
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
