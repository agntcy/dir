// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"crypto"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The files under testdata/ were captured from the Agent Name Service
// reference implementation (github.com/agentnameservice/ans, commit recorded
// in fixture.json) running its demo stack. The transparency log listens on
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
//	curl -fsS -H "Authorization: Bearer $RA_API_KEY" \
//	  "http://127.0.0.1:18080/v2/ans/agents/$ID/certificates/identity" |
//	  jq -r '.[0].certificatePEM' > testdata/identity-cert.pem
//	scripts/demo/stop.sh
//
// fixture.json holds the agent id, the ANS name, the badge URL and the capture
// time the test pins its clock to. The certificate's private key was never
// captured, so the resolver is driven with the certificate alone: Resolve
// returns its key, and the claim signature that key would verify is the
// reconciler's concern.

type goldenFixture struct {
	AgentID        string `json:"agentId"`
	AnsName        string `json:"ansName"`
	BadgeURL       string `json:"badgeUrl"`
	CapturedAtUnix int64  `json:"capturedAtUnix"`
}

// golden is the captured material wired into the resolver's dependencies:
// a fake DNS answer pointing at the captured badge URL, and a fake fetcher
// serving the captured token and root keys to the real log client.
type golden struct {
	fixture    goldenFixture
	rootKeys   []string
	certDER    []byte
	cert       *x509.Certificate
	logHost    string
	capturedAt time.Time
	dns        *fakeTXT
	fetcher    *fakeFetcher
}

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)

	return data
}

func loadGolden(t *testing.T) *golden {
	t.Helper()

	g := &golden{fetcher: newFakeFetcher()}

	require.NoError(t, json.Unmarshal(readTestdata(t, "fixture.json"), &g.fixture))

	g.rootKeys = strings.Fields(string(readTestdata(t, "root-keys.txt")))
	g.capturedAt = time.Unix(g.fixture.CapturedAtUnix, 0)

	block, _ := pem.Decode(readTestdata(t, "identity-cert.pem"))
	require.NotNil(t, block, "identity-cert.pem does not hold a PEM block")
	require.Equal(t, "CERTIFICATE", block.Type)

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	g.certDER, g.cert = block.Bytes, cert

	badge, err := url.Parse(g.fixture.BadgeURL)
	require.NoError(t, err)

	g.logHost = badge.Host

	host := strings.TrimPrefix(g.fixture.AnsName, ansScheme+"v1.0.0.")
	g.dns = &fakeTXT{records: map[string][]string{badgeRecordPrefix + host: {badgeTXT("v1.0.0", g.fixture.BadgeURL)}}}

	origin := badge.Scheme + "://" + badge.Host
	g.fetcher.serve(origin+"/root-keys", readTestdata(t, "root-keys.txt"))
	g.fetcher.serve(g.fixture.BadgeURL+"/status-token", readTestdata(t, "status-token.cbor"))

	return g
}

// build wires the captured material into a resolver whose log client is the
// real one, fetching through the fake, with the clock at now.
func (g *golden) build(t *testing.T, cfg Config, now time.Time) *Resolver {
	t.Helper()

	r, err := New(cfg, WithLookupTXT(g.dns.lookup), WithFetcher(g.fetcher), WithClock(newFakeClock(now).Now))
	require.NoError(t, err)

	return r
}

func TestGolden(t *testing.T) {
	g := loadGolden(t)

	tests := []struct {
		name          string
		cfg           Config
		wantRootKeyGs int
	}{
		{
			name: "pinned root keys",
			cfg:  Config{TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys},
		},
		{
			name:          "fetched root keys",
			cfg:           Config{TrustedLogHosts: []string{g.logHost}, AllowUnpinnedRootKeys: true},
			wantRootKeyGs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := loadGolden(t)
			r := g.build(t, tt.cfg, g.capturedAt)

			keys, err := r.Resolve(t.Context(), g.fixture.AnsName, g.certDER)
			require.NoError(t, err)
			require.Len(t, keys, 1)

			equaler, ok := keys[0].(interface{ Equal(crypto.PublicKey) bool })
			require.True(t, ok)
			assert.True(t, equaler.Equal(g.cert.PublicKey), "the key is not the captured certificate's")

			assert.Equal(t, int64(1), g.dns.calls.Load())
			assert.Equal(t, 1, g.fetcher.count(g.fixture.BadgeURL+"/status-token"))
			assert.Equal(t, tt.wantRootKeyGs, g.fetcher.count(strings.TrimSuffix(g.fixture.BadgeURL, "/v1/agents/"+g.fixture.AgentID)+"/root-keys"))
		})
	}
}

// The captured log attests the captured certificate only: another
// certificate naming the same agent is refused at the fingerprint check.
func TestGoldenRejectsAnotherCertificate(t *testing.T) {
	g := loadGolden(t)
	r := g.build(t, Config{TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys}, g.capturedAt)

	other := mintIdentityCert(t, g.fixture.AnsName, g.capturedAt.Add(-time.Hour), g.capturedAt.Add(time.Hour))

	_, err := r.Resolve(t.Context(), g.fixture.AnsName, other.der)
	require.EqualError(t, err, "ans log: the certificate is not among the agent's valid identity certificates")
}

// The captured token expires; a clock past its expiry refuses it.
func TestGoldenTokenExpiry(t *testing.T) {
	g := loadGolden(t)
	r := g.build(t, Config{TrustedLogHosts: []string{g.logHost}, RootKeys: g.rootKeys}, g.cert.NotAfter.Add(-time.Minute))

	_, err := r.Resolve(t.Context(), g.fixture.AnsName, g.certDER)
	require.ErrorContains(t, err, "ans log: status token did not verify: token expired")
}
