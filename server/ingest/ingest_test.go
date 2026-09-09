// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/identity/spiffe"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDB implements types.DatabaseAPI via embedding; only the methods the
// Ingestor uses are overridden. Any other method would panic if called (nil
// embedded interface), which keeps the mock small and surfaces unexpected calls.
type mockDB struct {
	types.DatabaseAPI

	addRecordErr error
	setSignedErr error

	addRecordCalls  int
	setSignedCalls  int
	invalidateCalls int
	upsertCalls     int
	upsertReport    types.ScanReportObject

	upsertIdentityCalls  int
	upsertOwnershipCalls int
	lastIdentityClaim    types.ClaimObject
	lastOwnershipClaim   types.ClaimObject
}

func (m *mockDB) AddRecord(coretypes.Record) error {
	m.addRecordCalls++

	return m.addRecordErr
}

func (m *mockDB) SetRecordSigned(string) error {
	m.setSignedCalls++

	return m.setSignedErr
}

func (m *mockDB) InvalidateSignatureVerificationsForRecord(string) error {
	m.invalidateCalls++

	return nil
}

func (m *mockDB) UpsertScanReport(r types.ScanReportObject, _ types.ScanSchedule) error {
	m.upsertCalls++
	m.upsertReport = r

	return nil
}

func (m *mockDB) UpsertClaim(role string, c types.ClaimObject) error {
	switch role {
	case types.ClaimRoleIdentity:
		m.upsertIdentityCalls++
		m.lastIdentityClaim = c
	case types.ClaimRoleOwner:
		m.upsertOwnershipCalls++
		m.lastOwnershipClaim = c
	}

	return nil
}

// mockStore implements types.FullStore (StoreAPI + ReferrerStoreAPI).
type mockStore struct {
	pushRef   *corev1.RecordRef
	pushErr   error
	pushCalls int

	pushReferrerRef   *corev1.ReferrerRef
	pushReferrerErr   error
	pushReferrerCalls int

	// pullRecord is returned by Pull, used to supply the record whose
	// identity/owner annotations a claim's subject is checked against.
	pullRecord *corev1.Record
}

func (m *mockStore) Push(_ context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	m.pushCalls++

	if m.pushErr != nil {
		return nil, m.pushErr
	}

	if m.pushRef != nil {
		return m.pushRef, nil
	}

	return &corev1.RecordRef{Cid: record.GetCid()}, nil
}

func (m *mockStore) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	return m.pullRecord, nil
}

func (m *mockStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	return &corev1.RecordMeta{Cid: ref.GetCid()}, nil
}

func (m *mockStore) Delete(context.Context, *corev1.RecordRef) error { return nil }
func (m *mockStore) IsReady(context.Context) bool                    { return true }

func (m *mockStore) PushReferrer(_ context.Context, _ string, _ *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	m.pushReferrerCalls++

	if m.pushReferrerErr != nil {
		return nil, m.pushReferrerErr
	}

	if m.pushReferrerRef != nil {
		return m.pushReferrerRef, nil
	}

	return &corev1.ReferrerRef{Cid: "referrer-cid"}, nil
}

func (m *mockStore) WalkReferrers(context.Context, string, string, func(*corev1.RecordReferrer) error) error {
	return nil
}

func (m *mockStore) DeleteReferrer(context.Context, string, string, string) ([]string, error) {
	return nil, nil
}

func (m *mockStore) DeleteReferrers(context.Context, string, []string, string) ([]string, error) {
	return nil, nil
}

// recordOnlyStore implements only types.StoreAPI (no referrer support).
type recordOnlyStore struct {
	types.StoreAPI
}

func newTestRecord() *corev1.Record {
	return corev1.New(&typesv1alpha1.Record{
		Name:          "test-record",
		SchemaVersion: "0.7.0",
	})
}

func newTestRecordWithAnnotations(annotations map[string]string) *corev1.Record {
	return corev1.New(&typesv1alpha1.Record{
		Name:          "test-record",
		SchemaVersion: "0.7.0",
		Annotations:   annotations,
	})
}

func TestImportRecord_Success(t *testing.T) {
	store := &mockStore{pushRef: &corev1.RecordRef{Cid: "cid-1"}}
	db := &mockDB{}

	ref, err := New(store, db).ImportRecord(t.Context(), newTestRecord())

	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, "cid-1", ref.GetCid())
	assert.Equal(t, 1, store.pushCalls)
	assert.Equal(t, 1, db.addRecordCalls, "record should be added to the search index")
}

func TestImportRecord_PushError(t *testing.T) {
	store := &mockStore{pushErr: errors.New("push boom")}
	db := &mockDB{}

	ref, err := New(store, db).ImportRecord(t.Context(), newTestRecord())

	require.Error(t, err)
	assert.Nil(t, ref)
	assert.Equal(t, 0, db.addRecordCalls, "index must not be touched when store push fails")
}

func TestImportRecord_IndexErrorIsNonFatal(t *testing.T) {
	store := &mockStore{pushRef: &corev1.RecordRef{Cid: "cid-1"}}
	db := &mockDB{addRecordErr: errors.New("index boom")}

	ref, err := New(store, db).ImportRecord(t.Context(), newTestRecord())

	// Store is the source of truth: an indexing failure must not fail the import.
	require.NoError(t, err)
	require.NotNil(t, ref)
	assert.Equal(t, "cid-1", ref.GetCid())
	assert.Equal(t, 1, db.addRecordCalls)
}

func TestImportReferrer_Signature(t *testing.T) {
	store := &mockStore{}
	db := &mockDB{}

	ref, err := New(store, db).ImportReferrer(t.Context(), "record-cid", &corev1.RecordReferrer{
		Type:      corev1.SignatureReferrerType,
		RecordRef: &corev1.RecordRef{Cid: "record-cid"},
	})

	require.NoError(t, err)
	assert.Equal(t, "referrer-cid", ref.GetCid())
	assert.Equal(t, 1, store.pushReferrerCalls)
	assert.Equal(t, 1, db.setSignedCalls, "signature should mark record signed")
	assert.Equal(t, 1, db.invalidateCalls, "signature should invalidate cached verifications")
	assert.Equal(t, 0, db.upsertCalls)
}

func TestImportReferrer_PublicKey(t *testing.T) {
	store := &mockStore{}
	db := &mockDB{}

	_, err := New(store, db).ImportReferrer(t.Context(), "record-cid", &corev1.RecordReferrer{
		Type:      corev1.PublicKeyReferrerType,
		RecordRef: &corev1.RecordRef{Cid: "record-cid"},
	})

	require.NoError(t, err)
	assert.Equal(t, 0, db.setSignedCalls, "public key must not mark record signed")
	assert.Equal(t, 1, db.invalidateCalls, "public key should invalidate cached verifications")
	assert.Equal(t, 0, db.upsertCalls)
}

func TestImportReferrer_ScanReport(t *testing.T) {
	store := &mockStore{}
	db := &mockDB{}

	referrer, err := (&securityv1.ScanReport{
		ScannerType: securityv1.ScannerType_SCANNER_TYPE_MCP,
		IsSafe:      true,
		MaxSeverity: securityv1.Severity_SEVERITY_HIGH,
	}).MarshalReferrer()
	require.NoError(t, err)

	_, err = New(store, db).ImportReferrer(t.Context(), "record-cid", referrer)

	require.NoError(t, err)
	assert.Equal(t, 0, db.setSignedCalls)
	assert.Equal(t, 0, db.invalidateCalls)
	require.Equal(t, 1, db.upsertCalls)
	require.NotNil(t, db.upsertReport)
	assert.Equal(t, "record-cid", db.upsertReport.GetRecordCID())
	assert.Equal(t, "MCP", db.upsertReport.GetScannerType())
	assert.Equal(t, "HIGH", db.upsertReport.GetMaxSeverity())
	assert.True(t, db.upsertReport.GetIsSafe())
}

func TestImportReferrer_IdentityClaim_EagerlyVerifiedAndIndexed(t *testing.T) {
	store := &mockStore{pullRecord: newTestRecordWithAnnotations(map[string]string{
		corev1.AnnotationKeyIdentity: "spiffe://acme.com/agents/finance",
	})}
	db := &mockDB{}

	keyPEM, certPEM, cert := generateSpiffeKeyCert(t, "spiffe://acme.com/agents/finance")
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim("spiffe://acme.com/agents/finance")
	require.NoError(t, identityv1.SignIdentityClaim(claim, "record-cid", signer))

	referrer, err := claim.MarshalReferrer()
	require.NoError(t, err)

	bundles := spiffe.NewBundles(map[string][]*x509.Certificate{"acme.com": {cert}})

	_, err = New(store, db, WithSpiffeBundles(bundles)).ImportReferrer(t.Context(), "record-cid", referrer)

	require.NoError(t, err)
	require.Equal(t, 1, db.upsertIdentityCalls)
	require.NotNil(t, db.lastIdentityClaim)
	assert.Equal(t, "record-cid", db.lastIdentityClaim.GetRecordCID())
	assert.Equal(t, "verified", db.lastIdentityClaim.GetStatus())
	assert.Empty(t, db.lastIdentityClaim.GetError())
}

func TestImportReferrer_OwnershipClaim_FailedWhenUnsigned(t *testing.T) {
	store := &mockStore{pullRecord: newTestRecordWithAnnotations(map[string]string{
		corev1.AnnotationKeyOwner: "did:web:acme.com",
	})}
	db := &mockDB{}

	claim := identityv1.NewOwnershipClaim("did:web:acme.com")

	referrer, err := claim.MarshalReferrer()
	require.NoError(t, err)

	_, err = New(store, db).ImportReferrer(t.Context(), "record-cid", referrer)

	require.NoError(t, err)
	require.Equal(t, 1, db.upsertOwnershipCalls)
	require.NotNil(t, db.lastOwnershipClaim)
	assert.Equal(t, "failed", db.lastOwnershipClaim.GetStatus())
	assert.NotEmpty(t, db.lastOwnershipClaim.GetError())
}

// generateSpiffeKeyCert returns a PEM-encoded ECDSA key, a self-signed
// certificate whose URI SAN is set to spiffeID, and the parsed certificate
// (usable as its own trust anchor, since it's self-signed).
func generateSpiffeKeyCert(t *testing.T, spiffeID string) (keyPEM, certPEM []byte, cert *x509.Certificate) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	spiffeURI, err := url.Parse(spiffeID)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{spiffeURI},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	parsedCert, err := x509.ParseCertificate(derBytes)
	require.NoError(t, err)

	privDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return keyPEM, certPEM, parsedCert
}

func TestImportReferrer_UnsupportedStore(t *testing.T) {
	// A store that implements StoreAPI but not ReferrerStoreAPI.
	_, err := New(&recordOnlyStore{}, &mockDB{}).ImportReferrer(t.Context(), "record-cid", &corev1.RecordReferrer{
		Type:      corev1.SignatureReferrerType,
		RecordRef: &corev1.RecordRef{Cid: "record-cid"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "referrer storage not supported")
}

func TestImportReferrer_PushError(t *testing.T) {
	store := &mockStore{pushReferrerErr: errors.New("nope")}
	db := &mockDB{}

	_, err := New(store, db).ImportReferrer(t.Context(), "record-cid", &corev1.RecordReferrer{
		Type:      corev1.SignatureReferrerType,
		RecordRef: &corev1.RecordRef{Cid: "record-cid"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to push referrer for record")
	assert.Equal(t, 0, db.setSignedCalls, "DB effects must not run when the referrer push fails")
}
