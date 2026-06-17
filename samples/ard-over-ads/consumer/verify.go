// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package consumer

import (
	"context"
	"fmt"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	adscorev1 "github.com/agntcy/dir/api/core/v1"
	adssignv1 "github.com/agntcy/dir/api/sign/v1"
	adsclient "github.com/agntcy/dir/client"
)

// Verify confirms the entry owns its claimed identity and trust signals.
// Identifier format is expected to be urn:ai:org.agntcy:cid:<cid>.
func Verify(ctx context.Context, client *adsclient.Client, e *catalogv1.CatalogEntry) error {
	// Get AGNTYCY identifier
	cid, found := strings.CutPrefix(e.GetIdentifier(), "urn:ai:org.agntcy:cid:")
	if !found {
		return fmt.Errorf("invalid identifier format: %s", e.GetIdentifier())
	}

	// Verify provenance
	if verified, err := client.Verify(ctx, &adssignv1.VerifyRequest{
		RecordRef: &adscorev1.RecordRef{Cid: cid},
	}); err != nil {
		return fmt.Errorf("failed to verify provenance: %w", err)
	} else if !verified.GetSuccess() {
		return fmt.Errorf("provenance verification failed: %s", verified.GetErrorMessage())
	}

	// Verify naming
	if verified, err := client.GetVerificationInfo(ctx, cid); err != nil {
		return fmt.Errorf("failed to get verification info: %w", err)
	} else if !verified.GetVerified() {
		// we do not fail, just log as warning
		fmt.Printf("Warning: identity verification failed for CID %s: %+v\n", e.Identifier, verified)
	}

	return nil
}
