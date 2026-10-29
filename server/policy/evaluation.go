// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// FactGroups are the groups of record metadata this version of the policy
	// reads, which decide which metadata changes make its verdicts stale.
	// Empty means none: nothing a scanner or verifier writes can change the
	// answer, so its verdicts stand until the record or the policy does.
	//
	// It is reported alongside the version because the two are registered
	// together: a version recorded without its dependencies would read as
	// depending on nothing and would never be invalidated.
	FactGroups() []string

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
	id         string
	version    string
	validator  Validator
	factGroups []string
	records    RecordSource
	facts      FactSource
}

// PolicyID returns the policy's ID.
func (e *evaluator) PolicyID() string { return e.id }

// PolicyVersion returns the version of the policy as it was loaded.
func (e *evaluator) PolicyVersion() string { return e.version }

// FactGroups returns the metadata groups the policy reads, as read off its
// source when it was loaded.
func (e *evaluator) FactGroups() []string { return e.factGroups }

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

	state, err := e.recordState(ctx, full)
	if err != nil {
		return false, "", err
	}

	compliant, messages, err := validateEvaluation(ctx, full, state, e.validator)
	if err != nil {
		return false, "", fmt.Errorf("evaluate record %s against policy %s: %w", cid, e.id, err)
	}

	if compliant {
		return true, "", nil
	}

	return false, strings.Join(messages, "; "), nil
}

// recordState returns the input an evaluation policy sees. With no fact
// source, meta and indexed_at stay unset. With one, meta is the projection of
// what this node has stored, including the explicit "nothing here" statuses.
func (e *evaluator) recordState(ctx context.Context, record *corev1.Record) (*policyv1.RecordState, error) {
	state := recordState(record)
	if e.facts == nil || record == nil {
		return state, nil
	}

	observed, err := e.facts.Facts(ctx, record.GetCid())
	if err != nil {
		return nil, fmt.Errorf("read facts for record %s: %w", record.GetCid(), err)
	}

	state.Meta = project(record, observed)

	// Left unset for a record that is not indexed, so that absent reads as
	// "not indexed here" rather than as the zero time, which a policy
	// comparing against a date would read as older than anything.
	if !observed.IndexedAt.IsZero() {
		state.IndexedAt = timestamppb.New(observed.IndexedAt)
	}

	return state, nil
}
