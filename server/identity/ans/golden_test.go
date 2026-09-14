// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The files under testdata/ were captured from the ANS reference
// implementation (github.com/agentnameservice/ans, commit recorded in
// fixture.json) running its demo stack. The transparency log listens on
// plain HTTP at 127.0.0.1:18081 while the badge URLs it publishes use the
// configured public base https://localhost:18081, which is what fixture.json
// records. Capture commands, run from the ans checkout:
//
//	scripts/demo/start.sh --with-dns
//	scripts/demo/run-lifecycle.sh dir-fixture.example.com 1.0.0
//	ID=$(cat data/demo/last-agent-id)
//	TL=http://127.0.0.1:18081
//	curl -fsS "$TL/root-keys" > testdata/root-keys.txt
//	curl -fsS "$TL/v1/agents/$ID/status-token" > testdata/status-token.cbor
//	until curl -fsS "$TL/v1/agents/$ID/receipt" > testdata/receipt.cbor; do sleep 2; done
//	curl -fsS -H "Authorization: Bearer $RA_API_KEY" \
//	  "http://127.0.0.1:18080/v2/ans/agents/$ID/certificates/identity" |
//	  jq -r '.[0].certificatePEM' > testdata/identity-cert.pem
//	scripts/demo/stop.sh
//
// fixture.json holds the agent id, the ANS name, the badge URL, the capture
// time the test pins its clock to, and the ans commit. The certificate's
// private key was never captured, so the attestation is driven directly and
// the one Verify case signs with a fresh key.

type goldenFixture struct {
	AgentID        string `json:"agentId"`
	AnsName        string `json:"ansName"`
	BadgeURL       string `json:"badgeUrl"`
	CapturedAtUnix int64  `json:"capturedAtUnix"`
	AnsCommit      string `json:"ansCommit"`
}

// golden is the captured material wired into the resolver's dependencies.
type golden struct {
	fixture    goldenFixture
	rootKeys   []string
	cert       identityCert
	name       agentName
	logHost    string
	capturedAt time.Time
	dns        *countingBadges
	log        *fakeLogClient
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	return data
}

func loadGolden(t *testing.T) golden {
	t.Helper()

	var g golden

	require.NoError(t, json.Unmarshal(readTestdata(t, "fixture.json"), &g.fixture))

	g.rootKeys = strings.Fields(string(readTestdata(t, "root-keys.txt")))
	g.capturedAt = time.Unix(g.fixture.CapturedAtUnix, 0)

	block, _ := pem.Decode(readTestdata(t, "identity-cert.pem"))
	require.NotNil(t, block, "identity-cert.pem does not hold a PEM block")
	require.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	g.cert = identityCert{der: block.Bytes, cert: cert, fingerprint: verify.CertFingerprintFromDER(block.Bytes).String()}

	badge, err := url.Parse(g.fixture.BadgeURL)
	require.NoError(t, err)

	g.logHost = badge.Host

	g.name, err = parseAgentName(g.fixture.AnsName)
	require.NoError(t, err)

	g.dns = &countingBadges{badgeFinder: badgeRecords(t, g.name.host, g.name.version.String(), g.fixture.BadgeURL)}
	g.log = &fakeLogClient{
		rootKeys: g.rootKeys,
		token:    readTestdata(t, "status-token.cbor"),
		receipt:  readTestdata(t, "receipt.cbor"),
	}

	return g
}

// build wires the captured material into a resolver.
func (g golden) build(t *testing.T, cfg ansconfig.Config) *Resolver {
	t.Helper()

	r, err := New(cfg)
	require.NoError(t, err)

	r.badges = g.dns
	r.clock = func() time.Time { return g.capturedAt }
	r.newLogClient = func(string) (scitt.Client, error) { return g.log, nil }

	return r
}

func TestGoldenAttestation(t *testing.T) {
	g := loadGolden(t)

	tests := []struct {
		name      string
		cfg       ansconfig.Config
		wantCalls int
	}{
		{
			name:      "pinned root keys",
			cfg:       ansconfig.Config{Enabled: true, TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys},
			wantCalls: 2,
		},
		{
			name:      "fetched root keys",
			cfg:       ansconfig.Config{Enabled: true, TrustedLogHosts: []string{g.logHost}, AllowUnpinnedRootKeys: true},
			wantCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := loadGolden(t)
			r := g.build(t, tt.cfg)

			require.NoError(t, checkCertificate(g.cert.cert, g.name, g.capturedAt))
			require.NoError(t, r.verifyAttestation(t.Context(), g.name, g.cert.cert))

			assert.Equal(t, tt.wantCalls, g.log.callCount())
			assert.Equal(t, []string{g.fixture.AgentID, g.fixture.AgentID}, g.log.agentIDs)
			assert.Equal(t, 1, g.dns.callCount())
		})
	}
}

// TestGoldenAttestationRejectsAnotherCertificate checks that the captured
// log attests the captured certificate only.
func TestGoldenAttestationRejectsAnotherCertificate(t *testing.T) {
	g := loadGolden(t)
	r := g.build(t, ansconfig.Config{Enabled: true, TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys})

	other := mintIdentityCert(t, g.fixture.AnsName, g.capturedAt.Add(-time.Hour), g.capturedAt.Add(time.Hour))

	err := r.verifyAttestation(t.Context(), g.name, other.cert)
	require.EqualError(t, err, "ans certificate: certificate is not attested for this agent")
}

// TestGoldenCertificateWithAnotherKey signs a claim with a fresh key while
// carrying the captured certificate: the signature does not verify against
// the certificate, so the verdict is (false, nil) and nothing is fetched.
func TestGoldenCertificateWithAnotherKey(t *testing.T) {
	g := loadGolden(t)
	r := g.build(t, ansconfig.Config{Enabled: true, TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys})

	payload := identityv1.CanonicalBytes(testRecordCID, g.fixture.AnsName, g.capturedAt.UTC().Format(time.RFC3339))
	fresh := mintIdentityCert(t, g.fixture.AnsName, g.capturedAt.Add(-time.Hour), g.capturedAt.Add(time.Hour))

	ok, err := r.Verify(t.Context(), g.fixture.AnsName, []byte(signWithChain(t, fresh.key, g.cert.der, payload)), payload)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, 0, g.dns.callCount())
	assert.Equal(t, 0, g.log.callCount())
}
