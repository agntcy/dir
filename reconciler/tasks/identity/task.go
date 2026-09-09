// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package identity implements the identity/ownership claim reconciler task.
// It periodically re-verifies IdentityClaim/OwnershipClaim referrers and
// keeps the search DB cache (identity_claims/ownership_claims) in sync. This
// complements the eager, best-effort verification done at ingest time
// (server/ingest) by re-checking claims whose external state (a DID document,
// a domain's JWKS, a DNS TXT record) may have changed since the claim was
// first pushed.
package identity

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	serveridentity "github.com/agntcy/dir/server/identity"
	"github.com/agntcy/dir/server/identity/spiffe"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/identity")

// Task reconciles identity/ownership claims from OCI referrers into the search DB.
type Task struct {
	config   Config
	db       types.DatabaseAPI
	store    types.ReferrerStoreAPI
	registry *serveridentity.Registry
	bundles  *spiffe.Bundles
}

// NewTask creates a new identity/ownership claim reconciler task.
func NewTask(config Config, db types.DatabaseAPI, store types.ReferrerStoreAPI, registry *serveridentity.Registry, bundles *spiffe.Bundles) (*Task, error) {
	return &Task{config: config, db: db, store: store, registry: registry, bundles: bundles}, nil
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

// Run walks all records and re-verifies their identity/ownership claim referrers.
func (t *Task) Run(ctx context.Context) error {
	records, err := t.db.GetRecords()
	if err != nil {
		return fmt.Errorf("get records: %w", err)
	}

	var identityChecked, ownershipChecked int

	for _, r := range records {
		cid := r.GetCid()

		found, err := t.reconcileIdentity(ctx, cid, r.GetAnnotations()[corev1.AnnotationKeyIdentity])
		if err != nil {
			logger.Warn("Failed to reconcile identity claim", "cid", cid, "error", err)
		} else if found {
			identityChecked++
		}

		found, err = t.reconcileOwnership(ctx, cid, r.GetAnnotations()[corev1.AnnotationKeyOwner])
		if err != nil {
			logger.Warn("Failed to reconcile ownership claim", "cid", cid, "error", err)
		} else if found {
			ownershipChecked++
		}
	}

	logger.Info("Identity reconciliation complete",
		"identity_checked", identityChecked, "ownership_checked", ownershipChecked, "total_records", len(records))

	return nil
}

// reconcileIdentity re-verifies recordCID's identity claim referrer, if any.
func (t *Task) reconcileIdentity(ctx context.Context, recordCID, expectedSubject string) (bool, error) {
	found := false

	err := t.store.WalkReferrers(ctx, recordCID, corev1.IdentityClaimReferrerType, func(ref *corev1.RecordReferrer) error {
		var claim identityv1.IdentityClaim
		if err := claim.UnmarshalReferrer(ref); err != nil {
			logger.Warn("Failed to unmarshal identity claim referrer", "cid", recordCID, "error", err)

			return nil
		}

		found = true

		result := identityv1.VerifyIdentityClaim(ctx, &claim, recordCID, expectedSubject, t.resolver(), t.trustedCertsFor(claim.GetSubject()))

		return t.db.UpsertClaim(types.ClaimRoleIdentity, claimResult{recordCID: recordCID, subject: claim.GetSubject(), result: result})
	})
	if err != nil {
		return found, fmt.Errorf("walk identity claim referrers: %w", err)
	}

	return found, nil
}

// reconcileOwnership re-verifies recordCID's ownership claim referrer, if any.
func (t *Task) reconcileOwnership(ctx context.Context, recordCID, expectedSubject string) (bool, error) {
	found := false

	err := t.store.WalkReferrers(ctx, recordCID, corev1.OwnershipClaimReferrerType, func(ref *corev1.RecordReferrer) error {
		var claim identityv1.OwnershipClaim
		if err := claim.UnmarshalReferrer(ref); err != nil {
			logger.Warn("Failed to unmarshal ownership claim referrer", "cid", recordCID, "error", err)

			return nil
		}

		found = true

		result := identityv1.VerifyOwnershipClaim(ctx, &claim, recordCID, expectedSubject, t.resolver(), t.trustedCertsFor(claim.GetSubject()))

		return t.db.UpsertClaim(types.ClaimRoleOwner, claimResult{recordCID: recordCID, subject: claim.GetSubject(), result: result})
	})
	if err != nil {
		return found, fmt.Errorf("walk ownership claim referrers: %w", err)
	}

	return found, nil
}

// resolver returns t.registry as a KeyResolver interface, or a true nil
// interface (not a non-nil interface wrapping a nil pointer) when unset.
func (t *Task) resolver() identityv1.KeyResolver {
	if t.registry == nil {
		return nil
	}

	return t.registry
}

// trustedCertsFor returns the configured SPIFFE trust bundle certificates for
// subject's trust domain, or nil if none are configured.
func (t *Task) trustedCertsFor(subject string) []*x509.Certificate {
	if t.bundles == nil {
		return nil
	}

	return t.bundles.TrustedCerts(subject)
}

// claimResult adapts an identityv1.Result to types.ClaimObject for DB upsert.
type claimResult struct {
	recordCID string
	subject   string
	result    *identityv1.Result
}

func (r claimResult) GetRecordCID() string { return r.recordCID }
func (r claimResult) GetSubject() string   { return r.subject }

func (r claimResult) GetStatus() string {
	if r.result.Verified {
		return "verified"
	}

	return "failed"
}

func (r claimResult) GetError() string { return r.result.Error }

func (r claimResult) GetVerifiedAt() *time.Time {
	if !r.result.Verified {
		return nil
	}

	now := time.Now()

	return &now
}
