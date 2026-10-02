// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// The fakes return the store's statuses unwrapped: the tests compare codes and
// messages exactly.

//nolint:wrapcheck
package rpc

import (
	"context"
	"errors"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	peerCID     = "baeareidp4vt6jw7tirdvk6qlcuqndobz24yejxovcjgcuv3qnnhwzqz4mi"
	referrerCID = "baeareig7ja4xoyfuv3y4gltbxogsqgiteneu4uozhoz3tuaxfidzmonyem"
)

// peerStore holds at most the record peerCID and one referrer, and answers a
// missing record the way the OCI store does.
type peerStore struct {
	types.StoreAPI
	types.ReferrerStoreAPI

	present bool
	reads   int
}

func (s *peerStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	s.reads++

	if !s.present {
		return nil, types.RecordNotFoundError(ref.GetCid())
	}

	return &corev1.RecordMeta{Cid: ref.GetCid()}, nil
}

func (s *peerStore) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	s.reads++

	return &corev1.Record{}, nil
}

func (s *peerStore) WalkReferrers(_ context.Context, recordCID string, _ string, walkFn func(*corev1.RecordReferrer) error) error {
	s.reads++

	if !s.present {
		return types.RecordNotFoundError(recordCID)
	}

	return walkFn(&corev1.RecordReferrer{
		Type:        corev1.SignatureReferrerType,
		ReferrerRef: &corev1.ReferrerRef{Cid: referrerCID},
	})
}

// peerGate answers the policy gate.
type peerGate struct {
	servable bool
	err      error
}

func (g peerGate) IsRecordServable(string) (bool, error) { return g.servable, g.err }

func newAPI(store *peerStore, gate peerGate) *RPCAPI {
	return &RPCAPI{service: &Service{store: store, refStore: store, servability: gate}}
}

// peerCalls runs each RPC a remote peer can make for peerCID once.
var peerCalls = map[string]func(ctx context.Context, api *RPCAPI) error{
	"Lookup": func(ctx context.Context, api *RPCAPI) error {
		return api.Lookup(ctx, &corev1.RecordRef{Cid: peerCID}, &LookupResponse{})
	},
	"Pull": func(ctx context.Context, api *RPCAPI) error {
		return api.Pull(ctx, &corev1.RecordRef{Cid: peerCID}, &PullResponse{})
	},
	"ListReferrers": func(ctx context.Context, api *RPCAPI) error {
		return api.ListReferrers(ctx, &corev1.RecordRef{Cid: peerCID}, &ListReferrersResponse{})
	},
	"PullReferrer": func(ctx context.Context, api *RPCAPI) error {
		return api.PullReferrer(ctx, &PullReferrerRequest{
			RecordRef: &corev1.RecordRef{Cid: peerCID},
			Referrer:  &ReferrerDescriptor{Cid: referrerCID, Type: corev1.SignatureReferrerType},
		}, &PullReferrerResponse{})
	},
}

// A peer is told a record the gate excludes is not available under this
// node's content policy. A record that passes the gate but is missing from the
// store is still not found. Both runs use the same CID: one held but excluded,
// one let through by the gate but not in the store.
func TestPeerRPC_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	for name, call := range peerCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			exStore := &peerStore{present: true}
			exErr := call(t.Context(), newAPI(exStore, peerGate{servable: false}))

			missErr := call(t.Context(), newAPI(&peerStore{present: false}, peerGate{servable: true}))

			require.Error(t, exErr)
			require.Error(t, missErr)
			assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
			assert.Contains(t, status.Convert(exErr).Message(), status.Convert(types.RecordExcludedError(peerCID)).Message(), "no policy or reason in it")
			assert.Equal(t, codes.NotFound, status.Code(missErr), "a record that passes the gate but is not in the store is still not found")
			assert.Zero(t, exStore.reads, "the store must not be read for an excluded record")
		})
	}
}

func TestPeerRPC_ServableRecordIsServed(t *testing.T) {
	t.Parallel()

	for name, call := range peerCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			require.NoError(t, call(t.Context(), newAPI(&peerStore{present: true}, peerGate{servable: true})))
		})
	}
}

// When the gate cannot answer, nothing is served to the peer.
func TestPeerRPC_GateErrorFailsClosed(t *testing.T) {
	t.Parallel()

	for name, call := range peerCalls {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			store := &peerStore{present: true}
			err := call(t.Context(), newAPI(store, peerGate{err: errors.New("database unavailable")}))

			assert.Equal(t, codes.Internal, status.Code(err))
			assert.Zero(t, store.reads)
		})
	}
}

// The service cannot be built without the gate, so a caller cannot forget it.
func TestNew_RequiresServability(t *testing.T) {
	t.Parallel()

	_, err := New(nil, &peerStore{}, nil)
	require.ErrorContains(t, err, "servability check is required")
}
