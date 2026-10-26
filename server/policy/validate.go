// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Validator evaluates a policy.
//
// ValidateOperation reads the record struct. ValidateEvaluation reads a
// RecordState. A non-nil error from either means the validator failed, not the
// record.
type Validator interface {
	ValidateOperation(ctx context.Context, data *structpb.Struct) (valid bool, errors []string, warnings []string, err error)
	ValidateEvaluation(ctx context.Context, state *policyv1.RecordState) (valid bool, errors []string, warnings []string, err error)
}

// recordState is the RecordState a policy sees when nothing has been observed
// about the record yet: the content and its CID. Meta and indexed_at stay
// unset, which reads as "not produced" and "not indexed".
func recordState(record *corev1.Record) *policyv1.RecordState {
	if record == nil {
		return nil
	}

	return &policyv1.RecordState{
		Record:    record.GetData(),
		RecordCid: record.GetCid(),
	}
}

// validateEvaluation runs one evaluation policy against state, after the same nil,
// size and timeout checks Record.ValidateWith applies. The record is what
// those checks measure; state is what the validator reads.
func validateEvaluation(ctx context.Context, record *corev1.Record, state *policyv1.RecordState, v Validator) (bool, []string, error) {
	return runValidator(ctx, record, v == nil, func(ctx context.Context) (bool, []string, []string, error) {
		return v.ValidateEvaluation(ctx, state)
	})
}

// validateOperation runs one operation policy against the record struct, with
// the same nil, size and timeout checks as validateEvaluation.
func validateOperation(ctx context.Context, record *corev1.Record, v Validator) (bool, []string, error) {
	return runValidator(ctx, record, v == nil, func(ctx context.Context) (bool, []string, []string, error) {
		var data *structpb.Struct
		if record != nil {
			data = record.GetData()
		}

		return v.ValidateOperation(ctx, data)
	})
}

func runValidator(ctx context.Context, record *corev1.Record, validatorNil bool, eval func(context.Context) (bool, []string, []string, error)) (bool, []string, error) {
	if validatorNil && record != nil && record.GetData() != nil {
		return false, []string{"validator is nil"}, nil
	}

	validationCtx, cancel, msgs := record.ValidationContext(ctx)
	defer cancel()

	if msgs != nil {
		return false, msgs, nil
	}

	valid, errs, warnings, err := eval(validationCtx)
	if err != nil {
		return false, nil, fmt.Errorf("failed to validate record: %w", err)
	}

	return valid, prefixMessages(errs, warnings), nil
}

func prefixMessages(errs, warnings []string) []string {
	prefixedErrors := make([]string, len(errs))
	for i, e := range errs {
		prefixedErrors[i] = "ERROR: " + e
	}

	prefixedWarnings := make([]string, len(warnings))
	for i, w := range warnings {
		prefixedWarnings[i] = "WARNING: " + w
	}

	all := make([]string, 0, len(prefixedErrors)+len(prefixedWarnings))
	all = append(all, prefixedErrors...)
	all = append(all, prefixedWarnings...)

	return all
}
