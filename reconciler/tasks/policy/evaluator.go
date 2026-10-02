// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/types"
)

// Evaluator checks records against one policy.
//
// This task owns scheduling, record selection and persistence; it has no
// opinion on how a policy is expressed. Concrete evaluators (OPA, OASF
// schema validation, ...) come from #2135 and #2157.
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
	// A non-nil error means the evaluator could not reach a verdict — the
	// policy engine failed, not the record. The task stores that as a failed
	// evaluation, which reads as non-compliant (see
	// types.EvaluatedPolicyStatuses), never as a pass.
	Evaluate(ctx context.Context, record coretypes.Record) (compliant bool, reason string, err error)
}
