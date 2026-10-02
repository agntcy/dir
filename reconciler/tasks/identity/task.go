// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package identity implements the identity claim reconciler task. It is the only
// place identity and ownership claims are verified: on every run it looks up the
// current key material of each claim's subject, checks the claim's signature
// against it, and stores the outcome for IdentityService and the search filters.
// A rotated key, an expired certificate or a revoked trust bundle is therefore
// caught on the next run.
package identity

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	spifferesolver "github.com/agntcy/dir/client/utils/identity/resolvers/spiffe"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
)

var logger = logging.Logger("reconciler/identity")

const (
	// maxErrorLength caps the failure reason stored with a result.
	maxErrorLength = 1024

	// staleGrace is how long a result outlives lookups that keep failing for
	// reasons that say nothing about the claim, such as a subject being unreachable.
	staleGrace = 7 * 24 * time.Hour
)

// claimKind is one of the two kinds of claim a record can carry.
type claimKind struct {
	role         string               // stored role
	claimRole    identityv1.ClaimRole // role the claim itself must assert
	referrerType string               // referrer type the claims are stored under
	annotation   string               // record annotation declaring the expected subject
}

var claimKinds = []claimKind{
	{types.ClaimRoleIdentity, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, corev1.IdentityClaimReferrerType, corev1.AnnotationKeyIdentity},
	{types.ClaimRoleOwner, identityv1.ClaimRole_CLAIM_ROLE_OWNER, corev1.OwnershipClaimReferrerType, corev1.AnnotationKeyOwner},
}

// Task implements the identity claim verification task.
type Task struct {
	config   Config
	db       types.DatabaseAPI
	store    types.StoreAPI
	refStore types.ReferrerStoreAPI
	network  resolverSet
}

// NewTask creates a new identity claim verification task.
func NewTask(config Config, db types.DatabaseAPI, store types.StoreAPI, refStore types.ReferrerStoreAPI) (*Task, error) {
	return &Task{
		config:   config,
		db:       db,
		store:    store,
		refStore: refStore,
		network:  newNetworkResolvers(),
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "identity"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run verifies the claims of every record.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running identity claim verification")

	resolvers := t.network
	resolvers.spiffe = spifferesolver.New(t.loadTrustBundles())
	resolvers = resolvers.cached()

	// The CIDs alone are enough, and a fixed list does not shift under paging.
	cids, err := t.db.GetRecordCIDs()
	if err != nil {
		return fmt.Errorf("get record CIDs: %w", err)
	}

	var verified, failed int

	for _, cid := range cids {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("identity verification interrupted: %w", err)
		}

		v, f := t.reconcileRecord(ctx, resolvers, cid)
		verified += v
		failed += f
	}

	logger.Info("Identity claim verification complete", "verified", verified, "failed", failed)

	return nil
}

// loadTrustBundles reads the configured SPIFFE trust bundles. One that cannot be
// read is left out, so only its own trust domain fails closed, not the others and
// not the run.
func (t *Task) loadTrustBundles() x509bundle.Source {
	bundles, err := spifferesolver.LoadBundles(t.config.trustDomains())
	if err != nil {
		logger.Error("Some SPIFFE trust bundles could not be loaded; their spiffe:// claims will not verify", "error", err)
	}

	return bundles
}

// reconcileRecord verifies the claims of one record and returns how many of its
// results are verified and failed. A claim it cannot read leaves the stored
// result as it was.
func (t *Task) reconcileRecord(ctx context.Context, resolvers resolverSet, cid string) (int, int) {
	ctx, cancel := context.WithTimeout(ctx, t.config.GetRecordTimeout())
	defer cancel()

	claims, err := t.claims(ctx, cid)
	if err != nil {
		logger.Warn("Failed to read claims", "cid", cid, "error", err)

		return 0, 0
	}

	if len(claims) == 0 {
		for _, kind := range claimKinds {
			t.dropResult(cid, kind.role)
		}

		return 0, 0
	}

	// The record is only read once it is known to carry a claim.
	annotations, err := t.annotationsOf(ctx, cid)
	if err != nil {
		logger.Warn("Failed to read record annotations", "cid", cid, "error", err)

		return 0, 0
	}

	var verified, failed int

	for _, kind := range claimKinds {
		result, transient := t.verify(ctx, resolvers, cid, annotations[kind.annotation], claims[kind.role])
		if result == nil {
			t.dropResult(cid, kind.role)

			continue
		}

		result.Role = kind.role

		// An unreachable subject does not say the claim is wrong, so the last result stands for a while.
		if transient && t.previous(cid, kind.role, staleGrace) != nil {
			logger.Warn("Keeping the last claim result: the subject could not be looked up", "cid", cid, "role", kind.role, "error", result.Error)

			continue
		}

		if err := t.db.UpsertIdentityClaim(result); err != nil {
			logger.Warn("Failed to store claim result", "cid", cid, "role", kind.role, "error", err)

			continue
		}

		if result.Status == types.ClaimStatusVerified {
			verified++

			logger.Info("Claim verified", "cid", cid, "role", kind.role, "subject", result.Subject)
		} else {
			failed++

			logger.Info("Claim failed verification", "cid", cid, "role", kind.role, "subject", result.Subject, "error", result.Error)
		}
	}

	return verified, failed
}

// claims returns the claims of every kind attached to a record, by role, from
// one walk. The store's type filter maps all custom types onto a single OCI media
// type, so asking per kind would re-list and re-fetch the whole referrer set each
// time; the type is checked here instead.
func (t *Task) claims(ctx context.Context, cid string) (map[string][]*identityv1.Claim, error) {
	byRole := make(map[string][]*identityv1.Claim, len(claimKinds))

	err := t.refStore.WalkReferrers(ctx, cid, "", func(ref *corev1.RecordReferrer) error {
		for _, kind := range claimKinds {
			if ref.GetType() != kind.referrerType {
				continue
			}

			claim := &identityv1.Claim{}
			if err := claim.UnmarshalReferrer(ref); err != nil {
				logger.Debug("Skipping unparsable claim referrer", "cid", cid, "error", err)

				return nil //nolint:nilerr // one bad referrer must not hide the others
			}

			// A claim signs the role it asserts. One stored under the other kind's
			// referrer type would otherwise verify against that kind's annotation.
			if claim.GetRole() != kind.claimRole {
				logger.Debug("Skipping claim stored under the wrong referrer type", "cid", cid, "role", claim.GetRole())

				return nil
			}

			byRole[kind.role] = append(byRole[kind.role], claim)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk referrers: %w", err)
	}

	return byRole, nil
}

// annotationsOf returns the annotations of a record, read from the store because
// the database does not hold them with the record.
func (t *Task) annotationsOf(ctx context.Context, cid string) (map[string]string, error) {
	record, err := t.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return nil, fmt.Errorf("pull record: %w", err)
	}

	data, err := record.Decode()
	if err != nil {
		return nil, fmt.Errorf("decode record: %w", err)
	}

	annotations := data.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}

	return annotations, nil
}

// verify checks the claims of one kind that name the subject the record declares,
// and returns the result to store, or nil when none of them does. A claim that
// verifies decides it, whoever else attached claims to the record. Otherwise the
// newest claim's failure is the result, and transient says whether any claim failed
// only because its subject could not be looked up.
func (t *Task) verify(ctx context.Context, resolvers resolverSet, cid, expected string, claims []*identityv1.Claim) (*gormdb.IdentityClaim, bool) {
	var (
		result    *gormdb.IdentityClaim
		transient bool
		failedAt  time.Time
	)

	for _, claim := range claims {
		// Anyone can attach a claim to any record. One that names a subject the
		// record does not declare says nothing about the record, so it leaves no
		// result: its subject would otherwise be stored and matched by --identity/--owner.
		if expected == "" || claim.GetSubject() != expected {
			continue
		}

		err := t.verifyClaim(ctx, resolvers, cid, expected, claim)
		if err == nil {
			return &gormdb.IdentityClaim{
				RecordCID:  cid,
				Subject:    expected,
				Status:     types.ClaimStatusVerified,
				VerifiedAt: time.Now(),
			}, false
		}

		transient = transient || isTransient(err)

		// Whoever signs a claim writes its signed_at, so one that does not parse, or
		// is dated in the future, must not outrank an honest claim.
		signedAt, parseErr := time.Parse(time.RFC3339, claim.GetSignedAt())
		if parseErr != nil || signedAt.After(time.Now()) {
			signedAt = time.Time{}
		}

		if result == nil || signedAt.After(failedAt) {
			failedAt = signedAt
			result = &gormdb.IdentityClaim{
				RecordCID:  cid,
				Subject:    expected,
				Status:     types.ClaimStatusFailed,
				Error:      truncate(err.Error()),
				VerifiedAt: time.Now(),
			}
		}
	}

	return result, transient
}

// verifyClaim looks up the current keys of the claim's subject and verifies the
// claim against them. The checks that need no key come first, so a claim that
// cannot verify costs no lookup.
func (t *Task) verifyClaim(ctx context.Context, resolvers resolverSet, cid, expected string, claim *identityv1.Claim) error {
	if err := clientidentity.Check(claim, cid, expected); err != nil {
		return fmt.Errorf("check claim: %w", err)
	}

	resolver, err := resolvers.forSubject(claim.GetSubject())
	if err != nil {
		return err
	}

	var certificate []byte

	if claim.GetCertificate() != "" {
		certificate, err = base64.StdEncoding.DecodeString(claim.GetCertificate())
		if err != nil {
			return fmt.Errorf("decode claim certificate: %w", err)
		}
	}

	keys, err := resolver.Resolve(ctx, claim.GetSubject(), certificate)
	if err != nil {
		return fmt.Errorf("resolve keys of %s: %w", claim.GetSubject(), err)
	}

	if _, err := clientidentity.Verify(claim, cid, expected, keys...); err != nil {
		return fmt.Errorf("verify claim: %w", err)
	}

	return nil
}

// dropResult removes the stored result of a claim that is no longer attached to
// its record, so the record does not keep a verified status for a claim it no
// longer has.
func (t *Task) dropResult(cid, role string) {
	// Most records have no claims, so check before writing.
	if t.previous(cid, role, 0) == nil {
		return
	}

	if err := t.db.DeleteIdentityClaim(cid, role); err != nil {
		logger.Warn("Failed to remove stale claim result", "cid", cid, "role", role, "error", err)
	}
}

// previous returns the stored result of a record's claim, or nil if there is none
// or it was last checked more than maxAge ago (0 means any age).
func (t *Task) previous(cid, role string, maxAge time.Duration) types.IdentityClaimObject {
	prev, err := t.db.GetIdentityClaimByCID(cid, role)
	if err != nil || (maxAge > 0 && time.Since(prev.GetVerifiedAt()) > maxAge) {
		return nil
	}

	return prev
}

func truncate(msg string) string {
	if len(msg) <= maxErrorLength {
		return msg
	}

	return msg[:maxErrorLength]
}
