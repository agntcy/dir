// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
)

// RecordSource reads a record's content from the store. An evaluation selects
// records from the index, which holds only the fields searches use, and the
// policy decides on the record as published.
type RecordSource interface {
	Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error)
}

// Evaluator checks records against one evaluation policy.
//
// The reconciler task owns scheduling, record selection and persistence. An
// evaluator only reaches a verdict.
type Evaluator interface {
	// PolicyID is a stable identifier of the policy, unchanged across
	// versions, e.g. "opa:require-annotation".
	PolicyID() string

	// PolicyVersion is the content version of the policy currently loaded.
	// Bumping it makes every record evaluated under the previous version
	// due for re-evaluation.
	PolicyVersion() string

	// Evaluate reports whether record complies with the policy, with a
	// human-readable reason when it does not.
	//
	// A non-nil error means the evaluator could not reach a verdict: the
	// validator failed, not the record. The task stores that as a failed
	// evaluation, which reads as non-compliant, never as a pass.
	Evaluate(ctx context.Context, record coretypes.Record) (compliant bool, reason string, err error)
}

// evaluator decides one evaluation policy by reading the record and running
// its validator. A record complies when the validator accepts it.
type evaluator struct {
	id        string
	version   string
	validator Validator
	records   RecordSource
}

// PolicyID returns the policy's ID.
func (e *evaluator) PolicyID() string { return e.id }

// PolicyVersion returns the version of the policy as it was loaded.
func (e *evaluator) PolicyVersion() string { return e.version }

// Evaluate reads the record from the store and has the validator decide on it.
// When the record is rejected the reason is what the validator reported. A
// record that cannot be read, or a validator that cannot reach a verdict, is an
// error: the task stores that as a failed evaluation, which stays excluded.
func (e *evaluator) Evaluate(ctx context.Context, record coretypes.Record) (bool, string, error) {
	cid := record.GetCid()

	full, err := e.records.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return false, "", fmt.Errorf("read record %s: %w", cid, err)
	}

	compliant, messages, err := validateEvaluation(ctx, full, recordState(full), e.validator)
	if err != nil {
		return false, "", fmt.Errorf("evaluate record %s against policy %s: %w", cid, e.id, err)
	}

	if compliant {
		return true, "", nil
	}

	return false, strings.Join(messages, "; "), nil
}
