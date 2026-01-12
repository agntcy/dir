// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package verificationwrap provides a store wrapper that adds domain verification
// status to record metadata responses.
package verificationwrap

import (
	"context"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storeconfig "github.com/agntcy/dir/server/store/config"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/verification"
	"github.com/agntcy/dir/utils/logging"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

var logger = logging.Logger("store/verificationwrap")

// Store wraps a store implementation to add domain verification to Lookup responses.
type Store struct {
	source   types.StoreAPI
	verifier *verification.Verifier
	enabled  bool
}

// Wrap creates a verification-aware store wrapper.
// If verification is disabled in config, returns the source store unchanged.
func Wrap(source types.StoreAPI, cfg storeconfig.VerificationConfig) types.StoreAPI {
	if !cfg.Enabled {
		logger.Info("Domain verification disabled")
		return source
	}

	// Determine cache TTL
	cacheTTL := cfg.CacheTTL
	if cacheTTL == 0 {
		cacheTTL = storeconfig.DefaultVerificationCacheTTL
	}

	verifier := verification.NewVerifier(
		verification.WithCacheTTL(cacheTTL),
		verification.WithAllowInsecureWellKnown(cfg.AllowInsecure),
	)

	logger.Info("Domain verification enabled",
		"cache_ttl", cacheTTL,
		"allow_insecure", cfg.AllowInsecure)

	return &Store{
		source:   source,
		verifier: verifier,
		enabled:  true,
	}
}

// Push delegates to the underlying store.
func (s *Store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	return s.source.Push(ctx, record)
}

// Pull delegates to the underlying store.
func (s *Store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	return s.source.Pull(ctx, ref)
}

// Lookup retrieves metadata and adds verification status.
func (s *Store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	meta, err := s.source.Lookup(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Add verification status
	s.addVerificationStatus(ctx, ref.GetCid(), meta)

	return meta, nil
}

// Delete delegates to the underlying store.
func (s *Store) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	return s.source.Delete(ctx, ref)
}

// IsReady delegates to the underlying store.
func (s *Store) IsReady(ctx context.Context) bool {
	return s.source.IsReady(ctx)
}

// VerifyWithZot delegates to the source store if it supports Zot verification.
func (s *Store) VerifyWithZot(ctx context.Context, recordCID string) (bool, error) {
	zotStore, ok := s.source.(types.VerifierStore)
	if !ok {
		return false, nil
	}

	//nolint:wrapcheck
	return zotStore.VerifyWithZot(ctx, recordCID)
}

// PushReferrer delegates to the source store if it supports referrer operations.
func (s *Store) PushReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error {
	referrerStore, ok := s.source.(types.ReferrerStoreAPI)
	if !ok {
		return nil
	}

	//nolint:wrapcheck
	return referrerStore.PushReferrer(ctx, recordCID, referrer)
}

// WalkReferrers delegates to the source store if it supports referrer operations.
func (s *Store) WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	referrerStore, ok := s.source.(types.ReferrerStoreAPI)
	if !ok {
		return nil
	}

	//nolint:wrapcheck
	return referrerStore.WalkReferrers(ctx, recordCID, referrerType, walkFn)
}

// addVerificationStatus performs domain verification and adds results to metadata annotations.
func (s *Store) addVerificationStatus(ctx context.Context, cid string, meta *corev1.RecordMeta) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}

	// Get record name from existing annotations
	recordName := meta.Annotations[oci.MetadataKeyName]
	if recordName == "" {
		logger.Debug("No record name found, skipping verification", "cid", cid)
		meta.Annotations[oci.MetadataKeyDomainVerified] = "false"
		meta.Annotations[oci.MetadataKeyDomainVerifyError] = "no record name found"

		return
	}

	// Extract domain from name
	domain := verification.ExtractDomain(recordName)
	if domain == "" {
		logger.Debug("Could not extract domain from record name", "cid", cid, "name", recordName)
		meta.Annotations[oci.MetadataKeyDomainVerified] = "false"
		meta.Annotations[oci.MetadataKeyDomainVerifyError] = "could not extract domain from name"

		return
	}

	// Get public key for this record
	publicKey, err := s.getRecordPublicKey(ctx, cid)
	if err != nil {
		logger.Debug("Could not get public key for record", "cid", cid, "error", err)
		meta.Annotations[oci.MetadataKeyDomainVerified] = "false"
		meta.Annotations[oci.MetadataKeyDomainVerifyError] = "no public key found for record"
		meta.Annotations[oci.MetadataKeyDomainVerifiedDomain] = domain

		return
	}

	// Verify domain ownership
	result := s.verifier.Verify(ctx, recordName, publicKey)

	// Add verification results to annotations
	if result.Verified {
		meta.Annotations[oci.MetadataKeyDomainVerified] = "true"
		meta.Annotations[oci.MetadataKeyDomainVerifiedAt] = result.VerifiedAt.Format(time.RFC3339)
		meta.Annotations[oci.MetadataKeyDomainVerifyMethod] = result.Method
		meta.Annotations[oci.MetadataKeyDomainVerifiedDomain] = result.Domain
	} else {
		meta.Annotations[oci.MetadataKeyDomainVerified] = "false"

		if result.Error != "" {
			meta.Annotations[oci.MetadataKeyDomainVerifyError] = result.Error
		}

		if result.Domain != "" {
			meta.Annotations[oci.MetadataKeyDomainVerifiedDomain] = result.Domain
		}
	}

	logger.Debug("Domain verification completed",
		"cid", cid,
		"domain", domain,
		"verified", result.Verified,
		"method", result.Method)
}

// getRecordPublicKey retrieves the public key associated with a record.
// The public key is stored as a PEM-encoded string and is converted to DER bytes.
func (s *Store) getRecordPublicKey(ctx context.Context, cid string) ([]byte, error) {
	referrerStore, ok := s.source.(types.ReferrerStoreAPI)
	if !ok {
		return nil, errReferrerStoreNotSupported
	}

	var publicKey []byte

	// Walk public key referrers to find the key
	err := referrerStore.WalkReferrers(ctx, cid, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)
			return nil // Continue walking
		}

		// The public key is stored as PEM-encoded string
		pemKey := pk.GetKey()
		if pemKey == "" {
			logger.Debug("Empty public key")
			return nil // Continue walking
		}

		// Parse the PEM-encoded public key to get the actual key
		parsedKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pemKey))
		if err != nil {
			logger.Debug("Failed to parse PEM public key", "error", err)
			return nil // Continue walking
		}

		// Marshal the key to DER format for comparison
		keyBytes, err := cryptoutils.MarshalPublicKeyToDER(parsedKey)
		if err != nil {
			logger.Debug("Failed to marshal public key to DER", "error", err)
			return nil // Continue walking
		}

		publicKey = keyBytes

		return errStopWalk // Stop after finding the first key
	})

	if err != nil && err != errStopWalk {
		return nil, err
	}

	if publicKey == nil {
		return nil, errNoPublicKey
	}

	return publicKey, nil
}

// Sentinel errors for internal use.
var (
	errReferrerStoreNotSupported = &verificationError{msg: "referrer store not supported"}
	errNoPublicKey               = &verificationError{msg: "no public key found"}
	errStopWalk                  = &verificationError{msg: "stop walking"}
)

type verificationError struct {
	msg string
}

func (e *verificationError) Error() string {
	return e.msg
}
