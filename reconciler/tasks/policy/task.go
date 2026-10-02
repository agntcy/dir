// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package policy implements the policy evaluation reconciler task. It
// evaluates records against each registered policy and stores the verdicts
// that read paths filter on.
package policy

import (
	"context"
	"errors"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/policy")

// failedReason is stored in place of an evaluator's error. Evaluator errors
// can carry file paths, network addresses or engine internals, and the row is
// queryable, so the detail goes to the log and only this text is persisted.
const failedReason = "policy evaluator error"

// Task implements the policy evaluation reconciler task.
type Task struct {
	config     Config
	db         types.DatabaseAPI
	evaluators []Evaluator
}

// NewTask creates a policy evaluation task for the given evaluators. Every
// evaluator must report a non-empty policy ID and version, and no two may
// share a policy ID, since they would overwrite each other's verdicts.
func NewTask(config Config, db types.DatabaseAPI, evaluators ...Evaluator) (*Task, error) {
	seen := make(map[string]struct{}, len(evaluators))

	for i, ev := range evaluators {
		if ev == nil {
			return nil, fmt.Errorf("evaluator %d is nil", i)
		}

		policyID := ev.PolicyID()
		if policyID == "" {
			return nil, fmt.Errorf("evaluator %d has an empty policy ID", i)
		}

		if ev.PolicyVersion() == "" {
			return nil, fmt.Errorf("policy %q has an empty version", policyID)
		}

		if _, dup := seen[policyID]; dup {
			return nil, fmt.Errorf("policy %q is registered more than once", policyID)
		}

		seen[policyID] = struct{}{}
	}

	return &Task{
		config:     config,
		db:         db,
		evaluators: evaluators,
	}, nil
}

// TaskName is the name the task runs under, which is what another task asks
// the service to run it by.
const TaskName = "policy"

// Name returns the task name.
func (t *Task) Name() string {
	return TaskName
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled: turned on in configuration
// and with at least one policy to evaluate.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled && len(t.evaluators) > 0
}

// Run evaluates, for each policy, every record without a current verdict:
// never evaluated, evaluated under a superseded version, or whose last
// evaluation failed. Records that predate a policy are selected the same way
// as new ones, so backfill needs no separate pass.
//
// A policy whose records cannot be selected does not stop the others; the
// failures are returned together.
func (t *Task) Run(ctx context.Context) error {
	var errs []error

	for _, ev := range t.evaluators {
		if err := t.runPolicy(ctx, ev); err != nil {
			errs = append(errs, err)
		}

		if ctx.Err() != nil {
			break
		}
	}

	return errors.Join(errs...)
}

// outcomes counts how the verdicts of a run came out.
type outcomes struct {
	compliant, nonCompliant, failed int
}

func (o *outcomes) total() int { return o.compliant + o.nonCompliant + o.failed }

// runPolicy evaluates the records that need a verdict for one policy, a batch
// at a time, until none is left.
//
// The policy version is read once, so every row written in a run carries the
// version the records were selected under. If the evaluator reloads mid-run,
// those rows are simply stale next run and re-selected.
//
// The version is registered first, so the server learns of a changed policy
// while the first records are still being evaluated. Batches follow record_cid
// order, each starting after the last record of the one before: a record whose
// evaluation failed is selected again by the next run, not by the next batch
// of this one.
func (t *Task) runPolicy(ctx context.Context, ev Evaluator) error {
	policyID, policyVersion := ev.PolicyID(), ev.PolicyVersion()

	if err := t.db.RegisterPolicyVersion(policyID, policyVersion); err != nil {
		return fmt.Errorf("register version of policy %q: %w", policyID, err)
	}

	var (
		out   outcomes
		after string
	)

	for {
		records, err := t.db.GetRecordsNeedingPolicyEvaluation(policyID, policyVersion, after, t.config.GetBatchSize())
		if err != nil {
			return fmt.Errorf("select records for policy %q: %w", policyID, err)
		}

		if len(records) == 0 {
			break
		}

		logger.Debug("Evaluating a batch of records",
			"policy_id", policyID, "policy_version", policyVersion, "batch", len(records), "after", after)

		if err := t.evaluateBatch(ctx, ev, policyID, policyVersion, records, &out); err != nil {
			return err
		}

		after = records[len(records)-1].GetCid()
	}

	if out.total() == 0 {
		logger.Debug("No records need policy evaluation", "policy_id", policyID, "policy_version", policyVersion)

		return nil
	}

	// The per-run total is the backfill progress signal: a run that evaluates
	// nothing means every record has a verdict under the current version.
	logger.Info("Policy evaluation complete",
		"policy_id", policyID, "policy_version", policyVersion,
		"compliant", out.compliant, "non_compliant", out.nonCompliant, "failed", out.failed)

	return nil
}

// evaluateBatch evaluates and stores a verdict for each record of one batch,
// counting how each came out in out.
func (t *Task) evaluateBatch(ctx context.Context, ev Evaluator, policyID, policyVersion string, records []coretypes.Record, out *outcomes) error {
	for _, record := range records {
		row, evalErr := t.evaluateRecord(ctx, ev, policyID, policyVersion, record)

		// On shutdown the evaluator reports the cancellation, which says
		// nothing about the record. Store nothing; the rest wait for the
		// next run.
		if ctx.Err() != nil {
			logger.Info("Policy evaluation interrupted",
				"policy_id", policyID, "policy_version", policyVersion, "evaluated", out.total())

			return fmt.Errorf("policy %q evaluation interrupted: %w", policyID, ctx.Err())
		}

		if evalErr != nil {
			logger.Warn("Policy evaluation failed", "policy_id", policyID, "record_cid", row.RecordCID, "error", evalErr)
		}

		if err := t.db.UpsertPolicyEvaluation(row); err != nil {
			logger.Warn("Failed to store policy evaluation", "policy_id", policyID, "record_cid", row.RecordCID, "error", err)

			out.failed++

			continue
		}

		switch {
		case row.Status == types.PolicyEvalStatusFailed:
			out.failed++
		case row.Compliant:
			out.compliant++
		default:
			out.nonCompliant++
		}
	}

	return nil
}

// evaluateRecord evaluates one record under its own deadline and returns the
// row to store, along with the evaluator's error, if any, for logging. The row
// already reflects that error as a failed evaluation.
func (t *Task) evaluateRecord(ctx context.Context, ev Evaluator, policyID, policyVersion string, record coretypes.Record) (*gormdb.PolicyEvaluation, error) {
	recordCtx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
	defer cancel()

	compliant, reason, err := ev.Evaluate(recordCtx, record)

	row := &gormdb.PolicyEvaluation{
		RecordCID:     record.GetCid(),
		PolicyID:      policyID,
		PolicyVersion: policyVersion,
		Compliant:     compliant,
		Status:        types.PolicyEvalStatusEvaluated,
		Reason:        reason,
	}

	if err != nil {
		row.Compliant = false
		row.Status = types.PolicyEvalStatusFailed
		row.Reason = failedReason

		return row, fmt.Errorf("evaluate record: %w", err)
	}

	if compliant {
		row.Reason = ""
	}

	return row, nil
}
