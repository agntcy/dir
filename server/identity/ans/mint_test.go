// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/fxamacker/cbor/v2"
	"github.com/lestrrat-go/jwx/v2/cert"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The synthetic agent, log, and claim every test starts from.
const (
	testHost      = "agent.example.com"
	testVersion   = "v1.0.0"
	testAnsName   = "ans://v1.0.0.agent.example.com"
	testAgentID   = "5b1b6cc4-4b3e-4d4e-9a7d-2c1e7a6f9a10"
	otherAgentID  = "0e4c1d2a-7f6b-4c3d-8e9f-a1b2c3d4e5f6"
	testLogHost   = "log.example.com"
	testLogBase   = "https://log.example.com"
	testBadgeURL  = testLogBase + "/v1/agents/" + testAgentID
	testRecordCID = "baguqeerawgtdigwzhkk5ct5m6ozysc6onfjnrmr4kjmmxuewqyezs6qhiiq"
	testSignedAt  = "2026-09-11T12:00:00Z"

	// testOrigin is the origin the synthetic log claims as receipt issuer.
	testOrigin = "example-log"

	// oversizedPadding grows a certificate past the size verifiers accept.
	oversizedPadding = 17 << 10
)

var testNow = time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)

// testLog is a synthetic transparency log: one ES256 signing key, its 4-byte
// key id, and the origin its receipts claim as issuer. Every artifact it mints
// follows the layouts the SDK's own tests verify against.
type testLog struct {
	key    *ecdsa.PrivateKey
	kid    [4]byte
	origin string
}

func mintLog(t *testing.T) *testLog {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	sum := sha256.Sum256(spkiOf(t, &key.PublicKey))

	var kid [4]byte

	copy(kid[:], sum[:4])

	return &testLog{key: key, kid: kid, origin: testOrigin}
}

// rootKeyLine renders the log's verification key as one /root-keys line:
// origin+hex(kid)+base64(0x02 || SPKI DER).
func (l *testLog) rootKeyLine(t *testing.T) string {
	t.Helper()

	spki := spkiOf(t, &l.key.PublicKey)

	return l.origin + "+" + hex.EncodeToString(l.kid[:]) + "+" + base64.StdEncoding.EncodeToString(append([]byte{0x02}, spki...))
}

// tokenClaims are the status-token payload fields a test controls.
type tokenClaims struct {
	agentID       string
	ansName       string
	status        string
	iat           int64
	exp           int64
	identityCerts []string
}

// statusToken mints a COSE_Sign1 status token with the int-keyed payload the
// reference log emits: fingerprints as "SHA256:<hex>" strings.
func (l *testLog) statusToken(t *testing.T, claims tokenClaims) []byte {
	t.Helper()

	certs := make([]map[int64]any, 0, len(claims.identityCerts))
	for _, fp := range claims.identityCerts {
		certs = append(certs, map[int64]any{1: fp, 2: "X509-OV-CLIENT"})
	}

	payload := map[int64]any{
		1: claims.agentID,
		2: claims.status,
		3: claims.iat,
		4: claims.exp,
		5: claims.ansName,
		6: certs,
	}

	protected := map[int64]any{
		1: int64(-7),
		3: "application/ans-status-token+cbor",
		4: l.kid[:],
	}

	return l.coseSign1(t, protected, map[int64]any{}, mustCBOR(t, payload))
}

// receipt mints a COSE_Sign1 receipt over event with the inclusion proof in
// the unprotected header. An empty path with treeSize 1 is a single-leaf tree.
func (l *testLog) receipt(t *testing.T, event []byte, treeSize, leafIndex uint64, path [][]byte, iat int64) []byte {
	t.Helper()

	protected := map[int64]any{
		1:   int64(-7),
		4:   l.kid[:],
		395: int64(1),
		15:  map[int64]any{1: l.origin, 6: iat},
	}

	if path == nil {
		path = [][]byte{}
	}

	unprotected := map[int64]any{
		396: map[int64]any{-1: treeSize, -2: leafIndex, -3: path},
	}

	return l.coseSign1(t, protected, unprotected, event)
}

// coseSign1 signs payload under the protected header with ES256 and returns
// the tag-18 COSE_Sign1 encoding. r and s are written with FillBytes so the
// P1363 signature is always 64 bytes.
func (l *testLog) coseSign1(t *testing.T, protected, unprotected map[int64]any, payload []byte) []byte {
	t.Helper()

	protectedBytes := mustCBOR(t, protected)

	digest, err := scitt.ComputeSigStructureDigest(protectedBytes, payload)
	require.NoError(t, err)

	r, s, err := ecdsa.Sign(rand.Reader, l.key, digest[:])
	require.NoError(t, err)

	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return mustCBOR(t, cbor.Tag{Number: 18, Content: []any{protectedBytes, unprotected, payload, sig}})
}

func mustCBOR(t *testing.T, value any) []byte {
	t.Helper()

	encoded, err := cbor.Marshal(value)
	require.NoError(t, err)

	return encoded
}

func spkiOf(t *testing.T, pub crypto.PublicKey) []byte {
	t.Helper()

	spki, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return spki
}

// identityCert is a minted ANS identity certificate with its signing key.
type identityCert struct {
	der         []byte
	cert        *x509.Certificate
	key         crypto.Signer
	fingerprint string
}

// mintIdentityCert mints a self-signed P-256 identity certificate whose only
// SAN is the ANS name URI.
func mintIdentityCert(t *testing.T, ansName string, notBefore, notAfter time.Time) identityCert {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return mintIdentityCertWithKey(t, ansName, notBefore, notAfter, key)
}

func mintIdentityCertWithKey(t *testing.T, ansName string, notBefore, notAfter time.Time, key crypto.Signer) identityCert {
	t.Helper()

	return signCertificate(t, certTemplate(t, notBefore, notAfter, ansName), key)
}

// mintIdentityCertWithSANs mints a P-256 identity certificate carrying every
// URI in uris as a SAN, valid around testNow.
func mintIdentityCertWithSANs(t *testing.T, uris ...string) identityCert {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return signCertificate(t, certTemplate(t, testNow.Add(-time.Hour), testNow.Add(time.Hour), uris...), key)
}

// mintOversizedIdentityCert mints an identity certificate padded past the
// size verifiers accept.
func mintOversizedIdentityCert(t *testing.T, ansName string) identityCert {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := certTemplate(t, testNow.Add(-time.Hour), testNow.Add(time.Hour), ansName)
	template.ExtraExtensions = []pkix.Extension{{
		Id:    asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1},
		Value: make([]byte, oversizedPadding),
	}}

	return signCertificate(t, template, key)
}

func certTemplate(t *testing.T, notBefore, notAfter time.Time, uris ...string) *x509.Certificate {
	t.Helper()

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 62))
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "agent"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	for _, raw := range uris {
		uri, err := url.Parse(raw)
		require.NoError(t, err)

		template.URIs = append(template.URIs, uri)
	}

	return template
}

func signCertificate(t *testing.T, template *x509.Certificate, key crypto.Signer) identityCert {
	t.Helper()

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return identityCert{der: der, cert: parsed, key: key, fingerprint: verify.CertFingerprintFromDER(der).String()}
}

// signWithChain mints the compact detached-payload JWS a claim carries: signed
// by key, with certDER in the protected x5c header when it is not nil. It
// calls jwx directly so cases the production signer refuses by construction
// can still be minted.
func signWithChain(t *testing.T, key crypto.Signer, certDER []byte, payload []byte) string {
	t.Helper()

	if certDER == nil {
		return mintJWS(t, key, jws.NewHeaders(), payload)
	}

	return mintJWS(t, key, x5cHeaders(t, certDER), payload)
}

// x5cHeaders returns protected headers whose x5c lists every certificate in
// ders, each as the standard base64 of its DER.
func x5cHeaders(t *testing.T, ders ...[]byte) jws.Headers {
	t.Helper()

	var chain cert.Chain
	for _, der := range ders {
		require.NoError(t, chain.AddString(base64.StdEncoding.EncodeToString(der)))
	}

	hdrs := jws.NewHeaders()
	require.NoError(t, hdrs.Set(jws.X509CertChainKey, &chain))

	return hdrs
}

// mintJWS signs payload as a compact detached-payload JWS under hdrs.
func mintJWS(t *testing.T, key crypto.Signer, hdrs jws.Headers, payload []byte) string {
	t.Helper()

	sig, err := jws.Sign(nil, jws.WithKey(jwsAlgorithm(t, key), key, jws.WithProtectedHeaders(hdrs)), jws.WithDetachedPayload(payload))
	require.NoError(t, err)

	return string(sig)
}

// jwsAlgorithm names the JWS algorithm for key's type, including ES512, which
// the production code refuses.
func jwsAlgorithm(t *testing.T, key crypto.Signer) jwa.SignatureAlgorithm {
	t.Helper()

	switch pub := key.Public().(type) {
	case *ecdsa.PublicKey:
		switch pub.Curve {
		case elliptic.P256():
			return jwa.ES256
		case elliptic.P384():
			return jwa.ES384
		case elliptic.P521():
			return jwa.ES512
		}
	case ed25519.PublicKey:
		return jwa.EdDSA
	case *rsa.PublicKey:
		return jwa.RS256
	}

	t.Fatalf("no JWS algorithm for a %T key", key.Public())

	return ""
}

// eventJSON renders an ANS event envelope naming the agent under idKey
// ("ansId" for the reference log, "agentId" for the older shape).
func eventJSON(t *testing.T, idKey, agentID, ansName string) []byte {
	t.Helper()

	envelope := map[string]any{
		"payload": map[string]any{
			"logId": "0192a3b4-c5d6-7e8f-9a0b-1c2d3e4f5a6b",
			"producer": map[string]any{
				"event": map[string]any{
					idKey:       agentID,
					"ansName":   ansName,
					"eventType": "AGENT_REGISTERED",
				},
				"keyId":     "ra-key",
				"signature": "detached-jws",
			},
		},
		"schemaVersion": "V2",
	}

	encoded, err := json.Marshal(envelope)
	require.NoError(t, err)

	return encoded
}

// TestMintedFixturesVerify checks the minting helpers against the SDK
// verifiers, so a helper regression cannot masquerade as a resolver bug.
func TestMintedFixturesVerify(t *testing.T) {
	log := mintLog(t)
	cert := mintIdentityCert(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour))

	keys, err := scitt.NewKeyStore([]string{log.rootKeyLine(t)})
	require.NoError(t, err)

	token := log.statusToken(t, tokenClaims{
		agentID:       testAgentID,
		ansName:       testAnsName,
		status:        "ACTIVE",
		iat:           testNow.Unix() - 60,
		exp:           testNow.Unix() + 3600,
		identityCerts: []string{cert.fingerprint},
	})

	verified, err := scitt.VerifyStatusTokenAt(token, keys, 0, testNow.Unix())
	require.NoError(t, err)
	assert.True(t, scitt.MatchesIdentityCert(&verified.Payload, verify.CertFingerprintFromDER(cert.der).Bytes()), "minted token does not attest the minted certificate")

	event := eventJSON(t, "ansId", testAgentID, testAnsName)

	receipt, err := scitt.VerifyReceipt(log.receipt(t, event, 1, 0, nil, testNow.Unix()), keys)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), receipt.TreeSize)
	assert.Equal(t, uint64(0), receipt.LeafIndex)
	assert.Equal(t, event, receipt.EventBytes)
	require.NotNil(t, receipt.Iss)
	assert.Equal(t, testOrigin, *receipt.Iss)
}

// TestSignWithChainCarriesTheCertificate checks the JWS helper against the
// production reader and verifier for every key type claims support.
func TestSignWithChainCarriesTheCertificate(t *testing.T) {
	payload := identityv1.CanonicalBytes(testRecordCID, testAnsName, testSignedAt)

	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	_, edKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	p384Key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	tests := []struct {
		name string
		cert identityCert
	}{
		{name: "p-256", cert: mintIdentityCert(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour))},
		{name: "p-384", cert: mintIdentityCertWithKey(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), p384Key)},
		{name: "rsa", cert: mintIdentityCertWithKey(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), rsaKey)},
		{name: "ed25519", cert: mintIdentityCertWithKey(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), edKey)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwsCompact := signWithChain(t, tt.cert.key, tt.cert.der, payload)

			carried, err := identityv1.CertificateFromJWS(jwsCompact)
			require.NoError(t, err)
			assert.Equal(t, tt.cert.der, carried.Raw)

			require.NoError(t, identityv1.VerifyJWS(jwsCompact, carried.PublicKey, payload))
			require.Error(t, identityv1.VerifyJWS(jwsCompact, carried.PublicKey, []byte("other payload")))

			_, err = identityv1.CertificateFromJWS(signWithChain(t, tt.cert.key, nil, payload))
			require.ErrorContains(t, err, "no x5c")
		})
	}
}

// TestMintOversizedIdentityCertExceedsTheLimit checks that the padded
// certificate is one the production reader refuses.
func TestMintOversizedIdentityCertExceedsTheLimit(t *testing.T) {
	payload := identityv1.CanonicalBytes(testRecordCID, testAnsName, testSignedAt)
	oversized := mintOversizedIdentityCert(t, testAnsName)

	_, err := identityv1.CertificateFromJWS(signWithChain(t, oversized.key, oversized.der, payload))
	require.ErrorContains(t, err, "exceeds")
}
