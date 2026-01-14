// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package eventswrap provides an event-emitting wrapper for SigningAPI.
// It emits events for signing operations without modifying the underlying
// signing service implementation.
package eventswrap

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
)

// eventsSigning wraps a SigningAPI with event emission.
type eventsSigning struct {
	source   types.SigningAPI
	eventBus *events.SafeEventBus
}

// Wrap creates an event-emitting wrapper around a SigningAPI.
// All successful operations will emit corresponding events.
func Wrap(source types.SigningAPI, eventBus *events.SafeEventBus) types.SigningAPI {
	return &eventsSigning{
		source:   source,
		eventBus: eventBus,
	}
}

// Verify verifies a record signature and emits a RECORD_VERIFIED event.
func (s *eventsSigning) Verify(ctx context.Context, recordCID string) (bool, error) {
	verified, err := s.source.Verify(ctx, recordCID)
	if err != nil {
		return false, err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after verification
	s.eventBus.RecordVerified(recordCID, verified)

	return verified, nil
}

// PushSignature attaches a signature to a record and emits a RECORD_SIGNED event.
func (s *eventsSigning) PushSignature(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error {
	// Push signature to source
	err := s.source.PushSignature(ctx, recordCID, referrer)
	if err != nil {
		return err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after successful signature push
	s.eventBus.RecordSigned(recordCID, "client")

	return nil
}

// UploadPublicKey uploads a public key and emits a PUBLIC_KEY_UPLOADED event.
func (s *eventsSigning) UploadPublicKey(ctx context.Context, referrer *corev1.RecordReferrer) error {
	err := s.source.UploadPublicKey(ctx, referrer)
	if err != nil {
		return err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after successful public key upload
	s.eventBus.PublicKeyUploaded("public-key")

	return nil
}

// IsZotRegistry returns true if configured for a Zot registry.
func (s *eventsSigning) IsZotRegistry() bool {
	return s.source.IsZotRegistry()
}
