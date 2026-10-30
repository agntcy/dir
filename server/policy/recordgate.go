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

// RecordGate decides whether a record may be served when it is fetched by CID
// rather than found by searching.
//
// It carries both of the things that decide: the stored verdicts the search
// path already filters on, and the operation policies configured at pull. They
// travel together so that a read path cannot consult one and miss the other,
// which is a mistake this code has made before.
type RecordGate struct {
	servability types.RecordServabilityAPI
	policies    *Registry
}

// NewRecordGate returns the gate every read by CID passes through.
//
// Either half may be absent: a node that enforces no verdicts and configures
// no policy at pull serves whatever it stores, which is the default. The gate
// itself must not be: a read path reaching Check with no gate panics rather
// than quietly serving everything, which is the mistake this type exists to
// prevent.
func NewRecordGate(servability types.RecordServabilityAPI, policies *Registry) *RecordGate {
	return &RecordGate{servability: servability, policies: policies}
}

// Check returns nil when the record with cid may be served.
//
// When the gate excludes it, it returns types.RecordExcludedError, which tells
// the caller the record is not available under the node's content policy
// without saying which policy or why. When that cannot be determined it
// returns Internal: the check fails closed.
func (g *RecordGate) Check(ctx context.Context, cid string) error {
	if g.servability != nil {
		ok, err := g.servability.IsRecordServable(cid)
		if err != nil {
			recordGateLogger.Error("Failed to check whether a record may be served", "cid", cid, "error", err)

			return status.Error(codes.Internal, "failed to check record access") //nolint:wrapcheck // a gRPC status for the caller
		}

		if !ok {
			recordGateLogger.Debug("Record excluded by content policy", "cid", cid)

			return types.RecordExcludedError(cid) //nolint:wrapcheck // a gRPC status for the caller
		}
	}

	ok, _, err := g.policies.DecidePull(ctx, cid)
	if err != nil {
		recordGateLogger.Error("Failed to decide a read by CID", "cid", cid, "error", err)

		return status.Error(codes.Internal, "failed to check record access") //nolint:wrapcheck // a gRPC status for the caller
	}

	if !ok {
		recordGateLogger.Debug("Record refused by an operation policy", "cid", cid, "op", policyconfig.OpPull)

		return types.RecordExcludedError(cid) //nolint:wrapcheck // a gRPC status for the caller
	}

	return nil
}
