// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// GetIdentityStatus retrieves the cached identity/ownership claim
// verification status for a record. Accepts either a CID directly or a name
// (with optional version) that is resolved first. This is a pure cache read:
// verification itself happens ahead of time at ingest and in the reconciler.
func (c *identityCtrl) GetIdentityStatus(ctx context.Context, req *identityv1.GetIdentityStatusRequest) (*identityv1.GetIdentityStatusResponse, error) {
	identityLogger.Debug("GetIdentityStatus request received", "cid", req.GetCid(), "name", req.GetName(), "version", req.GetVersion())

	cid := req.GetCid()

	if cid == "" {
		if req.GetName() == "" {
			return nil, status.Error(codes.InvalidArgument, "either cid or name is required")
		}

		resolveResp, err := c.Resolve(ctx, &identityv1.ResolveRequest{Name: req.GetName(), Version: req.Version})
		if err != nil {
			return nil, err
		}

		if len(resolveResp.GetRecords()) == 0 {
			return nil, status.Errorf(codes.NotFound, "no record found with name %q", req.GetName())
		}

		cid = resolveResp.GetRecords()[0].GetCid()
	}

	resp := &identityv1.GetIdentityStatusResponse{}

	if identityClaim, err := c.db.GetClaimByCID(cid, types.ClaimRoleIdentity); err == nil {
		resp.Identity = &identityv1.ClaimStatus{
			Verified:     identityClaim.GetStatus() == gormdb.ClaimStatusVerified,
			Subject:      identityClaim.GetSubject(),
			ErrorMessage: identityClaim.GetError(),
		}
	} else if !errors.Is(err, gormdb.ErrClaimNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to get identity claim: %v", err)
	}

	if ownershipClaim, err := c.db.GetClaimByCID(cid, types.ClaimRoleOwner); err == nil {
		resp.Owner = &identityv1.ClaimStatus{
			Verified:     ownershipClaim.GetStatus() == gormdb.ClaimStatusVerified,
			Subject:      ownershipClaim.GetSubject(),
			ErrorMessage: ownershipClaim.GetError(),
		}
	} else if !errors.Is(err, gormdb.ErrClaimNotFound) {
		return nil, status.Errorf(codes.Internal, "failed to get ownership claim: %v", err)
	}

	return resp, nil
}

// Resolve resolves a record reference (name with optional version) to CIDs.
// If version is specified, returns the exact match(es). If no version is
// specified, returns all versions (newest first).
func (c *identityCtrl) Resolve(_ context.Context, req *identityv1.ResolveRequest) (*identityv1.ResolveResponse, error) {
	identityLogger.Debug("Resolve request received", "name", req.GetName(), "version", req.GetVersion())

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	filterOptions := []types.FilterOption{
		types.WithNames(expandIdentityNameWithProtocols(req.GetName())...),
	}

	if req.GetVersion() != "" {
		filterOptions = append(filterOptions, types.WithVersions(req.GetVersion()))
	}

	records, err := c.db.GetRecords(filterOptions...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search records: %v", err)
	}

	if len(records) == 0 {
		if req.GetVersion() != "" {
			return nil, status.Errorf(codes.NotFound, "no record found with name %q and version %q", req.GetName(), req.GetVersion())
		}

		return nil, status.Errorf(codes.NotFound, "no record found with name %q", req.GetName())
	}

	refs := make([]*corev1.NamedRecordRef, 0, len(records))

	for _, r := range records {
		refs = append(refs, &corev1.NamedRecordRef{
			Name:    r.GetName(),
			Version: r.GetVersion(),
			Cid:     r.GetCid(),
		})
	}

	return &identityv1.ResolveResponse{Records: refs}, nil
}

// expandIdentityNameWithProtocols expands a bare name into exact-match plus http(s)
// variants, so records stored with a protocol-qualified name (e.g.
// "https://my-agent.example.com") are still found by a bare-name lookup.
func expandIdentityNameWithProtocols(name string) []string {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return []string{name}
	}

	return []string{
		name, // exact match for non-verifiable names
		"http://" + name,
		"https://" + name,
	}
}
