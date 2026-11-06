// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package rpc

import (
	"bytes"
	"context"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/require"
	"github.com/ugorji/go/codec"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// referrerStore serves a single referrer that carries structured data, as a signature does.
type referrerStore struct {
	types.ReferrerStoreAPI

	referrer *corev1.RecordReferrer
}

func (s referrerStore) WalkReferrers(_ context.Context, _ string, _ string, walkFn func(*corev1.RecordReferrer) error) error {
	return walkFn(s.referrer)
}

// go-libp2p-gorpc encodes RPC messages with msgpack. A referrer's Data is a
// structpb.Struct, whose oneof msgpack cannot decode, so a PullReferrer
// response must survive the same codec round trip.
func TestPullReferrer_ResponseSurvivesMsgpack(t *testing.T) {
	t.Parallel()

	data, err := structpb.NewStruct(map[string]any{"algorithm": "ES256", "signature": "abc"})
	require.NoError(t, err)

	want := &corev1.RecordReferrer{
		Type:        corev1.SignatureReferrerType,
		ReferrerRef: &corev1.ReferrerRef{Cid: referrerCID},
		Data:        data,
	}

	api := &RPCAPI{service: &Service{
		store:    &peerStore{present: true},
		refStore: referrerStore{referrer: want},
		gate:     peerGate{servable: true},
	}}

	var out PullReferrerResponse

	require.NoError(t, api.PullReferrer(t.Context(), &PullReferrerRequest{
		RecordRef: &corev1.RecordRef{Cid: peerCID},
		Referrer:  &ReferrerDescriptor{Cid: referrerCID, Type: corev1.SignatureReferrerType},
	}, &out))

	var wire bytes.Buffer

	h := &codec.MsgpackHandle{}
	require.NoError(t, codec.NewEncoder(&wire, h).Encode(out))

	var got PullReferrerResponse
	require.NoError(t, codec.NewDecoder(&wire, h).Decode(&got))
	require.Equal(t, out, got)

	// The receiving node decodes what the serving node held.
	referrer, err := referrerFromResponse(&got)
	require.NoError(t, err)
	require.True(t, proto.Equal(want, referrer))
}
