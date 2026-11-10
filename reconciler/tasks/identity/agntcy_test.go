// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	agntcyresolver "github.com/agntcy/dir/client/utils/identity/resolvers/agntcy"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type claimAuthority struct {
	keys  []crypto.PublicKey
	until time.Time
	cids  []string
	err   error
}

func (e *claimAuthority) Resolve(_ context.Context, _ string, _ []byte, opts ...resolvers.Option) ([]crypto.PublicKey, error) {
	options := resolvers.ApplyOptions(opts...)

	e.cids = append(e.cids, options.Claim.GetRecordCid())
	if options.ValidUntil != nil {
		*options.ValidUntil = e.until
	}

	return e.keys, e.err
}

func TestAGNTCYReconciliationAndSearch(t *testing.T) {
	f := newFixture(t, Config{Enabled: true})
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, wrongKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	subject := "agntcy://Agent-One"
	badgeDeadline := time.Now().Add(10 * time.Minute)
	authority := &claimAuthority{keys: []crypto.PublicKey{pub}, until: badgeDeadline}
	f.task.network = resolverSet{clientidentity.SchemeAGNTCY: {resolver: authority, grace: time.Hour, claimAware: true}}
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

func TestAGNTCYRequiresConfiguredResolver(t *testing.T) {
	f := newFixture(t, Config{Enabled: true})
	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	subject := "agntcy://Agent"
	cid := f.addRecord("missing-verifier", map[string]string{corev1.AnnotationKeyIdentity: subject})
	f.store.setReferrers(cid, signedReferrer(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, cid, subject, newSigner(t, key)))
	f.run()
	assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleIdentity).GetStatus())
	assert.Contains(t, f.result(cid, types.ClaimRoleIdentity).GetError(), "unsupported")
}

func TestAGNTCYScheduleValidation(t *testing.T) {
	for _, interval := range []time.Duration{0, -time.Second, 30 * time.Minute, time.Hour} {
		_, err := NewTask(Config{Interval: interval, AGNTCY: agntcyresolver.Config{VerifierURL: "https://verifier.example"}}, nil, nil, nil)
		require.ErrorContains(t, err, "identity.interval")
	}

	cfg := Config{Interval: 15 * time.Minute, AGNTCY: agntcyresolver.Config{VerifierURL: "https://verifier.example"}}
	require.NoError(t, cfg.validateAGNTCY())
	cfg.AGNTCY.MaxVerificationAge = 10 * time.Minute
	require.Error(t, cfg.validateAGNTCY())
	cfg.AGNTCY.VerifierURL = ""
	require.NoError(t, cfg.validateAGNTCY(), "unconfigured AGNTCY does not change other schemes")
}

func TestAGNTCYReloadFailureStopsRun(t *testing.T) {
	f := newFixture(t, Config{Enabled: true})
	cid := f.addRecord("reload-failure", nil)
	until := time.Now().Add(time.Hour)
	require.NoError(t, f.db.UpsertIdentityClaim(&gormdb.IdentityClaim{RecordCID: cid, Role: types.ClaimRoleIdentity, Subject: "agntcy://Agent", Status: types.ClaimStatusVerified, ValidUntil: &until}))
	f.task.config.AGNTCY = agntcyresolver.Config{VerifierURL: "https://verifier.example", VerifierTrustBundleFile: t.TempDir() + "/missing.pem"}
	require.ErrorContains(t, f.task.Run(t.Context()), "reload AGNTCY")
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus())
	require.NoError(t, os.WriteFile(f.task.config.AGNTCY.VerifierTrustBundleFile, []byte("invalid key"), 0o600))
	require.ErrorContains(t, f.task.Run(t.Context()), "reload AGNTCY")
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus())
}
