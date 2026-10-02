// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"net/url"
	"testing"
	"time"

	oasftypes "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

const identityTestCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

// fakePushReferrerStream records what the client sends and answers with resp.
type fakePushReferrerStream struct {
	grpc.ClientStream
	sent    []*storev1.PushReferrerRequest
	resp    *storev1.PushReferrerResponse
	sendErr error
	recvErr error
}

func (s *fakePushReferrerStream) Send(req *storev1.PushReferrerRequest) error {
	s.sent = append(s.sent, req)

	return s.sendErr
}

func (s *fakePushReferrerStream) CloseSend() error { return nil }

func (s *fakePushReferrerStream) Recv() (*storev1.PushReferrerResponse, error) {
	return s.resp, s.recvErr
}

// fakePullStream answers a Pull with record, then ends.
type fakePullStream struct {
	grpc.ClientStream
	record *corev1.Record
	asked  chan struct{}
}

func (s *fakePullStream) Send(*corev1.RecordRef) error {
	close(s.asked)

	return nil
}

func (s *fakePullStream) CloseSend() error { return nil }

func (s *fakePullStream) Recv() (*corev1.Record, error) {
	<-s.asked

	record := s.record
	s.record = nil

	if record == nil {
		return nil, io.EOF
	}

	return record, nil
}

type fakeStoreClient struct {
	storev1.StoreServiceClient
	stream *fakePushReferrerStream
	record *corev1.Record
}

func (f *fakeStoreClient) Pull(context.Context, ...grpc.CallOption) (storev1.StoreService_PullClient, error) {
	return &fakePullStream{record: f.record, asked: make(chan struct{})}, nil
}

func (f *fakeStoreClient) PushReferrer(context.Context, ...grpc.CallOption) (storev1.StoreService_PushReferrerClient, error) {
	return f.stream, nil
}

type fakeIdentityClient struct {
	identityv1.IdentityServiceClient
	statusReq  *identityv1.GetIdentityStatusRequest
	statusResp *identityv1.GetIdentityStatusResponse
	resolveReq *identityv1.ResolveRequest
	resolveErr error
}

func (f *fakeIdentityClient) GetIdentityStatus(_ context.Context, req *identityv1.GetIdentityStatusRequest, _ ...grpc.CallOption) (*identityv1.GetIdentityStatusResponse, error) {
	f.statusReq = req

	return f.statusResp, nil
}

func (f *fakeIdentityClient) Resolve(_ context.Context, req *identityv1.ResolveRequest, _ ...grpc.CallOption) (*identityv1.ResolveResponse, error) {
	f.resolveReq = req

	return &identityv1.ResolveResponse{}, f.resolveErr
}

// newClaimTestClient returns a client whose store holds a record carrying the
// given annotations.
func newClaimTestClient(stream *fakePushReferrerStream, annotations map[string]string) *Client {
	record := corev1.New(&oasftypes.Record{Name: "acme.com/agent", Version: "1.0.0", Annotations: annotations})

	return &Client{StoreServiceClient: &fakeStoreClient{stream: stream, record: record}}
}

func newClaimKey(t *testing.T) (*ecdsa.PrivateKey, jws.Signer) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	signer, err := jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
	require.NoError(t, err)

	return key, signer
}

func claimTestCertPEM(t *testing.T, key *ecdsa.PrivateKey, subject string) []byte {
	t.Helper()

	uri, err := url.Parse(subject)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// pushedClaim decodes the claim carried by the single request the client sent.
func pushedClaim(t *testing.T, stream *fakePushReferrerStream) (*storev1.PushReferrerRequest, *identityv1.Claim) {
	t.Helper()

	require.Len(t, stream.sent, 1)

	req := stream.sent[0]

	claim := &identityv1.Claim{}
	require.NoError(t, claim.UnmarshalReferrer(&corev1.RecordReferrer{Type: req.GetType(), Data: req.GetData()}))

	return req, claim
}

func TestClaimIdentity_PushesVerifiableClaim(t *testing.T) {
	key, signer := newClaimKey(t)
	stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

	claimed, err := newClaimTestClient(stream, map[string]string{corev1.AnnotationKeyIdentity: "did:web:acme.com:agent"}).
		ClaimIdentity(t.Context(), identityTestCID, signer)
	require.NoError(t, err)

	req, claim := pushedClaim(t, stream)
	assert.Equal(t, identityTestCID, req.GetRecordRef().GetCid())
	assert.Equal(t, corev1.IdentityClaimReferrerType, req.GetType())
	assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, claim.GetRole())
	assert.Equal(t, "did:web:acme.com:agent", claim.GetSubject(), "subject comes from the record annotation")
	assert.Equal(t, claim.GetSubject(), claimed.GetSubject())
	assert.Nil(t, claim.Certificate, "no certificate unless asked for")

	// The pushed claim is genuinely signed for this record, by this key.
	ok, err := identity.Verify(claim, identityTestCID, "did:web:acme.com:agent", &key.PublicKey)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestClaimOwnership_PushesOwnershipClaim(t *testing.T) {
	key, signer := newClaimKey(t)
	stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

	// An identity annotation is not an owner.
	annotations := map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com", corev1.AnnotationKeyIdentity: "did:web:other.com"}

	_, err := newClaimTestClient(stream, annotations).ClaimOwnership(t.Context(), identityTestCID, signer)
	require.NoError(t, err)

	req, claim := pushedClaim(t, stream)
	assert.Equal(t, corev1.OwnershipClaimReferrerType, req.GetType())
	assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_OWNER, claim.GetRole())
	assert.Equal(t, "dns:acme.com", claim.GetSubject())

	ok, err := identity.Verify(claim, identityTestCID, "dns:acme.com", &key.PublicKey)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestClaimIdentity_WithCertificate(t *testing.T) {
	key, signer := newClaimKey(t)
	stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

	const subject = "spiffe://acme.com/agents/finance"

	_, err := newClaimTestClient(stream, map[string]string{corev1.AnnotationKeyIdentity: subject}).
		ClaimIdentity(t.Context(), identityTestCID, signer, identity.WithCertificate(claimTestCertPEM(t, key, subject)))
	require.NoError(t, err)

	_, claim := pushedClaim(t, stream)
	assert.NotEmpty(t, claim.GetCertificate())
}

func TestClaim_Errors(t *testing.T) {
	key, signer := newClaimKey(t)
	identityNote := map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"}

	t.Run("empty cid", func(t *testing.T) {
		stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

		_, err := newClaimTestClient(stream, identityNote).ClaimIdentity(t.Context(), "", signer)
		require.Error(t, err)
		assert.Empty(t, stream.sent)
	})

	t.Run("record without the annotation", func(t *testing.T) {
		stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

		_, err := newClaimTestClient(stream, identityNote).ClaimOwnership(t.Context(), identityTestCID, signer)
		require.ErrorContains(t, err, corev1.AnnotationKeyOwner)

		_, err = newClaimTestClient(stream, nil).ClaimIdentity(t.Context(), identityTestCID, signer)
		require.ErrorContains(t, err, corev1.AnnotationKeyIdentity)
		assert.Empty(t, stream.sent)
	})

	t.Run("record that cannot be pulled", func(t *testing.T) {
		stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}
		c := &Client{StoreServiceClient: &fakeStoreClient{stream: stream}}

		_, err := c.ClaimIdentity(t.Context(), identityTestCID, signer)
		require.Error(t, err)
		assert.Empty(t, stream.sent)
	})

	t.Run("claim that cannot be signed is not pushed", func(t *testing.T) {
		stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: true}}

		_, err := newClaimTestClient(stream, identityNote).ClaimIdentity(t.Context(), identityTestCID, nil)
		require.Error(t, err, "no signer")

		spiffe := map[string]string{corev1.AnnotationKeyIdentity: "spiffe://acme.com/x"}

		_, err = newClaimTestClient(stream, spiffe).ClaimIdentity(t.Context(), identityTestCID, signer,
			identity.WithCertificate(claimTestCertPEM(t, key, "spiffe://acme.com/other")))
		require.Error(t, err, "certificate for another subject")

		_, err = newClaimTestClient(stream, spiffe).ClaimIdentity(t.Context(), identityTestCID, signer)
		require.ErrorContains(t, err, "needs a certificate")

		_, err = newClaimTestClient(stream, identityNote).ClaimIdentity(t.Context(), identityTestCID, signer,
			identity.WithCertificate(claimTestCertPEM(t, key, "spiffe://acme.com/x")))
		require.Error(t, err, "certificate for a subject that is not spiffe://")

		// Even one that names the subject itself.
		const https = "https://acme.com/agents"

		_, err = newClaimTestClient(stream, map[string]string{corev1.AnnotationKeyIdentity: https}).ClaimIdentity(t.Context(), identityTestCID, signer,
			identity.WithCertificate(claimTestCertPEM(t, key, https)))
		require.ErrorContains(t, err, "only used for a spiffe:// subject")
		assert.Empty(t, stream.sent)
	})

	t.Run("server refuses the referrer", func(t *testing.T) {
		msg := "record not found"
		stream := &fakePushReferrerStream{resp: &storev1.PushReferrerResponse{Success: false, ErrorMessage: &msg}}

		_, err := newClaimTestClient(stream, identityNote).ClaimIdentity(t.Context(), identityTestCID, signer)
		require.ErrorContains(t, err, "record not found")
	})

	t.Run("transport failure", func(t *testing.T) {
		stream := &fakePushReferrerStream{recvErr: errors.New("connection reset")}

		_, err := newClaimTestClient(stream, identityNote).ClaimIdentity(t.Context(), identityTestCID, signer)
		require.ErrorContains(t, err, "connection reset")
	})
}

func TestGetIdentityStatus(t *testing.T) {
	want := &identityv1.GetIdentityStatusResponse{
		Identity: &identityv1.ClaimVerification{Subject: "did:web:acme.com"},
	}
	fake := &fakeIdentityClient{statusResp: want}
	c := &Client{IdentityServiceClient: fake}

	got, err := c.GetIdentityStatus(t.Context(), identityTestCID)
	require.NoError(t, err)
	assert.Same(t, want, got)
	assert.Equal(t, identityTestCID, fake.statusReq.GetCid())
	assert.Nil(t, fake.statusReq.Name)

	_, err = c.GetIdentityStatus(t.Context(), "")
	require.Error(t, err)
}

func TestResolveIdentity(t *testing.T) {
	fake := &fakeIdentityClient{}
	c := &Client{IdentityServiceClient: fake}

	_, err := c.ResolveIdentity(t.Context(), "acme.com/agent", "")
	require.NoError(t, err)
	assert.Equal(t, "acme.com/agent", fake.resolveReq.GetName())
	assert.Nil(t, fake.resolveReq.Version, "no version means all versions")

	_, err = c.ResolveIdentity(t.Context(), "acme.com/agent", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", fake.resolveReq.GetVersion())

	fake.resolveErr = errors.New("not found")
	_, err = c.ResolveIdentity(t.Context(), "missing.com/agent", "")
	require.ErrorContains(t, err, "not found")
}
