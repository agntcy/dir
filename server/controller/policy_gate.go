// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var policyGateLogger = logging.Logger("controller/policy-gate")

// recordServability is the part of the database a fetch by CID consults.
type recordServability interface {
	IsRecordServable(cid string) (bool, error)
}

// checkRecordServable is the policy gate for reads that fetch a record by
// CID instead of searching. It returns nil when the record may be served.
// When the gate excludes it, it returns types.RecordExcludedError, which tells
// the caller the record is not available under the node's content policy
// without saying which policy or why. When that cannot be determined it
// returns Internal: the check fails closed.
func checkRecordServable(db recordServability, cid string) error {
	ok, err := db.IsRecordServable(cid)
	if err != nil {
		policyGateLogger.Error("Failed to check whether a record may be served", "cid", cid, "error", err)

		return status.Error(codes.Internal, "failed to check record access") //nolint:wrapcheck // a gRPC status for the caller
	}

	if !ok {
		policyGateLogger.Debug("Record excluded by content policy", "cid", cid)

		return types.RecordExcludedError(cid) //nolint:wrapcheck // a gRPC status for the caller
	}

	return nil
}
