// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"io"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// gatedCID is a valid CID: PullReferrer validates it.
const gatedCID = "baeareidp4vt6jw7tirdvk6qlcuqndobz24yejxovcjgcuv3qnnhwzqz4mi"

// gatedStore holds at most the record gatedCID and its referrers, and
// answers a missing record the way the OCI store does.
type gatedStore struct {
	types.StoreAPI
	types.ReferrerStoreAPI

	present   bool
	referrers []*corev1.RecordReferrer
	reads     int
}

func (s *gatedStore) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	s.reads++

	if !s.present {
		return nil, types.RecordNotFoundError(ref.GetCid())
	}

	return &corev1.Record{}, nil
}

func (s *gatedStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	s.reads++

	if !s.present {
		return nil, types.RecordNotFoundError(ref.GetCid())
	}

	return &corev1.RecordMeta{Cid: ref.GetCid()}, nil
}

func (s *gatedStore) WalkReferrers(_ context.Context, _ string, _ string, walkFn func(*corev1.RecordReferrer) error) error {
	s.reads++

	if !s.present {
		return status.Error(codes.NotFound, "failed to resolve record manifest")
	}

	for _, referrer := range s.referrers {
		if err := walkFn(referrer); err != nil {
			return err
		}
	}

	return nil
}

// gatedDB answers the policy gate and counts pulls and lookups.
type gatedDB struct {
	types.DatabaseAPI

	servable bool
	gateErr  error
	pulls    int
	lookups  int
}

func (d *gatedDB) IsRecordServable(string) (bool, error) { return d.servable, d.gateErr }

func (d *gatedDB) IncrementPullCount(string) error {
	d.pulls++

	return nil
}

func (d *gatedDB) IncrementLookupCount(string) error {
	d.lookups++

	return nil
}

// fakeStream is a server stream that yields reqs, then EOF, and records what
// is sent.
type fakeStream[Req, Resp any] struct {
	grpc.ServerStream

	ctx  context.Context //nolint:containedctx // a stream carries its context
	reqs []*Req
	sent []*Resp
}

func (f *fakeStream[Req, Resp]) Recv() (*Req, error) {
	if len(f.reqs) == 0 {
		return nil, io.EOF
	}

	req := f.reqs[0]
	f.reqs = f.reqs[1:]

	return req, nil
}

func (f *fakeStream[Req, Resp]) Send(resp *Resp) error {
	f.sent = append(f.sent, resp)

	return nil
}

func (f *fakeStream[Req, Resp]) Context() context.Context { return f.ctx }

func pullOnce(t *testing.T, store *gatedStore, db *gatedDB) (*fakeStream[corev1.RecordRef, corev1.Record], error) {
	t.Helper()

	stream := &fakeStream[corev1.RecordRef, corev1.Record]{ctx: t.Context(), reqs: []*corev1.RecordRef{{Cid: gatedCID}}}

	return stream, NewStoreController(store, db, nil, nil, nil).Pull(stream)
}

func lookupOnce(t *testing.T, store *gatedStore, db *gatedDB) (*fakeStream[corev1.RecordRef, corev1.RecordMeta], error) {
	t.Helper()

	stream := &fakeStream[corev1.RecordRef, corev1.RecordMeta]{ctx: t.Context(), reqs: []*corev1.RecordRef{{Cid: gatedCID}}}

	return stream, NewStoreController(store, db, nil, nil, nil).Lookup(stream)
}

func pullReferrerOnce(t *testing.T, store *gatedStore, db *gatedDB) (*fakeStream[storev1.PullReferrerRequest, storev1.PullReferrerResponse], error) {
	t.Helper()

	stream := &fakeStream[storev1.PullReferrerRequest, storev1.PullReferrerResponse]{
		ctx:  t.Context(),
		reqs: []*storev1.PullReferrerRequest{{RecordRef: &corev1.RecordRef{Cid: gatedCID}}},
	}

	return stream, NewStoreController(store, db, nil, nil, nil).PullReferrer(stream)
}

// --- Pull ---

// excludedMessage is what a caller is told about a record the gate excludes:
// the record is not available under the node's content policy, and nothing
// more.
func excludedMessage(prefix string) string {
	return prefix + status.Convert(types.RecordExcludedError(gatedCID)).Message()
}

// A caller is told a record the gate excludes is not available under the
// node's content policy. A record that passes the gate but is missing from the
// store is still not found.
func TestPull_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	exStore, exDB := &gatedStore{present: true}, &gatedDB{servable: false}
	_, exErr := pullOnce(t, exStore, exDB)

	_, missErr := pullOnce(t, &gatedStore{present: false}, &gatedDB{servable: true})

	require.Error(t, exErr)
	require.Error(t, missErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
	assert.Equal(t, excludedMessage("failed to pull record: "), status.Convert(exErr).Message(), "no policy or reason in it")
	assert.Equal(t, codes.NotFound, status.Code(missErr), "a record that passes the gate but is not in the store is still not found")
	assert.Zero(t, exStore.reads, "the store must not be read for an excluded record")
	assert.Zero(t, exDB.pulls, "an excluded record is not a pull")
}

func TestPull_ServableRecordIsServed(t *testing.T) {
	t.Parallel()

	store, db := &gatedStore{present: true}, &gatedDB{servable: true}

	stream, err := pullOnce(t, store, db)
	require.NoError(t, err)
	assert.Len(t, stream.sent, 1)
	assert.Equal(t, 1, db.pulls)
}

// When the gate cannot answer, nothing is served.
func TestPull_GateErrorFailsClosed(t *testing.T) {
	t.Parallel()

	store, db := &gatedStore{present: true}, &gatedDB{gateErr: errors.New("database unavailable")}

	stream, err := pullOnce(t, store, db)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Empty(t, stream.sent)
	assert.Zero(t, store.reads)
}

// --- Lookup ---

func TestLookup_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	exStore, exDB := &gatedStore{present: true}, &gatedDB{servable: false}
	_, exErr := lookupOnce(t, exStore, exDB)

	_, missErr := lookupOnce(t, &gatedStore{present: false}, &gatedDB{servable: true})

	require.Error(t, exErr)
	require.Error(t, missErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
	assert.Equal(t, excludedMessage("failed to lookup record: "), status.Convert(exErr).Message(), "no policy or reason in it")
	assert.Equal(t, codes.NotFound, status.Code(missErr))
	assert.Zero(t, exStore.reads, "the store must not be read for an excluded record")
	assert.Zero(t, exDB.lookups, "an excluded record is not a lookup")
}

func TestLookup_ServableRecordIsServed(t *testing.T) {
	t.Parallel()

	store, db := &gatedStore{present: true}, &gatedDB{servable: true}

	stream, err := lookupOnce(t, store, db)
	require.NoError(t, err)
	require.Len(t, stream.sent, 1)
	assert.Equal(t, gatedCID, stream.sent[0].GetCid())
	assert.Equal(t, 1, db.lookups)
}

// --- PullReferrer ---

// The referrers of an excluded record are refused like the record itself, not
// answered with an empty list that would read as "no signatures".
func TestPullReferrer_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	exStore := &gatedStore{present: true, referrers: []*corev1.RecordReferrer{{Type: corev1.SignatureReferrerType}}}
	exStream, exErr := pullReferrerOnce(t, exStore, &gatedDB{servable: false})

	require.Error(t, exErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
	assert.Equal(t, excludedMessage(""), status.Convert(exErr).Message(), "no policy or reason in it")
	assert.Empty(t, exStream.sent)
	assert.Zero(t, exStore.reads, "the store must not be read for an excluded record")
}

func TestPullReferrer_ServableRecordStreamsReferrers(t *testing.T) {
	t.Parallel()

	store := &gatedStore{
		present:   true,
		referrers: []*corev1.RecordReferrer{{Type: corev1.SignatureReferrerType}, {Type: corev1.PublicKeyReferrerType}},
	}

	stream, err := pullReferrerOnce(t, store, &gatedDB{servable: true})
	require.NoError(t, err)
	assert.Len(t, stream.sent, 2)
}

// --- ExportAgent ---

func exportOnce(t *testing.T, store *gatedStore, db *fakeCatalogDB) error {
	t.Helper()

	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, store)
	_, err := ctrl.ExportAgent(t.Context(), &catalogv1.ExportAgentRequest{Cid: gatedCID})

	return err
}

func TestExportAgent_ExcludedRecordIsReportedAsExcluded(t *testing.T) {
	t.Parallel()

	exStore := &gatedStore{present: true}
	exErr := exportOnce(t, exStore, &fakeCatalogDB{excluded: true})

	missErr := exportOnce(t, &gatedStore{present: false}, &fakeCatalogDB{})

	require.Error(t, exErr)
	require.Error(t, missErr)
	assert.Equal(t, codes.PermissionDenied, status.Code(exErr))
	assert.Equal(t, excludedMessage(""), status.Convert(exErr).Message(), "no policy or reason in it")
	assert.Equal(t, codes.NotFound, status.Code(missErr))
	assert.Zero(t, exStore.reads, "the store must not be read for an excluded record")
}
