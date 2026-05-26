// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ownership implements the ownership claim reconciler task.
// It walks OwnershipClaim referrers from the store and keeps the DB owners table in sync.
package ownership

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ownershipv1 "github.com/agntcy/dir/api/ownership/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/ownership")

// Task reconciles ownership claims from OCI referrers into the search DB.
type Task struct {
	config       Config
	db           types.OwnershipDatabaseAPI
	store        types.ReferrerStoreAPI
	search       types.SearchDatabaseAPI
	trustedCerts []*x509.Certificate // nil when no CA cert configured (dev mode)
}

// NewTask creates a new ownership reconciler task.
func NewTask(config Config, db types.OwnershipDatabaseAPI, store types.ReferrerStoreAPI, search types.SearchDatabaseAPI) (*Task, error) {
	t := &Task{
		config: config,
		db:     db,
		store:  store,
		search: search,
	}

	if config.TrustedCACertFile != "" {
		certs, err := loadCACerts(config.TrustedCACertFile)
		if err != nil {
			return nil, fmt.Errorf("load trusted CA cert: %w", err)
		}

		t.trustedCerts = certs

		logger.Info("Ownership reconciler: signature verification enabled", "ca_cert_file", config.TrustedCACertFile)
	} else {
		logger.Warn("Ownership reconciler: no trusted_ca_cert_file configured — signed claims will have chain verification skipped, unsigned claims will be indexed without verification")
	}

	return t, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "ownership"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run walks all records and syncs their ownership referrers into the DB.
// This is the authoritative resync path: it compensates for any controller-side
// eager-write failures and keeps the owners search index consistent with the OCI store.
func (t *Task) Run(ctx context.Context) error {
	logger.Debug("Running ownership reconciliation")

	records, err := t.search.GetRecords()
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	for _, record := range records {
		if err := t.reconcileRecord(ctx, record.GetCid()); err != nil {
			logger.Error("Failed to reconcile ownership for record", "cid", record.GetCid(), "error", err)
		}
	}

	return nil
}

func (t *Task) reconcileRecord(ctx context.Context, cid string) error {
	if err := t.store.WalkReferrers(ctx, cid, corev1.OwnershipClaimReferrerType, func(ref *corev1.RecordReferrer) error {
		claim := &ownershipv1.Claim{}
		if err := claim.UnmarshalReferrer(ref); err != nil {
			logger.Warn("Failed to unmarshal ownership claim referrer", "cid", cid, "error", err)

			return nil
		}

		if err := t.verifyClaim(claim, cid); err != nil {
			// Reject the claim but continue walking other referrers.
			logger.Warn("Ownership claim rejected during reconciliation", "cid", cid, "owner_id", claim.GetOwnerId(), "error", err)

			return nil
		}

		if err := t.db.AddOwner(cid, claim.GetOwnerId(), claim.GetClaimedAt()); err != nil {
			return fmt.Errorf("add owner %s for %s: %w", claim.GetOwnerId(), cid, err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("walk ownership referrers for %s: %w", cid, err)
	}

	return nil
}

// verifyClaim enforces the signing policy:
//   - Signed claim: verify signature (and optionally the certificate chain).
//   - Unsigned claim: accept with a warning (dev/insecure mode).
func (t *Task) verifyClaim(claim *ownershipv1.Claim, cid string) error {
	if ownershipv1.IsSigned(claim) {
		if err := ownershipv1.VerifyClaim(claim, t.trustedCerts); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}

		logger.Debug("Ownership claim signature verified", "cid", cid, "owner_id", claim.GetOwnerId())

		return nil
	}

	// Unsigned claim — acceptable in environments without SPIFFE.
	logger.Debug("Indexing unsigned ownership claim (no SPIFFE signing configured)", "cid", cid, "owner_id", claim.GetOwnerId())

	return nil
}

// loadCACerts reads a PEM file and returns all certificate blocks found in it.
func loadCACerts(path string) ([]*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read CA cert file %q: %w", path, err)
	}

	var certs []*x509.Certificate

	for len(data) > 0 {
		var block *pem.Block

		block, data = pem.Decode(data)
		if block == nil {
			break
		}

		if block.Type != "CERTIFICATE" {
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse certificate in %q: %w", path, err)
		}

		certs = append(certs, cert)
	}

	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in %q", path)
	}

	return certs, nil
}
