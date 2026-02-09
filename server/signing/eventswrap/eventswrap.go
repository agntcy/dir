// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package eventswrap provides an event-emitting wrapper for SigningAPI.
// It emits events for signing operations without modifying the underlying
// signing service implementation.
package eventswrap

import (
	"context"

	signv1 "github.com/agntcy/dir/api/sign/v1"
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

// Verify verifies record signatures and emits a RECORD_VERIFIED event.
func (s *eventsSigning) Verify(ctx context.Context, recordCID string, options *signv1.VerifyOptions) (*types.VerifyResult, error) {
	result, err := s.source.Verify(ctx, recordCID, options)
	if err != nil {
		return nil, err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after verification
	s.eventBus.RecordVerified(recordCID, result.Success)

	return result, nil
}
