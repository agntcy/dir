// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var recordGateLogger = logging.Logger("policy/record-gate")

// RecordGate decides whether a record may be served when it is pulled by CID
// rather than found by searching.
//
// The operation policies configured at pull are the whole of that decision:
// a read by CID applies no filter the database adds of its own.
type RecordGate struct {
	policies *Registry

	// observer counts the records refused, and those a shadow policy would
	// have refused. Nil tells no one.
	observer types.PolicyGateObserver
}

// NewRecordGate returns the gate every read by CID passes through.
//
// policies may be absent: a node configuring no policy at pull serves
// whatever it stores, which is the default. The gate itself must not be: a
// read path reaching Check with no gate panics rather than quietly serving
// everything, which is the mistake this type exists to prevent.
func NewRecordGate(policies *Registry, observer types.PolicyGateObserver) *RecordGate {
	return &RecordGate{policies: policies, observer: observer}
}

// Check returns nil when the record with cid may be served.
//
// When the gate excludes it, it returns types.RecordExcludedError, which tells
// the caller the record is not available under the node's content policy
// without saying which policy or why. When that cannot be determined it
// returns Internal: the check fails closed.
func (g *RecordGate) Check(ctx context.Context, cid string) error {
	ok, _, refused, err := g.policies.DecidePull(ctx, cid)
	if err != nil {
		recordGateLogger.Error("Failed to decide a read by CID", "cid", cid, "error", err)

		return status.Error(codes.Internal, "failed to check record access") //nolint:wrapcheck // a gRPC status for the caller
	}

	// Counted here rather than where the policies run, because the question
	// the counter answers is about reads, not about policies: one refused
	// read is one increment however many policies refused it. A shadow
	// policy reaches this with ok still true, which is what lets the
	// would-have-excluded series be anything but zero.
	if refused != "" && g.observer != nil {
		g.observer.PullExcluded(refused)
	}

	if !ok {
		recordGateLogger.Debug("Record refused by an operation policy", "cid", cid, "op", policyconfig.OpPull)

		return types.RecordExcludedError(cid) //nolint:wrapcheck // a gRPC status for the caller
	}

	return nil
}
