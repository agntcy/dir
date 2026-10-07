// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type expiringAgentResolver struct {
	keys  []crypto.PublicKey
	until time.Time
	err   error
	calls int
}

func (r *expiringAgentResolver) Resolve(context.Context, string, []byte) ([]crypto.PublicKey, error) {
	r.calls++

	return r.keys, r.err
}
func (r *expiringAgentResolver) ResolutionValidUntil(string, []byte) time.Time { return r.until }

type claimAuthority struct {
	keys  []crypto.PublicKey
	until time.Time
	cids  []string
	err   error
}

func (e *claimAuthority) ResolveClaim(_ context.Context, claim *identityv1.Claim) (clientidentity.ClaimResolution, error) {
	e.cids = append(e.cids, claim.GetRecordCid())

	return clientidentity.ClaimResolution{PublicKeys: e.keys, ValidUntil: e.until}, e.err
}

func TestAGNTCYReconciliationAndSearch(t *testing.T) {
	f := newFixture(t, Config{Enabled: true})
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, wrongKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	subject := "agntcy://Agent-One"
	keyDeadline := time.Now().Add(time.Hour)
	badgeDeadline := time.Now().Add(10 * time.Minute)
	resolver := &expiringAgentResolver{keys: []crypto.PublicKey{pub}, until: keyDeadline}
	authority := &claimAuthority{keys: []crypto.PublicKey{pub}, until: badgeDeadline}
	f.task.network = resolverSet{agntcy: resolver}
	f.task.claimResolver = authority
	first := f.addRecord("agntcy-first", map[string]string{corev1.AnnotationKeyIdentity: subject})
	second := f.addRecord("agntcy-second", map[string]string{corev1.AnnotationKeyIdentity: subject})
	valid := signedReferrer(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, first, subject, newSigner(t, key))
	invalid := signedReferrer(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, first, subject, newSigner(t, wrongKey))
	f.store.setReferrers(first, invalid, valid)
	// Replay the first record's claim against another version of the same ID.
	f.store.setReferrers(second, valid)
	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(first, types.ClaimRoleIdentity).GetStatus())
	assert.Equal(t, types.ClaimStatusFailed, f.result(second, types.ClaimRoleIdentity).GetStatus())
	assert.Equal(t, 0, resolver.calls, "combined path must not make a preliminary key lookup")
	assert.Equal(t, []string{first, first}, authority.cids, "invalid and valid candidate claims each obtain evidence; Directory rejects the invalid signature locally")
	observation, ok := f.result(first, types.ClaimRoleIdentity).(types.IdentityClaimValidity)
	require.True(t, ok)

	until := observation.GetValidUntil()
	assert.True(t, until.Equal(badgeDeadline))

	cids, err := f.db.GetRecordCIDs(types.WithIdentities("agntcy://*"), types.WithIdentityVerified())
	require.NoError(t, err)
	assert.Equal(t, []string{first}, cids)

	// A temporary failure preserves the result only within its effective deadline.
	verifiedAt := f.result(first, types.ClaimRoleIdentity).GetVerifiedAt()
	authority.err = context.DeadlineExceeded

	f.run()
	assert.True(t, f.result(first, types.ClaimRoleIdentity).GetVerifiedAt().Equal(verifiedAt))

	expired := time.Now().Add(-time.Minute)
	require.NoError(t, f.db.UpsertIdentityClaim(&gormdb.IdentityClaim{RecordCID: first, Role: types.ClaimRoleIdentity, Subject: subject, Status: types.ClaimStatusVerified, VerifiedAt: verifiedAt, ValidUntil: &expired}))
	f.run()
	assert.Equal(t, types.ClaimStatusFailed, f.result(first, types.ClaimRoleIdentity).GetStatus())
	cids, err = f.db.GetRecordCIDs(types.WithIdentities("agntcy://*"), types.WithIdentityVerified())
	require.NoError(t, err)
	assert.Empty(t, cids)
	f.store.setReferrers(first)
	f.run()
	f.noResult(first, types.ClaimRoleIdentity)
}

func TestAGNTCYRequiresRegisteredClaimResolver(t *testing.T) {
	f := newFixture(t, Config{Enabled: true})
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	subject := "agntcy://Agent"
	cid := f.addRecord("missing-verifier", map[string]string{corev1.AnnotationKeyIdentity: subject})
	f.store.setReferrers(cid, signedReferrer(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, cid, subject, newSigner(t, key)))
	f.task.network = resolverSet{agntcy: &expiringAgentResolver{keys: []crypto.PublicKey{pub}, until: time.Now().Add(time.Hour)}}
	f.run()
	assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleIdentity).GetStatus())
	assert.Contains(t, f.result(cid, types.ClaimRoleIdentity).GetError(), "not registered")
}

func TestCachedResolutionRechecksExpiredKeys(t *testing.T) {
	r := &expiringAgentResolver{until: time.Now().Add(time.Hour)}
	set := (resolverSet{agntcy: r}).cached()
	ctx := context.Background()
	_, err := set.agntcy.Resolve(ctx, "agntcy://Agent", nil)
	require.NoError(t, err)
	_, err = set.agntcy.Resolve(ctx, "agntcy://Agent", nil)
	require.NoError(t, err)
	assert.Equal(t, 1, r.calls)
	r.until = time.Now().Add(-time.Second)
	_, err = set.agntcy.Resolve(ctx, "agntcy://Agent", nil)
	require.NoError(t, err)
	assert.Equal(t, 2, r.calls)
}
