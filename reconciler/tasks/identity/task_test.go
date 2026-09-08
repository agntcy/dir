// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"testing"

	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRecord is the minimal coretypes.Record the task needs (GetCid, GetAnnotations).
type fakeRecord struct {
	coretypes.Record

	cid         string
	annotations map[string]string
}

func (f fakeRecord) GetCid() string                    { return f.cid }
func (f fakeRecord) GetAnnotations() map[string]string { return f.annotations }

// fakeDB implements types.DatabaseAPI via embedding; only the methods Run
// uses are overridden.
type fakeDB struct {
	types.DatabaseAPI

	records []coretypes.Record

	upsertedIdentity  []types.ClaimObject
	upsertedOwnership []types.ClaimObject
}

func (f *fakeDB) GetRecords(...types.FilterOption) ([]coretypes.Record, error) {
	return f.records, nil
}

func (f *fakeDB) UpsertClaim(role string, c types.ClaimObject) error {
	switch role {
	case types.ClaimRoleIdentity:
		f.upsertedIdentity = append(f.upsertedIdentity, c)
	case types.ClaimRoleOwner:
		f.upsertedOwnership = append(f.upsertedOwnership, c)
	}

	return nil
}

// fakeReferrerStore serves a fixed set of referrers back per record CID.
type fakeReferrerStore struct {
	byRecordAndType map[string][]*corev1.RecordReferrer
}

func (f *fakeReferrerStore) key(recordCID, referrerType string) string {
	return recordCID + "|" + referrerType
}

func (f *fakeReferrerStore) add(recordCID string, ref *corev1.RecordReferrer) {
	if f.byRecordAndType == nil {
		f.byRecordAndType = map[string][]*corev1.RecordReferrer{}
	}

	k := f.key(recordCID, ref.GetType())
	f.byRecordAndType[k] = append(f.byRecordAndType[k], ref)
}

func (f *fakeReferrerStore) WalkReferrers(_ context.Context, recordCID, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	for _, ref := range f.byRecordAndType[f.key(recordCID, referrerType)] {
		if err := walkFn(ref); err != nil {
			return err
		}
	}

	return nil
}

func (f *fakeReferrerStore) PushReferrer(context.Context, string, *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	return nil, nil //nolint:nilnil
}

func (f *fakeReferrerStore) DeleteReferrer(context.Context, string, string, string) ([]string, error) {
	return nil, nil
}

func (f *fakeReferrerStore) DeleteReferrers(context.Context, string, []string, string) ([]string, error) {
	return nil, nil
}

func TestTask_Run_VerifiesIdentityAndOwnershipClaims(t *testing.T) {
	const recordCID = "cid-1"

	identityClaim := identityv1.NewIdentityClaim("did:web:acme.com:agents:finance")
	identityClaim.SignedAt = "2026-09-08T12:00:00Z"
	identityClaim.Signature = "invalid-signature"

	identityRef, err := identityClaim.MarshalReferrer()
	require.NoError(t, err)

	store := &fakeReferrerStore{}
	store.add(recordCID, identityRef)

	db := &fakeDB{records: []coretypes.Record{fakeRecord{
		cid:         recordCID,
		annotations: map[string]string{corev1.AnnotationKeyIdentity: "did:web:acme.com:agents:finance"},
	}}}

	task, err := NewTask(Config{Enabled: true}, db, store, nil, nil)
	require.NoError(t, err)

	require.NoError(t, task.Run(t.Context()))

	require.Len(t, db.upsertedIdentity, 1)
	assert.Equal(t, recordCID, db.upsertedIdentity[0].GetRecordCID())
	// No resolver configured for a "did" subject -> verification fails, but the
	// result must still be recorded (not silently dropped).
	assert.Equal(t, "failed", db.upsertedIdentity[0].GetStatus())
	assert.NotEmpty(t, db.upsertedIdentity[0].GetError())

	assert.Empty(t, db.upsertedOwnership, "record has no ownership claim referrer")
}

func TestTask_Run_NoClaimReferrers_NoUpsert(t *testing.T) {
	const recordCID = "cid-no-claims"

	store := &fakeReferrerStore{}
	db := &fakeDB{records: []coretypes.Record{fakeRecord{cid: recordCID}}}

	task, err := NewTask(Config{Enabled: true}, db, store, nil, nil)
	require.NoError(t, err)

	require.NoError(t, task.Run(t.Context()))

	assert.Empty(t, db.upsertedIdentity)
	assert.Empty(t, db.upsertedOwnership)
}

func TestConfig_GetInterval_Default(t *testing.T) {
	var cfg Config
	assert.Equal(t, DefaultInterval, cfg.GetInterval())
}
