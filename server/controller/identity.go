// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var identityLogger = logging.Logger("controller/identity")

type identityCtrl struct {
	identityv1.UnimplementedIdentityServiceServer
	db types.DatabaseAPI
}

// NewIdentityController creates a new identity service controller.
func NewIdentityController(db types.DatabaseAPI) identityv1.IdentityServiceServer {
	return &identityCtrl{db: db}
}

// GetIdentityStatus returns the last verification result of a record's identity
// and ownership claims. Accepts either a CID directly or a name (with optional
// version) that is resolved first. A claim with no result is left unset.
func (c *identityCtrl) GetIdentityStatus(_ context.Context, req *identityv1.GetIdentityStatusRequest) (*identityv1.GetIdentityStatusResponse, error) {
	identityLogger.Debug("GetIdentityStatus request received", "cid", req.GetCid(), "name", req.GetName(), "version", req.GetVersion())

	cid := req.GetCid()

	if cid == "" {
		if req.GetName() == "" {
			return nil, status.Error(codes.InvalidArgument, "either cid or name is required")
		}

		// Records come back newest first, so the first is the latest version.
		records, err := findRecordsByName(c.db, req.GetName(), req.GetVersion())
		if err != nil {
			return nil, err
		}

		cid = records[0].GetCid()
	}

	identity, err := c.claimVerification(cid, types.ClaimRoleIdentity)
	if err != nil {
		return nil, err
	}

	owner, err := c.claimVerification(cid, types.ClaimRoleOwner)
	if err != nil {
		return nil, err
	}

	return &identityv1.GetIdentityStatusResponse{Identity: identity, Owner: owner}, nil
}

// Resolve resolves a record reference (name with optional version) to CIDs.
// If version is specified, returns the exact match(es).
// If no version is specified, returns all versions (newest first).
func (c *identityCtrl) Resolve(_ context.Context, req *identityv1.ResolveRequest) (*identityv1.ResolveResponse, error) {
	identityLogger.Debug("Resolve request received", "name", req.GetName(), "version", req.GetVersion())

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	records, err := findRecordsByName(c.db, req.GetName(), req.GetVersion())
	if err != nil {
		return nil, err
	}

	return &identityv1.ResolveResponse{Records: namedRecordRefs(records)}, nil
}

// claimVerification returns the stored result for a record's claim of role, or
// nil if the claim has no result yet.
func (c *identityCtrl) claimVerification(cid, role string) (*identityv1.ClaimVerification, error) {
	claim, err := c.db.GetIdentityClaimByCID(cid, role)
	if err != nil {
		if errors.Is(err, gormdb.ErrIdentityClaimNotFound) {
			return nil, nil //nolint:nilnil // an absent claim is a valid, empty result
		}

		return nil, status.Errorf(codes.Internal, "failed to get %s claim: %v", role, err)
	}

	result := &identityv1.ClaimVerification{
		Role:       claimRole(claim.GetRole()),
		Subject:    claim.GetSubject(),
		Status:     claimVerificationStatus(claim.GetStatus()),
		VerifiedAt: timestamppb.New(claim.GetVerifiedAt()),
	}

	if msg := claim.GetError(); msg != "" {
		result.Error = &msg
	}

	return result, nil
}

func claimRole(role string) identityv1.ClaimRole {
	switch role {
	case types.ClaimRoleIdentity:
		return identityv1.ClaimRole_CLAIM_ROLE_IDENTITY
	case types.ClaimRoleOwner:
		return identityv1.ClaimRole_CLAIM_ROLE_OWNER
	default:
		return identityv1.ClaimRole_CLAIM_ROLE_UNSPECIFIED
	}
}

func claimVerificationStatus(status string) identityv1.ClaimVerificationStatus {
	switch status {
	case types.ClaimStatusVerified:
		return identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED
	case types.ClaimStatusFailed:
		return identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED
	default:
		return identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_UNSPECIFIED
	}
}
