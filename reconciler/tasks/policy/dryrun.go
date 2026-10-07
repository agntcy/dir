// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
)

// dryRunIDPrefix is put in front of a policy's ID to select the records a dry
// run evaluates. No verdict is ever stored under such an ID, so every indexed
// record is selected, including those the deployed policy of the same ID
// already has a verdict for.
const dryRunIDPrefix = "dry-run:"

// DryRunDB is the part of the database a dry run uses. It has no method that
// writes, so a dry run cannot store a verdict, register a version or change
// anything a read returns.
type DryRunDB interface {
	// GetRecordsNeedingPolicyEvaluation returns records in record_cid order,
	// after afterCID, at most limit of them, that have no evaluated verdict
	// under the policy ID and version.
	GetRecordsNeedingPolicyEvaluation(policyID, policyVersion, afterCID string, limit int) ([]coretypes.Record, error)
}

// Excluded is a record a policy would exclude.
type Excluded struct {
	// CID is the record's CID.
	CID string `json:"cid"`

	// Reason is what the policy reported, or for a record the evaluator could
	// not judge, what went wrong.
	Reason string `json:"reason"`

	// Failed is set when the evaluator reached no verdict. Such a record is
	// excluded as well: a record is served only with a passing verdict.
	Failed bool `json:"failed,omitempty"`
}

// DryRunReport is what evaluating a policy against the node's records, without
// storing anything, found.
type DryRunReport struct {
	// PolicyID and PolicyVersion identify the candidate policy.
	PolicyID      string `json:"policy_id"`
	PolicyVersion string `json:"policy_version"`

	// Evaluated is how many records the policy was run on, and the three that
	// follow how they came out: Compliant plus NonCompliant plus Failed is
	// Evaluated.
	Evaluated    int `json:"evaluated"`
	Compliant    int `json:"compliant"`
	NonCompliant int `json:"non_compliant"`
	Failed       int `json:"failed"`

	// Sample is the first records, in CID order, the policy would exclude.
	Sample []Excluded `json:"sample"`
}

// WouldExclude is how many records the policy would keep from being served.
func (r *DryRunReport) WouldExclude() int {
	return r.NonCompliant + r.Failed
}

// DryRun evaluates ev against every indexed record and reports how many it
// would exclude and, for the first samples of them, why. It reads the records
// and stores nothing: no verdict, no version, and so nothing a read returns
// changes. The records are evaluated with the limits cfg sets for the policy
// task, so a record that takes too long fails here as it would there.
//
// When ctx ends the report so far is returned with the error.
func DryRun(ctx context.Context, db DryRunDB, ev Evaluator, cfg Config, samples int) (*DryRunReport, error) {
	report := &DryRunReport{
		PolicyID:      ev.PolicyID(),
		PolicyVersion: ev.PolicyVersion(),
		Sample:        []Excluded{},
	}

	selectID := dryRunIDPrefix + report.PolicyID
	after := ""

	for {
		records, err := db.GetRecordsNeedingPolicyEvaluation(selectID, report.PolicyVersion, after, cfg.GetBatchSize())
		if err != nil {
			return report, fmt.Errorf("select records to evaluate: %w", err)
		}

		if len(records) == 0 {
			return report, nil
		}

		for _, record := range records {
			if err := report.evaluate(ctx, ev, record, cfg.GetRecordTimeout(), samples); err != nil {
				return report, err
			}
		}

		after = records[len(records)-1].GetCid()
	}
}

// evaluate runs the policy on one record under its own deadline and counts the
// outcome. It returns an error only when ctx ended, which says nothing about
// the record: it is not counted.
func (r *DryRunReport) evaluate(ctx context.Context, ev Evaluator, record coretypes.Record, timeout time.Duration, samples int) error {
	recordCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	compliant, reason, err := ev.Evaluate(recordCtx, record)

	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("dry run interrupted after %d records: %w", r.Evaluated, ctxErr)
	}

	r.Evaluated++

	switch {
	case err != nil:
		r.Failed++
		r.keep(Excluded{CID: record.GetCid(), Reason: err.Error(), Failed: true}, samples)
	case compliant:
		r.Compliant++
	default:
		r.NonCompliant++
		r.keep(Excluded{CID: record.GetCid(), Reason: reason}, samples)
	}

	return nil
}

// keep adds a record to the sample while there is room.
func (r *DryRunReport) keep(e Excluded, samples int) {
	if len(r.Sample) < samples {
		r.Sample = append(r.Sample, e)
	}
}
