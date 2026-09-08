// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClaim satisfies types.ClaimObject.
type fakeClaim struct {
	subject, status, errMsg string
}

func (f fakeClaim) GetRecordCID() string      { return "record-cid" }
func (f fakeClaim) GetSubject() string        { return f.subject }
func (f fakeClaim) GetStatus() string         { return f.status }
func (f fakeClaim) GetError() string          { return f.errMsg }
func (f fakeClaim) GetVerifiedAt() *time.Time { return nil }

type fakeIdentityDB struct {
	types.DatabaseAPI

	identityClaim  types.ClaimObject
	identityErr    error
	ownershipClaim types.ClaimObject
	ownershipErr   error
	records        []coretypes.Record
}

func (f *fakeIdentityDB) GetClaimByCID(_, role string) (types.ClaimObject, error) {
	if role == types.ClaimRoleIdentity {
		return f.identityClaim, f.identityErr
	}

	return f.ownershipClaim, f.ownershipErr
}

func (f *fakeIdentityDB) GetRecords(...types.FilterOption) ([]coretypes.Record, error) {
	return f.records, nil
}

func TestGetIdentityStatus_BothClaimsPresent(t *testing.T) {
	db := &fakeIdentityDB{
		identityClaim:  fakeClaim{subject: "did:web:acme.com:agents:finance", status: gormdb.ClaimStatusVerified},
		ownershipClaim: fakeClaim{subject: "did:web:acme.com", status: gormdb.ClaimStatusFailed, errMsg: "boom"},
	}

	ctrl := NewIdentityController(db)

	resp, err := ctrl.GetIdentityStatus(t.Context(), &identityv1.GetIdentityStatusRequest{Cid: strPtr("cid-1")})
	require.NoError(t, err)

	require.NotNil(t, resp.GetIdentity())
	assert.True(t, resp.GetIdentity().GetVerified())
	assert.Equal(t, "did:web:acme.com:agents:finance", resp.GetIdentity().GetSubject())

	require.NotNil(t, resp.GetOwner())
	assert.False(t, resp.GetOwner().GetVerified())
	assert.Equal(t, "boom", resp.GetOwner().GetErrorMessage())
}

func TestGetIdentityStatus_NoClaimsPresent(t *testing.T) {
	db := &fakeIdentityDB{
		identityErr:  gormdb.ErrClaimNotFound,
		ownershipErr: gormdb.ErrClaimNotFound,
	}

	ctrl := NewIdentityController(db)

	resp, err := ctrl.GetIdentityStatus(t.Context(), &identityv1.GetIdentityStatusRequest{Cid: strPtr("cid-1")})
	require.NoError(t, err)
	assert.Nil(t, resp.GetIdentity())
	assert.Nil(t, resp.GetOwner())
}

func TestGetIdentityStatus_RequiresCidOrName(t *testing.T) {
	ctrl := NewIdentityController(&fakeIdentityDB{})

	_, err := ctrl.GetIdentityStatus(t.Context(), &identityv1.GetIdentityStatusRequest{})
	require.Error(t, err)
}

func strPtr(s string) *string { return &s }
