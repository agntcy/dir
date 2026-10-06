// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/fxamacker/cbor/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLog is a synthetic transparency log: one ES256 signing key, its 4-byte
// key id, and the name its root-key line carries. The tokens it mints follow
// the layout the SDK's own tests verify against.
type testLog struct {
	key  *ecdsa.PrivateKey
	kid  [4]byte
	name string
}

func mintLog(t *testing.T) *testLog {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	sum := sha256.Sum256(spkiOf(t, &key.PublicKey))

	var kid [4]byte

	copy(kid[:], sum[:4])

	return &testLog{key: key, kid: kid, name: "example-log"}
}

// rootKeyLine renders the log's verification key as one /root-keys line:
// name+hex(kid)+base64(0x02 || SPKI DER).
func (l *testLog) rootKeyLine(t *testing.T) string {
	t.Helper()

	spki := spkiOf(t, &l.key.PublicKey)

	return l.name + "+" + hex.EncodeToString(l.kid[:]) + "+" + base64.StdEncoding.EncodeToString(append([]byte{0x02}, spki...))
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

// claims returns an ACTIVE token payload for the test agent attesting certs.
func (l *testLog) claims(certs ...identityCert) tokenClaims {
	fingerprints := make([]string, 0, len(certs))
	for _, cert := range certs {
		fingerprints = append(fingerprints, cert.fingerprint)
	}

	return tokenClaims{
		agentID:       testAgentID,
		ansName:       testAnsName,
		status:        string(StateActive),
		iat:           testNow.Unix() - 60,
		exp:           testNow.Unix() + 3600,
		identityCerts: fingerprints,
	}
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

	return l.coseSign1(t, protected, mustCBOR(t, payload))
}

// coseSign1 signs payload under the protected header with ES256 and returns
// the tag-18 COSE_Sign1 encoding. r and s are written with FillBytes so the
// P1363 signature is always 64 bytes.
func (l *testLog) coseSign1(t *testing.T, protected map[int64]any, payload []byte) []byte {
	t.Helper()

	protectedBytes := mustCBOR(t, protected)

	digest, err := scitt.ComputeSigStructureDigest(protectedBytes, payload)
	require.NoError(t, err)

	r, s, err := ecdsa.Sign(rand.Reader, l.key, digest[:])
	require.NoError(t, err)

	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return mustCBOR(t, cbor.Tag{Number: 18, Content: []any{protectedBytes, map[int64]any{}, payload, sig}})
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

// fingerprintOf is the SHA-256 fingerprint of a DER certificate.
func fingerprintOf(der []byte) [32]byte {
	return sha256.Sum256(der)
}

// mintIdentityCert mints a self-signed P-256 identity certificate whose only
// SAN is the ANS name URI.
func mintIdentityCert(t *testing.T, ansName string, notBefore, notAfter time.Time) identityCert {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return mintIdentityCertWithKey(t, ansName, notBefore, notAfter, key)
}

// mintTestCert mints an identity certificate for the test agent valid around testNow.
func mintTestCert(t *testing.T) identityCert {
	t.Helper()

	return mintIdentityCert(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(24*time.Hour))
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

	fingerprint := fingerprintOf(der)

	return identityCert{der: der, cert: parsed, key: key, fingerprint: "SHA256:" + hex.EncodeToString(fingerprint[:])}
}

// TestMintedFixturesVerify checks the minting helpers against the SDK
// verifier, so a helper regression cannot masquerade as a resolver bug.
func TestMintedFixturesVerify(t *testing.T) {
	log := mintLog(t)
	cert := mintTestCert(t)

	keys, err := scitt.NewKeyStore([]string{log.rootKeyLine(t)})
	require.NoError(t, err)

	verified, err := scitt.VerifyStatusTokenAt(log.statusToken(t, log.claims(cert)), keys, 0, testNow.Unix())
	require.NoError(t, err)

	assert.Equal(t, testAgentID, verified.Payload.AgentID)
	assert.Equal(t, testAnsName, verified.Payload.AnsName)
	assert.True(t, scitt.MatchesIdentityCert(&verified.Payload, fingerprintOf(cert.der)), "minted token does not attest the minted certificate")
}
