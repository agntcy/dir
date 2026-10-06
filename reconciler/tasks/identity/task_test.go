// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	didresolver "github.com/agntcy/dir/client/utils/identity/resolvers/did"
	dnsresolver "github.com/agntcy/dir/client/utils/identity/resolvers/dns"
	wellknownresolver "github.com/agntcy/dir/client/utils/identity/resolvers/wellknown"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/multiformats/go-multibase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordTemplate is a valid OASF record; each test gives it its own annotations.
const recordTemplate = `{
  "name": "example.com/agents/%s",
  "version": "v1.0.0",
  "schema_version": "0.8.0",
  "description": "A record that carries claims.",
  "authors": ["AGNTCY Contributors"],
  "created_at": "2025-03-19T17:06:37Z",
  "annotations": %s,
  "skills": [{"name": "natural_language_processing/natural_language_generation/text_completion", "id": 10201}],
  "locators": [{"type": "docker_image", "url": "https://ghcr.io/agntcy/example"}],
  "domains": [{"id": 301, "name": "life_science/biotechnology"}]
}`

// ---- fake store ----

// fakeStore serves records and referrers from memory. Like the real store it
// hands back every custom referrer of a record whatever type was asked for.
type fakeStore struct {
	types.StoreAPI
	types.ReferrerStoreAPI

	mu        sync.Mutex
	records   map[string]*corev1.Record
	referrers map[string][]*corev1.RecordReferrer
	walkErr   error
	pulls     int
}

func newFakeStore() *fakeStore {
	return &fakeStore{records: map[string]*corev1.Record{}, referrers: map[string][]*corev1.RecordReferrer{}}
}

func (s *fakeStore) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.pulls++

	rec, ok := s.records[ref.GetCid()]
	if !ok {
		return nil, errors.New("record not found")
	}

	return rec, nil
}

func (s *fakeStore) WalkReferrers(_ context.Context, cid, _ string, fn func(*corev1.RecordReferrer) error) error {
	s.mu.Lock()
	refs := append([]*corev1.RecordReferrer(nil), s.referrers[cid]...)
	err := s.walkErr
	s.mu.Unlock()

	if err != nil {
		return err
	}

	for _, ref := range refs {
		if err := fn(ref); err != nil {
			return err
		}
	}

	return nil
}

func (s *fakeStore) setReferrers(cid string, refs ...*corev1.RecordReferrer) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.referrers[cid] = refs
}

// ---- fixtures ----

type fixture struct {
	t     *testing.T
	db    types.DatabaseAPI
	store *fakeStore
	task  *Task
}

func newFixture(t *testing.T, config Config) *fixture {
	t.Helper()

	db, err := database.New(dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: "file::memory:"}})
	require.NoError(t, err)

	store := newFakeStore()
	task, err := NewTask(config, db, store, store)
	require.NoError(t, err)

	return &fixture{t: t, db: db, store: store, task: task}
}

// addRecord stores a record declaring the given annotations and returns its CID.
func (f *fixture) addRecord(name string, annotations map[string]string) string {
	f.t.Helper()

	encoded, err := json.Marshal(annotations)
	require.NoError(f.t, err)

	record, err := corev1.UnmarshalRecord([]byte(fmt.Sprintf(recordTemplate, name, encoded)))
	require.NoError(f.t, err)

	adapter, err := record.Decode()
	require.NoError(f.t, err)
	require.NoError(f.t, f.db.AddRecord(adapter))

	cid := record.GetCid()
	f.store.records[cid] = record

	return cid
}

func (f *fixture) run() {
	f.t.Helper()
	require.NoError(f.t, f.task.Run(context.Background()))
}

func (f *fixture) result(cid, role string) types.IdentityClaimObject {
	f.t.Helper()

	result, err := f.db.GetIdentityClaimByCID(cid, role)
	require.NoError(f.t, err)

	return result
}

func (f *fixture) noResult(cid, role string) {
	f.t.Helper()

	_, err := f.db.GetIdentityClaimByCID(cid, role)
	require.ErrorIs(f.t, err, gormdb.ErrIdentityClaimNotFound)
}

func newSigner(t *testing.T, key crypto.PrivateKey) jws.Signer {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	signer, err := jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
	require.NoError(t, err)

	return signer
}

// signedReferrer signs a claim of the given role for cid, and returns it as the
// referrer a push would store.
func signedReferrer(t *testing.T, role identityv1.ClaimRole, cid, subject string, signer jws.Signer, opts ...clientidentity.SignOption) *corev1.RecordReferrer {
	t.Helper()

	claim := &identityv1.Claim{Role: role, Subject: subject}
	require.NoError(t, clientidentity.Sign(claim, cid, signer, opts...))

	referrer, err := claim.MarshalReferrer()
	require.NoError(t, err)

	return referrer
}

func spkiBase64(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(der)
}

func didKey(pub ed25519.PublicKey) string {
	encoded, err := multibase.Encode(multibase.Base58BTC, append([]byte{0xed, 0x01}, pub...))
	if err != nil {
		panic(err)
	}

	return "did:key:" + encoded
}

// fakeFetcher serves documents by URL.
type fakeFetcher struct {
	mu   sync.Mutex
	docs map[string][]byte
}

func (f *fakeFetcher) Get(_ context.Context, u string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if doc, ok := f.docs[u]; ok {
		return doc, nil
	}

	return nil, fmt.Errorf("no document at %s", u)
}

func (f *fakeFetcher) set(u string, doc []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.docs[u] = doc
}

func jwksFor(t *testing.T, keys ...crypto.PublicKey) []byte {
	t.Helper()

	set := jwk.NewSet()

	for _, k := range keys {
		key, err := jwk.FromRaw(k)
		require.NoError(t, err)
		require.NoError(t, set.AddKey(key))
	}

	doc, err := json.Marshal(set)
	require.NoError(t, err)

	return doc
}

// countingResolver counts lookups and answers with keys.
type countingResolver struct {
	mu    sync.Mutex
	keys  []crypto.PublicKey
	err   error
	calls int
}

func (c *countingResolver) Resolve(context.Context, string, []byte) ([]crypto.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.calls++

	return c.keys, c.err
}

var _ resolvers.Resolver = (*countingResolver)(nil)

// ---- SPIFFE fixtures ----

type testCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newCA(t *testing.T, name string) *testCA {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &testCA{cert: cert, key: key}
}

func (ca *testCA) bundlePEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
}

// issue returns an SVID for spiffeID over key's public half, signed by the CA.
func (ca *testCA) issue(t *testing.T, key *ecdsa.PrivateKey, notAfter time.Time) []byte {
	t.Helper()

	uri, err := url.Parse(spiffeID)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// ---- tests ----

// spiffeID is the SPIFFE ID of every SVID the tests issue.
const spiffeID = "spiffe://acme.com/agents/finance"

const (
	identityRole = identityv1.ClaimRole_CLAIM_ROLE_IDENTITY
	ownerRole    = identityv1.ClaimRole_CLAIM_ROLE_OWNER
)

// A claim that has just been pushed is verified by one run, for every scheme,
// against real key and certificate material.
func TestRun_VerifiesAClaimOfEveryScheme(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	edPub, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	ca := newCA(t, "acme root")
	bundleFile := filepath.Join(t.TempDir(), "acme.pem")
	require.NoError(t, os.WriteFile(bundleFile, ca.bundlePEM(), 0o600))

	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json": jwksFor(t, &ecKey.PublicKey),
	}}

	didDoc, err := json.Marshal(map[string]any{
		"id": "did:web:acme.com",
		"verificationMethod": []map[string]any{{
			"id": "did:web:acme.com#key", "publicKeyJwk": json.RawMessage(jwksKey(t, &ecKey.PublicKey)),
		}},
		"assertionMethod": []string{"did:web:acme.com#key"},
	})
	require.NoError(t, err)
	fetcher.set("https://acme.com/.well-known/did.json", didDoc)

	tests := map[string]struct {
		subject string
		signer  jws.Signer
		opts    []clientidentity.SignOption
	}{
		"dns":         {"dns:acme.com", newSigner(t, ecKey), nil},
		"bare domain": {"acme.com", newSigner(t, ecKey), nil},
		"https":       {"https://acme.com", newSigner(t, ecKey), nil},
		"did:web":     {"did:web:acme.com", newSigner(t, ecKey), nil},
		"did:key":     {didKey(edPub), newSigner(t, edKey), nil},
		"spiffe": {spiffeID, newSigner(t, ecKey), []clientidentity.SignOption{
			clientidentity.WithCertificate(ca.issue(t, ecKey, time.Now().Add(time.Hour))),
		}},
	}

	for name, tt := range tests {
		for _, role := range []struct {
			role       identityv1.ClaimRole
			stored     string
			annotation string
		}{
			{identityRole, types.ClaimRoleIdentity, corev1.AnnotationKeyIdentity},
			{ownerRole, types.ClaimRoleOwner, corev1.AnnotationKeyOwner},
		} {
			t.Run(name+"/"+role.stored, func(t *testing.T) {
				f := newFixture(t, Config{SPIFFETrustBundles: []TrustBundle{{TrustDomain: "acme.com", BundleFile: bundleFile}}})
				f.task.network = resolverSet{
					dns: dnsresolver.New(dnsresolver.WithLookupTXT(func(_ context.Context, host string) ([]string, error) {
						require.Equal(t, "_agntcy-key.acme.com", host)

						return []string{"v=akv1;key=" + spkiBase64(t, &ecKey.PublicKey)}, nil
					})),
					did:       didresolver.New(fetcher),
					wellknown: wellknownresolver.New(fetcher),
				}

				cid := f.addRecord(name+role.stored, map[string]string{role.annotation: tt.subject})
				f.store.setReferrers(cid, signedReferrer(t, role.role, cid, tt.subject, tt.signer, tt.opts...))

				f.noResult(cid, role.stored) // nothing verified until the task has run
				f.run()

				result := f.result(cid, role.stored)
				assert.Equal(t, types.ClaimStatusVerified, result.GetStatus(), result.GetError())
				assert.Equal(t, tt.subject, result.GetSubject())
				assert.Empty(t, result.GetError())
				assert.WithinDuration(t, time.Now(), result.GetVerifiedAt(), time.Minute)
			})
		}
	}
}

func jwksKey(t *testing.T, pub crypto.PublicKey) []byte {
	t.Helper()

	key, err := jwk.FromRaw(pub)
	require.NoError(t, err)

	doc, err := json.Marshal(key)
	require.NoError(t, err)

	return doc
}

// A claim whose key material changes flips from verified to failed on the next
// run, and back once it is restored.
func TestRun_RotatedKeyFlipsTheResult(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	rotated, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	published := &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: published}

	cid := f.addRecord("rotated", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
	f.store.setReferrers(cid, signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, key)))

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	// The domain publishes a new key: the old claim no longer verifies.
	published.keys = []crypto.PublicKey{&rotated.PublicKey}

	f.run()

	result := f.result(cid, types.ClaimRoleOwner)
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
	assert.Contains(t, result.GetError(), "signature does not verify")

	// Back to the key it was signed with.
	published.keys = []crypto.PublicKey{&key.PublicKey}

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	// The subject stops publishing a key at all.
	published.keys, published.err = nil, resolvers.ErrNoKeys

	f.run()

	result = f.result(cid, types.ClaimRoleOwner)
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
	assert.Contains(t, result.GetError(), "no usable public keys")
}

// A revoked trust bundle, and a certificate that has expired, are caught on the
// next run.
func TestRun_SPIFFEBundleAndCertificateChanges(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ca := newCA(t, "acme root")
	otherCA := newCA(t, "someone else")
	bundleFile := filepath.Join(t.TempDir(), "acme.pem")
	require.NoError(t, os.WriteFile(bundleFile, ca.bundlePEM(), 0o600))

	f := newFixture(t, Config{SPIFFETrustBundles: []TrustBundle{{TrustDomain: "acme.com", BundleFile: bundleFile}}})

	cid := f.addRecord("spiffe", map[string]string{corev1.AnnotationKeyIdentity: spiffeID})
	f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, spiffeID, newSigner(t, key),
		clientidentity.WithCertificate(ca.issue(t, key, time.Now().Add(time.Hour)))))

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus())

	// The bundle now trusts a different CA, so the SVID no longer chains to it.
	require.NoError(t, os.WriteFile(bundleFile, otherCA.bundlePEM(), 0o600))

	f.run()

	result := f.result(cid, types.ClaimRoleIdentity)
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
	assert.Contains(t, result.GetError(), "verify SVID")

	// Trusted again, but the claim carries a certificate that has run out.
	require.NoError(t, os.WriteFile(bundleFile, ca.bundlePEM(), 0o600))

	expired := f.addRecord("spiffe-expired", map[string]string{corev1.AnnotationKeyIdentity: spiffeID})
	f.store.setReferrers(expired, signedReferrer(t, identityRole, expired, spiffeID, newSigner(t, key),
		clientidentity.WithCertificate(ca.issue(t, key, time.Now().Add(-time.Second)))))

	f.run()

	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus(), "restored")
	assert.Equal(t, types.ClaimStatusFailed, f.result(expired, types.ClaimRoleIdentity).GetStatus(), "expired")
}

func TestRun_SPIFFEFailsClosedWithoutATrustBundle(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ca := newCA(t, "acme root")

	for name, config := range map[string]Config{
		"no bundle configured": {},
		"bundle file missing":  {SPIFFETrustBundles: []TrustBundle{{TrustDomain: "acme.com", BundleFile: filepath.Join(t.TempDir(), "nope.pem")}}},
		"other trust domain":   {SPIFFETrustBundles: []TrustBundle{{TrustDomain: "other.org", BundleFile: writeBundle(t, ca)}}},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, config)

			cid := f.addRecord("spiffe", map[string]string{corev1.AnnotationKeyIdentity: spiffeID})
			f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, spiffeID, newSigner(t, key),
				clientidentity.WithCertificate(ca.issue(t, key, time.Now().Add(time.Hour)))))

			f.run() // a bundle that cannot be read does not fail the run

			assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleIdentity).GetStatus())
		})
	}
}

func writeBundle(t *testing.T, ca *testCA) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "bundle.pem")
	require.NoError(t, os.WriteFile(path, ca.bundlePEM(), 0o600))

	return path
}

// A claim that names a subject the record does not declare says nothing about the
// record: it costs no lookup and leaves no result, so it cannot put its own subject
// in the search index or turn a record without a claim into a failed one.
func TestRun_ClaimForAnotherSubjectLeavesNoResult(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	lookups := &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}
	signer := newSigner(t, key)

	tests := map[string]struct {
		annotations map[string]string
	}{
		"record declares no identity": {nil},
		"record declares another one": {map[string]string{corev1.AnnotationKeyIdentity: "dns:other.com"}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, Config{})
			f.task.network = resolverSet{dns: lookups}

			cid := f.addRecord("unmatched", tt.annotations)
			f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, "dns:acme.com", signer))

			f.run()

			f.noResult(cid, types.ClaimRoleIdentity)
		})
	}

	assert.Zero(t, lookups.calls, "no key lookup for a claim of another subject")
}

// A result left by a claim that no longer applies is removed with it.
func TestRun_RemovesTheResultOfAClaimForAnotherSubject(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}}

	cid := f.addRecord("stale", map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"})
	f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, "dns:other.com", newSigner(t, key)))

	require.NoError(t, f.db.UpsertIdentityClaim(&gormdb.IdentityClaim{
		RecordCID: cid, Role: types.ClaimRoleIdentity, Subject: "dns:other.com",
		Status: types.ClaimStatusFailed, VerifiedAt: time.Now(),
	}))

	f.run()

	f.noResult(cid, types.ClaimRoleIdentity)
}

// A certificate is no part of a claim for anything but a spiffe:// or ans://
// subject, so one grafted onto such a claim fails it instead of decorating a
// verified result.
func TestRun_CertificateOnANonSPIFFEClaimFails(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ca := newCA(t, "acme root")

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}}

	cid := f.addRecord("grafted", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})

	claim := &identityv1.Claim{Role: ownerRole, Subject: "dns:acme.com"}
	require.NoError(t, clientidentity.Sign(claim, cid, newSigner(t, key)))

	block, _ := pem.Decode(ca.issue(t, key, time.Now().Add(time.Hour)))
	require.NotNil(t, block)

	graft := base64.StdEncoding.EncodeToString(block.Bytes)
	claim.Certificate = &graft

	referrer, err := claim.MarshalReferrer()
	require.NoError(t, err)

	f.store.setReferrers(cid, referrer)
	f.run()

	result := f.result(cid, types.ClaimRoleOwner)
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
	assert.Contains(t, result.GetError(), "only used for spiffe:// and ans:// subjects")
}

func TestRun_ClaimOfAnotherRecordFails(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}}

	other := f.addRecord("other", map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"})
	cid := f.addRecord("victim", map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"})

	// A claim signed for one record, attached to another.
	f.store.setReferrers(cid, signedReferrer(t, identityRole, other, "dns:acme.com", newSigner(t, key)))

	f.run()

	result := f.result(cid, types.ClaimRoleIdentity)
	assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
	assert.Contains(t, result.GetError(), "record_cid")
}

func TestRun_UnsupportedSubjectScheme(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	for _, subject := range []string{"http://acme.com", "mailto:ops@acme.com", "ftp://acme.com"} {
		t.Run(subject, func(t *testing.T) {
			f := newFixture(t, Config{})

			cid := f.addRecord("unsupported", map[string]string{corev1.AnnotationKeyIdentity: subject})
			f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, subject, newSigner(t, key)))

			f.run()

			result := f.result(cid, types.ClaimRoleIdentity)
			assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
			assert.Contains(t, result.GetError(), "unsupported subject scheme")
		})
	}
}

// Anyone can attach a claim to a record, so a claim that does not verify must
// not take the place of one that does.
func TestRun_AVerifiedClaimWinsOverOthers(t *testing.T) {
	good, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	attacker, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&good.PublicKey}}}

	cid := f.addRecord("contested", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
	forged := signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, attacker))
	valid := signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, good))

	// Whichever order they are walked in.
	for _, refs := range [][]*corev1.RecordReferrer{{forged, valid}, {valid, forged}} {
		f.store.setReferrers(cid, refs...)
		f.run()

		assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())
	}

	// With only the forged one left, it fails.
	f.store.setReferrers(cid, forged)
	f.run()

	assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleOwner).GetStatus())
}

func TestRun_ReportsTheNewestFailure(t *testing.T) {
	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{err: errors.New("lookup timed out")}}

	cid := f.addRecord("failing", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})

	// Both fail, for different reasons, so the result shows which claim it reports.
	expired := "2020-01-01T00:00:00Z"
	older := &identityv1.Claim{Role: ownerRole, Subject: "dns:acme.com", SignedAt: "2026-01-01T00:00:00Z", ExpiresAt: &expired}
	newer := &identityv1.Claim{Role: ownerRole, Subject: "dns:acme.com", SignedAt: "2026-02-01T00:00:00Z"}

	referrer := func(claim *identityv1.Claim) *corev1.RecordReferrer {
		claim.RecordCid = cid
		claim.Signature = "unused"

		ref, err := claim.MarshalReferrer()
		require.NoError(t, err)

		return ref
	}

	olderRef, newerRef := referrer(older), referrer(newer)

	for _, refs := range [][]*corev1.RecordReferrer{{olderRef, newerRef}, {newerRef, olderRef}} {
		f.store.setReferrers(cid, refs...)
		f.run()

		result := f.result(cid, types.ClaimRoleOwner)
		assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
		assert.Contains(t, result.GetError(), "lookup timed out")
		assert.NotContains(t, result.GetError(), "expired")
	}
}

// signed_at is written by whoever signs the claim, so a date in the future or one
// that does not parse must not let a claim outrank an honest one.
func TestRun_ForgedSignedAtDoesNotOutrankAnHonestFailure(t *testing.T) {
	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{err: errors.New("lookup timed out")}}

	cid := f.addRecord("forged-time", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})

	expired := "2020-01-01T00:00:00Z"
	claims := map[string]*identityv1.Claim{
		"honest":     {Role: ownerRole, Subject: "dns:acme.com", SignedAt: "2026-01-01T00:00:00Z", ExpiresAt: &expired},
		"future":     {Role: ownerRole, Subject: "dns:acme.com", SignedAt: "2999-01-01T00:00:00Z"},
		"unparsable": {Role: ownerRole, Subject: "dns:acme.com", SignedAt: "next tuesday"},
	}

	referrers := map[string]*corev1.RecordReferrer{}

	for name, claim := range claims {
		claim.RecordCid = cid
		claim.Signature = "unused"

		ref, err := claim.MarshalReferrer()
		require.NoError(t, err)

		referrers[name] = ref
	}

	for _, order := range [][]string{{"honest", "future", "unparsable"}, {"unparsable", "future", "honest"}} {
		var refs []*corev1.RecordReferrer
		for _, name := range order {
			refs = append(refs, referrers[name])
		}

		f.store.setReferrers(cid, refs...)
		f.run()

		result := f.result(cid, types.ClaimRoleOwner)
		assert.Equal(t, types.ClaimStatusFailed, result.GetStatus())
		assert.Contains(t, result.GetError(), "expired", order)
	}
}

// Only referrers of the right type, asserting the right role, are claims.
func TestRun_IgnoresReferrersThatAreNotClaimsOfTheRole(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	lookups := &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: lookups}

	cid := f.addRecord("mixed", map[string]string{
		corev1.AnnotationKeyIdentity: "dns:acme.com",
		corev1.AnnotationKeyOwner:    "dns:acme.com",
	})

	signer := newSigner(t, key)

	// An owner claim stored as an identity claim: valid signature, subject that
	// matches the identity annotation, but it asserts ownership.
	misfiled := signedReferrer(t, ownerRole, cid, "dns:acme.com", signer)
	misfiled.Type = corev1.IdentityClaimReferrerType

	// And an identity claim stored under the ownership type.
	other := signedReferrer(t, identityRole, cid, "dns:acme.com", signer)
	other.Type = corev1.OwnershipClaimReferrerType

	scanReport := &corev1.RecordReferrer{Type: corev1.ScanReportReferrerType}
	garbage := &corev1.RecordReferrer{Type: corev1.IdentityClaimReferrerType}

	f.store.setReferrers(cid, misfiled, other, scanReport, garbage)
	f.run()

	f.noResult(cid, types.ClaimRoleIdentity)
	f.noResult(cid, types.ClaimRoleOwner)
	assert.Zero(t, lookups.calls)
}

func TestRun_RecordsWithoutClaimsAreNotReadFromTheStore(t *testing.T) {
	f := newFixture(t, Config{})

	cid := f.addRecord("plain", map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"})
	f.run()

	f.noResult(cid, types.ClaimRoleIdentity)
	f.noResult(cid, types.ClaimRoleOwner)
	assert.Zero(t, f.store.pulls, "a record is only pulled once it is known to carry a claim")
}

// A claim that is no longer attached to the record takes its result with it.
func TestRun_RemovesTheResultOfAClaimThatIsGone(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}}

	cid := f.addRecord("withdrawn", map[string]string{
		corev1.AnnotationKeyIdentity: "dns:acme.com",
		corev1.AnnotationKeyOwner:    "dns:acme.com",
	})

	signer := newSigner(t, key)
	identityClaim := signedReferrer(t, identityRole, cid, "dns:acme.com", signer)
	ownerClaim := signedReferrer(t, ownerRole, cid, "dns:acme.com", signer)

	f.store.setReferrers(cid, identityClaim, ownerClaim)
	f.run()

	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus())
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	f.store.setReferrers(cid, ownerClaim)
	f.run()

	f.noResult(cid, types.ClaimRoleIdentity)
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())
}

// When the claims cannot be read, what was stored is left alone.
func TestRun_KeepsTheResultWhenTheStoreFails(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}}

	cid := f.addRecord("flaky", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
	f.store.setReferrers(cid, signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, key)))

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	f.store.walkErr = errors.New("registry unavailable")
	f.run()

	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())
}

func TestRun_ReadsEveryRecordAndLooksUpASharedSubjectOnce(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	lookups := &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: lookups}

	signer := newSigner(t, key)

	var cids []string

	for i := range 5 {
		cid := f.addRecord(fmt.Sprintf("batch%d", i), map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
		f.store.setReferrers(cid, signedReferrer(t, ownerRole, cid, "dns:acme.com", signer))

		cids = append(cids, cid)
	}

	f.run()

	for _, cid := range cids {
		assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus(), cid)
	}

	assert.Equal(t, 1, lookups.calls, "one lookup for the subject all five records share")
}

func TestRun_AnUnreachableSubjectKeepsTheLastResultForAWhile(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	lookups := &countingResolver{keys: []crypto.PublicKey{&key.PublicKey}}

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: lookups}

	cid := f.addRecord("flaky-subject", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
	f.store.setReferrers(cid, signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, key)))

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	// The subject cannot be reached: that says nothing about the claim.
	lookups.keys, lookups.err = nil, &net.OpError{Op: "dial", Err: errors.New("connection refused")}

	f.run()
	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleOwner).GetStatus())

	// An answer that says the subject publishes no key does.
	lookups.err = resolvers.ErrNoKeys

	f.run()
	assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleOwner).GetStatus())

	// Unreachable for longer than the grace period stops being tolerated.
	require.NoError(t, f.db.UpsertIdentityClaim(&gormdb.IdentityClaim{
		RecordCID: cid, Role: types.ClaimRoleOwner, Subject: "dns:acme.com",
		Status: types.ClaimStatusVerified, VerifiedAt: time.Now().Add(-staleGrace - time.Hour),
	}))

	lookups.err = context.DeadlineExceeded

	f.run()
	assert.Equal(t, types.ClaimStatusFailed, f.result(cid, types.ClaimRoleOwner).GetStatus())
}

func TestRun_OneUnreadableBundleDoesNotFailOtherTrustDomains(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ca := newCA(t, "acme root")

	f := newFixture(t, Config{SPIFFETrustBundles: []TrustBundle{
		{TrustDomain: "other.org", BundleFile: filepath.Join(t.TempDir(), "nope.pem")},
		{TrustDomain: "acme.com", BundleFile: writeBundle(t, ca)},
	}})

	cid := f.addRecord("spiffe", map[string]string{corev1.AnnotationKeyIdentity: spiffeID})
	f.store.setReferrers(cid, signedReferrer(t, identityRole, cid, spiffeID, newSigner(t, key),
		clientidentity.WithCertificate(ca.issue(t, key, time.Now().Add(time.Hour)))))

	f.run()

	assert.Equal(t, types.ClaimStatusVerified, f.result(cid, types.ClaimRoleIdentity).GetStatus())
}

func TestIsTransient(t *testing.T) {
	wrapped := func(err error) error { return fmt.Errorf("fetch JWKS: %w", err) }

	for name, tt := range map[string]struct {
		err  error
		want bool
	}{
		"deadline":           {wrapped(context.DeadlineExceeded), true},
		"connection refused": {wrapped(&net.OpError{Op: "dial", Err: errors.New("refused")}), true},
		"dns timeout":        {wrapped(&net.DNSError{Err: "i/o timeout", IsTimeout: true}), true},
		"dns server failure": {wrapped(&net.DNSError{Err: "server misbehaving"}), true},
		"500":                {wrapped(&safefetch.StatusError{Code: 500}), true},
		"429":                {wrapped(&safefetch.StatusError{Code: 429}), true},
		"dns no such host":   {wrapped(&net.DNSError{Err: "no such host", IsNotFound: true}), false},
		"404":                {wrapped(&safefetch.StatusError{Code: 404}), false},
		"disallowed address": {wrapped(fmt.Errorf("%w: x", safefetch.ErrDisallowedAddress)), false},
		"no keys":            {wrapped(resolvers.ErrNoKeys), false},
		"anything else":      {errors.New("parse JWKS"), false},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, isTransient(tt.err))
		})
	}
}

func TestRun_StopsWhenCancelled(t *testing.T) {
	f := newFixture(t, Config{})
	f.addRecord("one", nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.ErrorIs(t, f.task.Run(ctx), context.Canceled)
}

func TestRun_TruncatesLongFailureReasons(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	f := newFixture(t, Config{})
	f.task.network = resolverSet{dns: &countingResolver{err: errors.New(string(make([]byte, 5000)))}}

	cid := f.addRecord("verbose", map[string]string{corev1.AnnotationKeyOwner: "dns:acme.com"})
	f.store.setReferrers(cid, signedReferrer(t, ownerRole, cid, "dns:acme.com", newSigner(t, key)))

	f.run()

	assert.LessOrEqual(t, len(f.result(cid, types.ClaimRoleOwner).GetError()), maxErrorLength)
}

func TestForSubject(t *testing.T) {
	set := resolverSet{
		dns:       &countingResolver{},
		did:       &countingResolver{},
		wellknown: &countingResolver{},
		spiffe:    &countingResolver{},
	}

	for subject, want := range map[string]resolvers.Resolver{
		"dns:acme.com":                     set.dns,
		"acme.com":                         set.dns,
		"did:web:acme.com":                 set.did,
		"did:key:z6Mk":                     set.did,
		"https://acme.com/agents":          set.wellknown,
		"spiffe://acme.com/agents/finance": set.spiffe,
	} {
		got, err := set.forSubject(subject)
		require.NoError(t, err, subject)
		assert.Same(t, want, got, subject)
	}

	for _, subject := range []string{"http://acme.com", "ftp://acme.com", "mailto:a@b.c", "acme.com:8080", ""} {
		_, err := set.forSubject(subject)
		if subject == "" {
			// An empty subject has no scheme and reads as a bare domain; the key-free
			// checks reject it before it is ever resolved.
			require.NoError(t, err)

			continue
		}

		require.ErrorContains(t, err, "unsupported subject scheme", subject)
	}
}

func TestTaskMetadata(t *testing.T) {
	task, err := NewTask(Config{}, nil, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "identity", task.Name())
	assert.False(t, task.IsEnabled(), "off unless configured")
	assert.Equal(t, DefaultInterval, task.Interval())

	task, err = NewTask(Config{Enabled: true, Interval: time.Minute}, nil, nil, nil)
	require.NoError(t, err)

	assert.True(t, task.IsEnabled())
	assert.Equal(t, time.Minute, task.Interval())
}

func TestConfig(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, DefaultInterval, cfg.GetInterval())
	assert.Equal(t, DefaultRecordTimeout, cfg.GetRecordTimeout())
	assert.Empty(t, cfg.trustDomains())

	cfg = Config{
		Interval:      time.Minute,
		RecordTimeout: time.Second,
		SPIFFETrustBundles: []TrustBundle{
			{TrustDomain: "acme.com", BundleFile: "/a.pem"},
			{TrustDomain: "other.org", BundleFile: "/b.pem"},
		},
	}
	assert.Equal(t, time.Minute, cfg.GetInterval())
	assert.Equal(t, time.Second, cfg.GetRecordTimeout())
	assert.Equal(t, map[string]string{"acme.com": "/a.pem", "other.org": "/b.pem"}, cfg.trustDomains())
}
