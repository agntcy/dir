// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package autosync

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- mocks ---

type fakeTransport struct {
	pullRecord      *corev1.Record
	pullErr         error
	pullCalls       int
	pullFailsBefore int // number of initial Pull calls that return a transient error

	descriptors []rpc.ReferrerDescriptor
	listErr     error

	referrers         map[string]*corev1.RecordReferrer
	pullReferrerErr   error
	pullReferrerCalls int
}

func (f *fakeTransport) Pull(_ context.Context, _ peer.ID, _ *corev1.RecordRef) (*corev1.Record, error) {
	f.pullCalls++

	if f.pullCalls <= f.pullFailsBefore {
		return nil, errors.New("transient pull failure")
	}

	return f.pullRecord, f.pullErr
}

func (f *fakeTransport) ListReferrers(_ context.Context, _ peer.ID, _ *corev1.RecordRef) ([]rpc.ReferrerDescriptor, error) {
	return f.descriptors, f.listErr
}

func (f *fakeTransport) PullReferrer(_ context.Context, _ peer.ID, _ *corev1.RecordRef, desc rpc.ReferrerDescriptor) (*corev1.RecordReferrer, error) {
	f.pullReferrerCalls++

	if f.pullReferrerErr != nil {
		return nil, f.pullReferrerErr
	}

	return f.referrers[desc.Cid], nil
}

type fakeRouter struct{}

func (fakeRouter) AddAddrs(peer.ID, []ma.Multiaddr) {}

func (fakeRouter) FindPeer(context.Context, peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, errors.New("find peer not available in test")
}

// countingRouter records FindPeer calls and returns configurable addresses.
type countingRouter struct {
	findPeerCalls int
	addrs         []ma.Multiaddr
	findErr       error
}

func (r *countingRouter) AddAddrs(peer.ID, []ma.Multiaddr) {}

func (r *countingRouter) FindPeer(context.Context, peer.ID) (peer.AddrInfo, error) {
	r.findPeerCalls++

	if r.findErr != nil {
		return peer.AddrInfo{}, r.findErr
	}

	return peer.AddrInfo{Addrs: r.addrs}, nil
}

type fakeIngestor struct {
	importRecordCalls   int
	importReferrerCalls int
	lastRecord          *corev1.Record
}

func (f *fakeIngestor) ImportRecord(_ context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	f.importRecordCalls++
	f.lastRecord = record

	return &corev1.RecordRef{Cid: record.GetCid()}, nil
}

func (f *fakeIngestor) ImportReferrer(_ context.Context, _ string, _ *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	f.importReferrerCalls++

	return &corev1.ReferrerRef{Cid: "referrer-cid"}, nil
}

type fakeStore struct {
	found bool
}

func (f *fakeStore) Push(context.Context, *corev1.Record) (*corev1.RecordRef, error) {
	return nil, nil //nolint:nilnil
}

func (f *fakeStore) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	return nil, nil //nolint:nilnil
}

func (f *fakeStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	if f.found {
		return &corev1.RecordMeta{Cid: ref.GetCid()}, nil
	}

	return nil, errors.New("not found")
}

func (f *fakeStore) Delete(context.Context, *corev1.RecordRef) error { return nil }
func (f *fakeStore) IsReady(context.Context) bool                    { return true }

type fakeValidator struct {
	valid bool
}

func (f fakeValidator) ValidateRecord(context.Context, *structpb.Struct) (bool, []string, []string, error) {
	if f.valid {
		return true, nil, nil, nil
	}

	return false, []string{"schema invalid"}, nil, nil
}

// --- helpers ---

func randomPeerID(t *testing.T) peer.ID {
	t.Helper()

	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	pid, err := peer.IDFromPublicKey(pub)
	require.NoError(t, err)

	return pid
}

func testRecord(t *testing.T) *corev1.Record {
	t.Helper()

	rec := corev1.New(&typesv1alpha1.Record{Name: "autosync-test", SchemaVersion: "0.7.0"})
	require.NotEmpty(t, rec.GetCid())

	return rec
}

func newTestManager(allow map[peer.ID]struct{}, tr *fakeTransport, ing *fakeIngestor, st *fakeStore, valid bool) *Manager {
	return newManager(allow, tr, fakeRouter{}, ing, st, fakeValidator{valid: valid})
}

// --- tests ---

func TestMaybeEnqueue_AllowListGating(t *testing.T) {
	trusted := randomPeerID(t)
	untrusted := randomPeerID(t)

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, &fakeTransport{}, &fakeIngestor{}, &fakeStore{}, true)

	// Untrusted peer: ignored.
	m.MaybeEnqueue(&corev1.RecordRef{Cid: "cid-a"}, peer.AddrInfo{ID: untrusted})
	assert.Empty(t, m.queue)

	// Trusted peer: enqueued.
	m.MaybeEnqueue(&corev1.RecordRef{Cid: "cid-a"}, peer.AddrInfo{ID: trusted})
	assert.Len(t, m.queue, 1)
}

func TestMaybeEnqueue_Dedupe(t *testing.T) {
	trusted := randomPeerID(t)
	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, &fakeTransport{}, &fakeIngestor{}, &fakeStore{}, true)

	ref := &corev1.RecordRef{Cid: "cid-a"}
	m.MaybeEnqueue(ref, peer.AddrInfo{ID: trusted})
	m.MaybeEnqueue(ref, peer.AddrInfo{ID: trusted}) // same CID already in flight

	assert.Len(t, m.queue, 1, "duplicate CID must not be enqueued twice")
}

func TestProcess_HappyPath(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	cid := rec.GetCid()

	tr := &fakeTransport{
		pullRecord:  rec,
		descriptors: []rpc.ReferrerDescriptor{{Cid: "ref1", Type: corev1.SignatureReferrerType}},
		referrers: map[string]*corev1.RecordReferrer{
			"ref1": {
				Type:        corev1.SignatureReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: cid},
				ReferrerRef: &corev1.ReferrerRef{Cid: "ref1"},
			},
		},
	}
	ing := &fakeIngestor{}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, true)
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: cid}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 1, ing.importRecordCalls, "record should be ingested")
	assert.Equal(t, cid, ing.lastRecord.GetCid())
	assert.Equal(t, 1, ing.importReferrerCalls, "signature referrer should be ingested")
}

func TestProcess_AlreadyLocalSkipsPull(t *testing.T) {
	trusted := randomPeerID(t)
	tr := &fakeTransport{}
	ing := &fakeIngestor{}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{found: true}, true)
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: "cid-a"}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 0, tr.pullCalls, "must not pull a record we already have")
	assert.Equal(t, 0, ing.importRecordCalls)
}

func TestProcess_CIDMismatchRejected(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	ing := &fakeIngestor{}

	tr := &fakeTransport{pullRecord: rec}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, true)
	// Announce a CID that does not match the pulled record's content.
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: "different-cid"}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 0, ing.importRecordCalls, "record with mismatched CID must be rejected")
}

func TestProcess_ValidationFailureRejected(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	ing := &fakeIngestor{}

	tr := &fakeTransport{pullRecord: rec}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, false) // validator rejects
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: rec.GetCid()}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 0, ing.importRecordCalls, "OASF-invalid record must be rejected")
}

func TestProcess_DisallowedReferrerTypeSkipped(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	cid := rec.GetCid()
	ing := &fakeIngestor{}

	tr := &fakeTransport{
		pullRecord:  rec,
		descriptors: []rpc.ReferrerDescriptor{{Cid: "evil1", Type: "com.attacker.evil"}},
	}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, true)
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: cid}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 1, ing.importRecordCalls, "record still ingested")
	assert.Equal(t, 0, tr.pullReferrerCalls, "disallowed referrer type must not be pulled")
	assert.Equal(t, 0, ing.importReferrerCalls)
}

func TestProcess_ReferrerBelongsToOtherRecordRejected(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	cid := rec.GetCid()
	ing := &fakeIngestor{}

	tr := &fakeTransport{
		pullRecord:  rec,
		descriptors: []rpc.ReferrerDescriptor{{Cid: "ref1", Type: corev1.SignatureReferrerType}},
		referrers: map[string]*corev1.RecordReferrer{
			"ref1": {
				Type:        corev1.SignatureReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: "some-other-record"}, // belongs elsewhere
				ReferrerRef: &corev1.ReferrerRef{Cid: "ref1"},
			},
		},
	}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, true)
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: cid}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 1, ing.importRecordCalls)
	assert.Equal(t, 0, ing.importReferrerCalls, "referrer belonging to a different record must be rejected")
}

func TestPullRecord_RetriesThenSucceeds(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	cid := rec.GetCid()

	tr := &fakeTransport{pullRecord: rec, pullFailsBefore: 1} // fail once, then succeed
	ing := &fakeIngestor{}

	m := newTestManager(map[peer.ID]struct{}{trusted: {}}, tr, ing, &fakeStore{}, true)
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: cid}, peer: peer.AddrInfo{ID: trusted}})

	assert.Equal(t, 2, tr.pullCalls, "should retry after a transient failure")
	assert.Equal(t, 1, ing.importRecordCalls, "record ingested after retry")
}

func TestPullRecord_ResolvesAddressesWhenNoneProvided(t *testing.T) {
	trusted := randomPeerID(t)
	rec := testRecord(t)
	cid := rec.GetCid()

	tr := &fakeTransport{pullRecord: rec}
	ing := &fakeIngestor{}
	router := &countingRouter{}

	// GossipSub-triggered job: peer has an ID but no addresses.
	m := newManager(map[peer.ID]struct{}{trusted: {}}, tr, router, ing, &fakeStore{}, fakeValidator{valid: true})
	m.process(t.Context(), job{ref: &corev1.RecordRef{Cid: cid}, peer: peer.AddrInfo{ID: trusted}})

	assert.GreaterOrEqual(t, router.findPeerCalls, 1, "must resolve addresses via FindPeer when none are provided")
	assert.Equal(t, 1, ing.importRecordCalls, "record should still be ingested")
}

func TestComputeBackoff(t *testing.T) {
	// First retry (attempt 2) is at least the base and below the cap+jitter.
	b := computeBackoff(2)
	assert.GreaterOrEqual(t, b, pullBackoffBase)

	// Large attempts are capped (plus at most ~50% jitter).
	capped := computeBackoff(100)
	assert.LessOrEqual(t, capped, pullBackoffMax+pullBackoffMax/2+1)
}

func TestVerifyReferrer(t *testing.T) {
	m := &Manager{}
	desc := rpc.ReferrerDescriptor{Cid: "ref1", Type: corev1.SignatureReferrerType}

	tests := []struct {
		name     string
		referrer *corev1.RecordReferrer
		want     bool
	}{
		{
			name: "valid",
			referrer: &corev1.RecordReferrer{
				Type:        corev1.SignatureReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: "rec"},
				ReferrerRef: &corev1.ReferrerRef{Cid: "ref1"},
			},
			want: true,
		},
		{
			name:     "nil referrer",
			referrer: nil,
			want:     false,
		},
		{
			name: "wrong record",
			referrer: &corev1.RecordReferrer{
				Type:        corev1.SignatureReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: "other"},
				ReferrerRef: &corev1.ReferrerRef{Cid: "ref1"},
			},
			want: false,
		},
		{
			name: "type mismatch",
			referrer: &corev1.RecordReferrer{
				Type:        corev1.PublicKeyReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: "rec"},
				ReferrerRef: &corev1.ReferrerRef{Cid: "ref1"},
			},
			want: false,
		},
		{
			name: "referrer cid mismatch",
			referrer: &corev1.RecordReferrer{
				Type:        corev1.SignatureReferrerType,
				RecordRef:   &corev1.RecordRef{Cid: "rec"},
				ReferrerRef: &corev1.ReferrerRef{Cid: "different"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, m.verifyReferrer(tt.referrer, "rec", desc))
		})
	}
}
