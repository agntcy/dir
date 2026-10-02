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

// RecordSource reads a record's content from the store. The task selects
// records from the index, which holds only the fields searches use, and a
// validator decides on the record as published.
type RecordSource interface {
	Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error)
}

// ValidatorEvaluator is an Evaluator whose verdicts come from a record
// validator, such as an OPA or CEL policy: a record complies when the
// validator accepts it.
type ValidatorEvaluator struct {
	id        string
	version   string
	validator corev1.Validator
	records   RecordSource
}

// NewValidatorEvaluator returns the evaluator for the policy id at version,
// decided by validator on the records records returns.
func NewValidatorEvaluator(id, version string, validator corev1.Validator, records RecordSource) *ValidatorEvaluator {
	return &ValidatorEvaluator{id: id, version: version, validator: validator, records: records}
}

// PolicyID returns the policy's ID.
func (e *ValidatorEvaluator) PolicyID() string { return e.id }

// PolicyVersion returns the version of the policy as it was loaded.
func (e *ValidatorEvaluator) PolicyVersion() string { return e.version }

// Evaluate reads the record from the store and has the validator decide on it.
// When the record is rejected the reason is what the validator reported. A
// record that cannot be read, or a validator that cannot reach a verdict, is an
// error: the task stores that as a failed evaluation, which stays excluded.
func (e *ValidatorEvaluator) Evaluate(ctx context.Context, record coretypes.Record) (bool, string, error) {
	cid := record.GetCid()

	full, err := e.records.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return false, "", fmt.Errorf("read record %s: %w", cid, err)
	}

	compliant, messages, err := full.ValidateWith(ctx, e.validator)
	if err != nil {
		return false, "", fmt.Errorf("validate record %s against policy %s: %w", cid, e.id, err)
	}

	if compliant {
		return true, "", nil
	}

	return false, strings.Join(messages, "; "), nil
}
