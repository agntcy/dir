// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package eventswrap provides an event-emitting wrapper for StoreAPI.
// It emits events for all store operations (push, pull, delete) without
// modifying the underlying store implementation.
package eventswrap

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/events"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// eventsStore wraps a StoreAPI with event emission.
type eventsStore struct {
	source   types.StoreAPI
	eventBus *events.SafeEventBus
}

// Wrap creates an event-emitting wrapper around a StoreAPI.
// All successful operations will emit corresponding events.
func Wrap(source types.StoreAPI, eventBus *events.SafeEventBus) types.StoreAPI {
	return &eventsStore{
		source:   source,
		eventBus: eventBus,
	}
}

// Push pushes a record to the source store and emits a RECORD_PUSHED event.
func (s *eventsStore) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	// Push to source store
	ref, err := s.source.Push(ctx, record)
	if err != nil {
		return nil, err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after successful push
	labels := types.GetLabelsFromRecord(adapters.NewRecordAdapter(record))
	labelStrings := make([]string, len(labels))

	for i, label := range labels {
		labelStrings[i] = label.String()
	}

	s.eventBus.RecordPushed(ref.GetCid(), labelStrings)

	return ref, nil
}

// Pull pulls a record from the source store and emits a RECORD_PULLED event.
func (s *eventsStore) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	// Pull from source store
	record, err := s.source.Pull(ctx, ref)
	if err != nil {
		return nil, err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after successful pull
	labels := types.GetLabelsFromRecord(adapters.NewRecordAdapter(record))
	labelStrings := make([]string, len(labels))

	for i, label := range labels {
		labelStrings[i] = label.String()
	}

	s.eventBus.RecordPulled(ref.GetCid(), labelStrings)

	return record, nil
}

// Lookup forwards to the source store (no event emitted for metadata lookups).
func (s *eventsStore) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	//nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	return s.source.Lookup(ctx, ref)
}

// Delete deletes a record from the source store and emits a RECORD_DELETED event.
func (s *eventsStore) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	// Delete from source store
	err := s.source.Delete(ctx, ref)
	if err != nil {
		return err //nolint:wrapcheck // Transparent wrapper - pass through errors unchanged
	}

	// Emit event after successful deletion
	s.eventBus.RecordDeleted(ref.GetCid())

	return nil
}

// IsReady checks if the store is ready to serve traffic.
func (s *eventsStore) IsReady(ctx context.Context) bool {
	return s.source.IsReady(ctx)
}

// VerifyWithZot delegates to the source store if it supports Zot verification.
// This ensures the wrapper doesn't hide optional methods from the underlying store.
func (s *eventsStore) VerifyWithZot(ctx context.Context, recordCID string) (bool, error) {
	// Check if source supports Zot verification
	zotStore, ok := s.source.(types.VerifierStore)
	if !ok {
		// Source doesn't support it - this shouldn't happen with OCI store,
		// but handle gracefully
		return false, nil
	}

	// Delegate to source
	//nolint:wrapcheck
	return zotStore.VerifyWithZot(ctx, recordCID)
}

// PushReferrer delegates to the source store if it supports referrer operations.
// This is needed for signature and public key storage.
func (s *eventsStore) PushReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error {
	// Check if source supports referrer operations
	referrerStore, ok := s.source.(types.ReferrerStoreAPI)
	if !ok {
		return status.Errorf(codes.Unimplemented, "source store does not support referrer operations")
	}

	// Delegate to source (no event emitted for referrer operations)
	//nolint:wrapcheck
	return referrerStore.PushReferrer(ctx, recordCID, referrer)
}

// WalkReferrers delegates to the source store if it supports referrer operations.
// This is needed for retrieving signatures and public keys.
func (s *eventsStore) WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	// Check if source supports referrer operations
	referrerStore, ok := s.source.(types.ReferrerStoreAPI)
	if !ok {
		return status.Errorf(codes.Unimplemented, "source store does not support referrer operations")
	}

	// Delegate to source (no event emitted for referrer operations)
	//nolint:wrapcheck
	return referrerStore.WalkReferrers(ctx, recordCID, referrerType, walkFn)
}
