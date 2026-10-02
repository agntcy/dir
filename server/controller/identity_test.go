// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeRecord struct {
	coretypes.Record
	cid, name, version string
}

func (r *fakeRecord) GetCid() string     { return r.cid }
func (r *fakeRecord) GetName() string    { return r.name }
func (r *fakeRecord) GetVersion() string { return r.version }

// fakeIdentityDB serves records and claims from memory. Records are returned
// newest first, as the real database does, and filtered by the requested
// versions.
type fakeIdentityDB struct {
	types.DatabaseAPI
	records    []*fakeRecord // newest first
	claims     map[string]*gormdb.IdentityClaim
	claimErr   error
	recordsErr error
	gotFilters types.RecordFilters
}

func (f *fakeIdentityDB) GetRecords(opts ...types.FilterOption) ([]coretypes.Record, error) {
	f.gotFilters = types.RecordFilters{}
	for _, opt := range opts {
		opt(&f.gotFilters)
	}

	var out []coretypes.Record

	for _, r := range f.records {
		if len(f.gotFilters.Versions) > 0 && r.version != f.gotFilters.Versions[0] {
			continue
		}

		if slices.Contains(f.gotFilters.Names, r.name) {
			out = append(out, r)
		}
	}

	return out, nil
}

func (f *fakeIdentityDB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	if f.recordsErr != nil {
		return nil, f.recordsErr
	}

	var filters types.RecordFilters
	for _, opt := range opts {
		opt(&filters)
	}

	var cids []string

	for _, r := range f.records {
		if slices.Contains(filters.CIDs, r.cid) {
			cids = append(cids, r.cid)
		}
	}

	return cids, nil
}

func (f *fakeIdentityDB) GetIdentityClaimByCID(cid, role string) (types.IdentityClaimObject, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}

	if c, ok := f.claims[cid+"/"+role]; ok {
		return c, nil
	}

	return nil, gormdb.ErrIdentityClaimNotFound
}

func newFakeIdentityDB() *fakeIdentityDB {
	return &fakeIdentityDB{
		records: []*fakeRecord{
			{cid: "cid-v2", name: "acme.com/agent", version: "2.0.0"},
			{cid: "cid-v1", name: "acme.com/agent", version: "1.0.0"},
			{cid: "cid-other", name: "other.com/agent", version: "1.0.0"},
		},
		claims: map[string]*gormdb.IdentityClaim{},
	}
}

func TestIdentityGetIdentityStatus_NoClaims(t *testing.T) {
	ctrl := NewIdentityController(newFakeIdentityDB())

	resp, err := ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	require.NoError(t, err)
	assert.Nil(t, resp.GetIdentity())
	assert.Nil(t, resp.GetOwner())
}

func TestIdentityGetIdentityStatus_ReturnsBothClaims(t *testing.T) {
	db := newFakeIdentityDB()
	verifiedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	db.claims["cid-v1/"+types.ClaimRoleIdentity] = &gormdb.IdentityClaim{
		RecordCID: "cid-v1", Role: types.ClaimRoleIdentity, Subject: "did:web:acme.com:agent",
		Status: types.ClaimStatusVerified, VerifiedAt: verifiedAt,
	}
	db.claims["cid-v1/"+types.ClaimRoleOwner] = &gormdb.IdentityClaim{
		RecordCID: "cid-v1", Role: types.ClaimRoleOwner, Subject: "dns:acme.com",
		Status: types.ClaimStatusFailed, Error: "no key found", VerifiedAt: verifiedAt,
	}

	resp, err := NewIdentityController(db).GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	require.NoError(t, err)

	identity := resp.GetIdentity()
	require.NotNil(t, identity)
	assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, identity.GetRole())
	assert.Equal(t, "did:web:acme.com:agent", identity.GetSubject())
	assert.Equal(t, identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED, identity.GetStatus())
	assert.Nil(t, identity.Error)
	assert.True(t, verifiedAt.Equal(identity.GetVerifiedAt().AsTime()))

	owner := resp.GetOwner()
	require.NotNil(t, owner)
	assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_OWNER, owner.GetRole())
	assert.Equal(t, identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED, owner.GetStatus())
	assert.Equal(t, "no key found", owner.GetError())
}

func TestIdentityGetIdentityStatus_OneClaimOnly(t *testing.T) {
	db := newFakeIdentityDB()
	db.claims["cid-v1/"+types.ClaimRoleOwner] = &gormdb.IdentityClaim{
		RecordCID: "cid-v1", Role: types.ClaimRoleOwner, Subject: "dns:acme.com", Status: types.ClaimStatusVerified,
	}

	resp, err := NewIdentityController(db).GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	require.NoError(t, err)
	assert.Nil(t, resp.GetIdentity())
	assert.NotNil(t, resp.GetOwner())
}

func TestIdentityGetIdentityStatus_ByName(t *testing.T) {
	db := newFakeIdentityDB()
	db.claims["cid-v2/"+types.ClaimRoleIdentity] = &gormdb.IdentityClaim{
		RecordCID: "cid-v2", Role: types.ClaimRoleIdentity, Subject: "latest", Status: types.ClaimStatusVerified,
	}
	db.claims["cid-v1/"+types.ClaimRoleIdentity] = &gormdb.IdentityClaim{
		RecordCID: "cid-v1", Role: types.ClaimRoleIdentity, Subject: "first", Status: types.ClaimStatusVerified,
	}

	ctrl := NewIdentityController(db)

	// Without a version, the latest one is used.
	resp, err := ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Name: new("acme.com/agent")})
	require.NoError(t, err)
	assert.Equal(t, "latest", resp.GetIdentity().GetSubject())

	resp, err = ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Name: new("acme.com/agent"), Version: new("1.0.0")})
	require.NoError(t, err)
	assert.Equal(t, "first", resp.GetIdentity().GetSubject())
}

func TestIdentityGetIdentityStatus_CIDWinsOverName(t *testing.T) {
	db := newFakeIdentityDB()
	db.claims["cid-other/"+types.ClaimRoleOwner] = &gormdb.IdentityClaim{
		RecordCID: "cid-other", Role: types.ClaimRoleOwner, Subject: "dns:other.com", Status: types.ClaimStatusVerified,
	}

	resp, err := NewIdentityController(db).GetIdentityStatus(context.Background(),
		&identityv1.GetIdentityStatusRequest{Cid: new("cid-other"), Name: new("acme.com/agent")})
	require.NoError(t, err)
	assert.Equal(t, "dns:other.com", resp.GetOwner().GetSubject())
}

func TestIdentityGetIdentityStatus_Errors(t *testing.T) {
	ctrl := NewIdentityController(newFakeIdentityDB())

	_, err := ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Name: new("missing.com/agent")})
	assert.Equal(t, codes.NotFound, status.Code(err))

	// A CID that is not a record is not "unverified".
	_, err = ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-missing")})
	assert.Equal(t, codes.NotFound, status.Code(err))

	_, err = ctrl.GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Name: new("acme.com/agent"), Version: new("9.9.9")})
	assert.Equal(t, codes.NotFound, status.Code(err))

	db := newFakeIdentityDB()
	db.claimErr = errors.New("database is down")

	_, err = NewIdentityController(db).GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	assert.Equal(t, codes.Internal, status.Code(err))

	db = newFakeIdentityDB()
	db.recordsErr = errors.New("database is down")

	_, err = NewIdentityController(db).GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	assert.Equal(t, codes.Internal, status.Code(err))
}

// The reason only describes a failure, whatever is stored with a verified result.
func TestIdentityGetIdentityStatus_VerifiedResultHasNoError(t *testing.T) {
	db := newFakeIdentityDB()
	db.claims["cid-v1/"+types.ClaimRoleOwner] = &gormdb.IdentityClaim{
		RecordCID: "cid-v1", Role: types.ClaimRoleOwner, Subject: "dns:acme.com", Status: types.ClaimStatusVerified, Error: "stale",
	}

	resp, err := NewIdentityController(db).GetIdentityStatus(context.Background(), &identityv1.GetIdentityStatusRequest{Cid: new("cid-v1")})
	require.NoError(t, err)
	assert.Nil(t, resp.GetOwner().Error)
}

func TestIdentityResolve(t *testing.T) {
	ctrl := NewIdentityController(newFakeIdentityDB())

	// All versions, newest first.
	resp, err := ctrl.Resolve(context.Background(), &identityv1.ResolveRequest{Name: "acme.com/agent"})
	require.NoError(t, err)
	require.Len(t, resp.GetRecords(), 2)
	assert.Equal(t, "cid-v2", resp.GetRecords()[0].GetCid())
	assert.Equal(t, "2.0.0", resp.GetRecords()[0].GetVersion())
	assert.Equal(t, "cid-v1", resp.GetRecords()[1].GetCid())
	assert.Equal(t, "acme.com/agent", resp.GetRecords()[1].GetName())

	// The specific version.
	resp, err = ctrl.Resolve(context.Background(), &identityv1.ResolveRequest{Name: "acme.com/agent", Version: new("1.0.0")})
	require.NoError(t, err)
	require.Len(t, resp.GetRecords(), 1)
	assert.Equal(t, "cid-v1", resp.GetRecords()[0].GetCid())
}

func TestIdentityResolve_Errors(t *testing.T) {
	ctrl := NewIdentityController(newFakeIdentityDB())

	_, err := ctrl.Resolve(context.Background(), &identityv1.ResolveRequest{})
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	_, err = ctrl.Resolve(context.Background(), &identityv1.ResolveRequest{Name: "missing.com/agent"})
	assert.Equal(t, codes.NotFound, status.Code(err))
}

//go:fix inline
