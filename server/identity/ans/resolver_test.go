// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/models"
	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The resolver must satisfy identity.Resolver without importing it, since
// server/identity will import this package.
var _ interface {
	Scheme() string
	Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error)
} = (*Resolver)(nil)

// countingBadges counts badge lookups on top of a scripted finder and can
// hold each answer back for a while.
type countingBadges struct {
	badgeFinder

	calls atomic.Int64
	delay time.Duration
}

func (b *countingBadges) FindBadgeForVersion(ctx context.Context, fqdn models.Fqdn, version models.Version) (*verify.AnsBadgeRecord, error) {
	b.calls.Add(1)

	if b.delay > 0 {
		select {
		case <-time.After(b.delay):
		case <-ctx.Done():
			return nil, ctx.Err() //nolint:wrapcheck // test double
		}
	}

	return b.badgeFinder.FindBadgeForVersion(ctx, fqdn, version) //nolint:wrapcheck // pass-through test double
}

func (b *countingBadges) callCount() int {
	return int(b.calls.Load())
}

// badgeRecords scripts one badge record for host at version.
func badgeRecords(t *testing.T, host, version, url string) *verify.MockDNSResolver {
	t.Helper()

	parsed, err := models.ParseVersion(version)
	require.NoError(t, err)

	return verify.NewMockDNSResolver().WithRecords(host, []verify.AnsBadgeRecord{{FormatVersion: "ans-badge1", Version: &parsed, URL: url}})
}

// fixture is a complete, passing verification scenario that individual cases
// mutate before calling verify.
type fixture struct {
	t          *testing.T
	log        *testLog
	cert       identityCert
	dns        *countingBadges
	client     *fakeLogClient
	clock      time.Time
	cfg        ansconfig.Config
	subject    string
	payload    []byte
	signature  string
	factoryErr error
	realClient bool
	resolver   *Resolver

	mu       sync.Mutex
	logBases []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	log := mintLog(t)
	cert := mintIdentityCert(t, testAnsName, testNow.Add(-time.Hour), testNow.Add(24*time.Hour))
	payload := identityv1.CanonicalBytes(testRecordCID, testAnsName, testSignedAt)

	f := &fixture{
		t:         t,
		log:       log,
		cert:      cert,
		clock:     testNow,
		subject:   testAnsName,
		payload:   payload,
		signature: signWithChain(t, cert.key, cert.der, payload),
		cfg: ansconfig.Config{
			Enabled:         true,
			TrustedLogHosts: []string{testLogHost},
			RootKeys:        []string{log.rootKeyLine(t)},
		},
		client: &fakeLogClient{rootKeys: []string{log.rootKeyLine(t)}},
	}

	f.setBadgeURL(testBadgeURL)
	f.setToken(f.claims())
	f.setReceipt(eventJSON(t, "ansId", testAgentID, testAnsName))

	return f
}

func (f *fixture) claims() tokenClaims {
	return tokenClaims{
		agentID:       testAgentID,
		ansName:       testAnsName,
		status:        "ACTIVE",
		iat:           testNow.Unix() - 60,
		exp:           testNow.Unix() + 3600,
		identityCerts: []string{f.cert.fingerprint},
	}
}

func (f *fixture) setToken(claims tokenClaims) {
	f.client.token = f.log.statusToken(f.t, claims)
}

func (f *fixture) setReceipt(event []byte) {
	f.client.receipt = f.log.receipt(f.t, event, 1, 0, nil, testNow.Unix())
}

func (f *fixture) setBadgeURL(raw string) {
	f.dns = &countingBadges{badgeFinder: badgeRecords(f.t, testHost, testVersion, raw)}
}

func (f *fixture) setDNSError(err error) {
	f.dns = &countingBadges{badgeFinder: verify.NewMockDNSResolver().WithError(testHost, err)}
}

// mintCert mints another identity certificate valid at the fixture clock.
func (f *fixture) mintCert(ansName string) identityCert {
	return mintIdentityCert(f.t, ansName, testNow.Add(-time.Hour), testNow.Add(time.Hour))
}

// sign makes cert the certificate the claim carries and signs the payload
// with its key.
func (f *fixture) sign(cert identityCert) {
	f.signature = signWithChain(f.t, cert.key, cert.der, f.payload)
}

// useCert makes cert the fixture's attested certificate: the token attests
// it and the claim is signed with it.
func (f *fixture) useCert(cert identityCert) {
	f.cert = cert
	f.sign(cert)
	f.setToken(f.claims())
}

func (f *fixture) recordBase(base string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logBases = append(f.logBases, base)
}

func (f *fixture) build() *Resolver {
	if f.resolver != nil {
		return f.resolver
	}

	r, err := New(f.cfg)
	require.NoError(f.t, err)

	r.badges = f.dns
	r.clock = func() time.Time { return f.clock }

	if !f.realClient {
		r.newLogClient = func(base string) (scitt.Client, error) {
			f.recordBase(base)

			if f.factoryErr != nil {
				return nil, f.factoryErr
			}

			return f.client, nil
		}
	}

	f.resolver = r

	return r
}

func (f *fixture) verify() (bool, error) {
	return f.build().Verify(f.t.Context(), f.subject, []byte(f.signature), f.payload)
}

// verifyCase is one Verify scenario. An empty wantErr with wantOK false is
// the (false, nil) verdict of a signature that does not verify.
type verifyCase struct {
	name      string
	setup     func(f *fixture)
	wantOK    bool
	wantErr   string
	wantNoDNS bool
	wantNoLog bool
}

func runVerifyCases(t *testing.T, cases []verifyCase) {
	t.Helper()

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			if tt.setup != nil {
				tt.setup(f)
			}

			ok, err := f.verify()

			assertNetworkUse(t, f, tt)

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.False(t, ok)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func assertNetworkUse(t *testing.T, f *fixture, tt verifyCase) {
	t.Helper()

	if tt.wantNoDNS {
		assert.Equal(t, 0, f.dns.callCount(), "DNS was queried")
	}

	if tt.wantNoDNS || tt.wantNoLog {
		assert.Equal(t, 0, f.client.callCount(), "the transparency log was called")
	}
}

func TestVerifyNameStage(t *testing.T) {
	subject := func(s string) func(f *fixture) {
		return func(f *fixture) { f.subject = s }
	}

	runVerifyCases(t, []verifyCase{
		{name: "uppercase host is not canonical", setup: subject("ans://v1.0.0.Agent.Example.COM"), wantErr: "ans name: subject is not in canonical form", wantNoDNS: true},
		{name: "leading zero in the version is not canonical", setup: subject("ans://v01.0.0.agent.example.com"), wantErr: "ans name: subject is not in canonical form", wantNoDNS: true},
		{name: "trailing dot is not canonical", setup: subject(testAnsName + "."), wantErr: "ans name: subject is not in canonical form", wantNoDNS: true},
		{name: "uppercase scheme is not an ans name", setup: subject("ANS://v1.0.0.agent.example.com"), wantErr: "ans name: subject is not an ANS name", wantNoDNS: true},
		{name: "version without v", setup: subject("ans://1.0.0.agent.example.com"), wantErr: "ans name: subject is not an ANS name", wantNoDNS: true},
		{name: "dns subject", setup: subject("dns:agent.example.com"), wantErr: "ans name: subject is not an ANS name", wantNoDNS: true},
		{name: "host is not a dns name", setup: subject("ans://v1.0.0.bad_host.example.com"), wantErr: `ans name: host "bad_host.example.com" is not a valid DNS name`, wantNoDNS: true},
	})
}

func TestVerifyCertificateStage(t *testing.T) {
	runVerifyCases(t, []verifyCase{
		{
			name: "jws without x5c",
			setup: func(f *fixture) {
				f.signature = signWithChain(f.t, f.cert.key, nil, f.payload)
			},
			wantErr:   "ans certificate: jws carries no x5c certificate",
			wantNoDNS: true,
		},
		{
			name: "json serialization",
			setup: func(f *fixture) {
				sig, err := jws.Sign(f.payload, jws.WithKey(jwsAlgorithm(f.t, f.cert.key), f.cert.key, jws.WithProtectedHeaders(x5cHeaders(f.t, f.cert.der))), jws.WithJSON())
				require.NoError(f.t, err)

				f.signature = string(sig)
			},
			wantErr:   "ans certificate: jws has 1 segments; a compact serialization has 3",
			wantNoDNS: true,
		},
		{
			name: "two-certificate chain",
			setup: func(f *fixture) {
				f.signature = mintJWS(f.t, f.cert.key, x5cHeaders(f.t, f.cert.der, f.mintCert(testAnsName).der), f.payload)
			},
			wantErr:   "ans certificate: jws x5c carries 2 certificates; the leaf only is accepted",
			wantNoDNS: true,
		},
		{
			name: "not a jws",
			setup: func(f *fixture) {
				f.signature = "not a jws"
			},
			wantErr:   "ans certificate: jws has 1 segments",
			wantNoDNS: true,
		},
		{
			name: "oversized certificate",
			setup: func(f *fixture) {
				f.sign(mintOversizedIdentityCert(f.t, testAnsName))
			},
			wantErr:   "ans certificate: x5c certificate exceeds",
			wantNoDNS: true,
		},
		{
			name:      "certificate for another host",
			setup:     func(f *fixture) { f.sign(f.mintCert("ans://v1.0.0.other.example.com")) },
			wantErr:   "ans certificate: certificate does not name this agent",
			wantNoDNS: true,
		},
		{
			name:      "certificate for another version",
			setup:     func(f *fixture) { f.sign(f.mintCert("ans://v1.0.1.agent.example.com")) },
			wantErr:   "ans certificate: certificate does not name this agent",
			wantNoDNS: true,
		},
		{
			name:      "certificate without an ans uri san",
			setup:     func(f *fixture) { f.sign(f.mintCert("spiffe://trust.example.com/agent")) },
			wantErr:   "ans certificate: certificate does not name this agent",
			wantNoDNS: true,
		},
		{
			name: "expired certificate",
			setup: func(f *fixture) {
				f.sign(mintIdentityCert(f.t, testAnsName, testNow.Add(-2*time.Hour), testNow.Add(-time.Minute)))
			},
			wantErr:   "ans certificate: certificate expired or not yet valid",
			wantNoDNS: true,
		},
		{
			name: "certificate not yet valid",
			setup: func(f *fixture) {
				f.sign(mintIdentityCert(f.t, testAnsName, testNow.Add(time.Minute), testNow.Add(time.Hour)))
			},
			wantErr:   "ans certificate: certificate expired or not yet valid",
			wantNoDNS: true,
		},
		{
			name: "unsupported key type",
			setup: func(f *fixture) {
				key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
				require.NoError(f.t, err)

				f.sign(mintIdentityCertWithKey(f.t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), key))
			},
			wantErr:   "ans certificate: unsupported key type",
			wantNoDNS: true,
		},
		{
			name: "another key's certificate",
			setup: func(f *fixture) {
				f.signature = signWithChain(f.t, f.cert.key, f.mintCert(testAnsName).der, f.payload)
			},
			wantNoDNS: true,
		},
		{
			name: "attested key over a different record",
			setup: func(f *fixture) {
				f.signature = signWithChain(f.t, f.cert.key, f.cert.der, identityv1.CanonicalBytes("baguqeeraothercid", testAnsName, testSignedAt))
			},
			wantNoDNS: true,
		},
		{
			name: "tampered signature",
			setup: func(f *fixture) {
				// The first character of the signature segment carries six
				// signature bits, unlike the last one, whose low bits are padding.
				start := strings.LastIndexByte(f.signature, '.') + 1

				replacement := "A"
				if f.signature[start] == 'A' {
					replacement = "B"
				}

				f.signature = f.signature[:start] + replacement + f.signature[start+1:]
			},
			wantNoDNS: true,
		},
		{
			name: "production signer output verifies",
			setup: func(f *fixture) {
				f.signature = signWithProductionSigner(f.t, f.cert, f.payload)
			},
			wantOK: true,
		},
		{
			name: "production signer output verifies for an ed25519 key",
			setup: func(f *fixture) {
				_, key, err := ed25519.GenerateKey(rand.Reader)
				require.NoError(f.t, err)

				f.useCert(mintIdentityCertWithKey(f.t, testAnsName, testNow.Add(-time.Hour), testNow.Add(time.Hour), key))
				f.signature = signWithProductionSigner(f.t, f.cert, f.payload)
			},
			wantOK: true,
		},
	})
}

// signWithProductionSigner signs payload through identityv1.NewAnsSigner, the
// signer dirctl uses, so the resolver is proven against real claims.
func signWithProductionSigner(t *testing.T, cert identityCert, payload []byte) string {
	t.Helper()

	signer, err := identityv1.NewAnsSigner(keyPEM(t, cert.key), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.der}))
	require.NoError(t, err)

	jwsCompact, err := signer.Sign(payload)
	require.NoError(t, err)

	return jwsCompact
}

// keyPEM encodes key as PKCS8, the form the production signer reads for
// every key type.
func keyPEM(t *testing.T, key crypto.Signer) []byte {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestVerifyDNSStage(t *testing.T) {
	runVerifyCases(t, []verifyCase{
		{
			name: "no badge record",
			setup: func(f *fixture) {
				f.dns = &countingBadges{badgeFinder: verify.NewMockDNSResolver()}
			},
			wantErr:   "ans dns: no _ans-badge record for agent.example.com version v1.0.0",
			wantNoLog: true,
		},
		{
			name: "legacy _ra-badge record is accepted",
			setup: func(f *fixture) {
				version, err := models.ParseVersion(testVersion)
				require.NoError(f.t, err)

				records := []verify.AnsBadgeRecord{{FormatVersion: "ra-badge1", Version: &version, URL: testBadgeURL}}
				f.dns = &countingBadges{badgeFinder: verify.NewMockDNSResolver().WithRaBadgeRecords(testHost, records)}
			},
			wantOK: true,
		},
		{
			name: "badge record for another version only",
			setup: func(f *fixture) {
				f.dns = &countingBadges{badgeFinder: badgeRecords(f.t, testHost, "v2.0.0", testBadgeURL)}
			},
			wantErr:   "ans dns: no _ans-badge record for agent.example.com version v1.0.0",
			wantNoLog: true,
		},
		{
			name: "dns timeout",
			setup: func(f *fixture) {
				f.setDNSError(&verify.DNSError{Type: verify.DNSErrorTimeout, Fqdn: "_ans-badge." + testHost})
			},
			wantErr:   "ans dns: DNS lookup timed out",
			wantNoLog: true,
		},
		{
			name: "dns lookup failed",
			setup: func(f *fixture) {
				f.setDNSError(&verify.DNSError{Type: verify.DNSErrorLookupFailed, Fqdn: "_ans-badge." + testHost, Reason: "server misbehaving"})
			},
			wantErr:   "ans dns: DNS lookup failed",
			wantNoLog: true,
		},
		{
			name: "unexpected resolver error",
			setup: func(f *fixture) {
				f.setDNSError(errBoom)
			},
			wantErr:   "ans dns: unexpected failure",
			wantNoLog: true,
		},
	})
}

func TestVerifyBadgeStage(t *testing.T) {
	longHost := strings.Repeat("a", maxEchoLength+20) + ".example.com"

	runVerifyCases(t, []verifyCase{
		{
			name: "log host not allow-listed",
			setup: func(f *fixture) {
				f.setBadgeURL("https://evil.example.com/v1/agents/" + testAgentID)
			},
			wantErr:   `ans badge-url: log host "evil.example.com" is not a trusted transparency log`,
			wantNoLog: true,
		},
		{
			name: "log host on an unexpected port",
			setup: func(f *fixture) {
				f.setBadgeURL("https://log.example.com:8443/v1/agents/" + testAgentID)
			},
			wantErr:   `ans badge-url: log host "log.example.com:8443" is not a trusted transparency log`,
			wantNoLog: true,
		},
		{
			name: "untrusted log host is truncated in the error",
			setup: func(f *fixture) {
				f.setBadgeURL("https://" + longHost + "/v1/agents/" + testAgentID)
			},
			wantErr:   `ans badge-url: log host "` + longHost[:maxEchoLength] + `" is not a trusted transparency log`,
			wantNoLog: true,
		},
		{
			name: "trusted host configured with the default port in mixed case",
			setup: func(f *fixture) {
				f.cfg.TrustedLogHosts = []string{" LOG.Example.COM:443 "}
			},
			wantOK: true,
		},
		{
			name: "badge url is not https",
			setup: func(f *fixture) {
				f.setBadgeURL("http://log.example.com/v1/agents/" + testAgentID)
			},
			wantErr:   "ans badge-url: badge URL scheme is not https",
			wantNoLog: true,
		},
		{
			name: "badge path is not the agent resource",
			setup: func(f *fixture) {
				f.setBadgeURL(testBadgeURL + "/status-token")
			},
			wantErr:   "ans badge-url: badge URL path must be /v1/agents/{agentId}",
			wantNoLog: true,
		},
		{
			name: "badge url carries a query",
			setup: func(f *fixture) {
				f.setBadgeURL(testBadgeURL + "?x=1")
			},
			wantErr:   "ans badge-url: badge URL must not carry a query",
			wantNoLog: true,
		},
		{
			name: "badge agent id is not a uuid",
			setup: func(f *fixture) {
				f.setBadgeURL(testLogBase + "/v1/agents/not-a-uuid")
			},
			wantErr:   "ans badge-url: badge URL agent id is not a UUID",
			wantNoLog: true,
		},
		{
			name: "log client cannot be created",
			setup: func(f *fixture) {
				f.factoryErr = errBoom
			},
			wantErr:   "ans log: cannot create a client for the transparency log",
			wantNoLog: true,
		},
	})
}

func unpinned(f *fixture) {
	f.cfg.RootKeys = nil
	f.cfg.AllowUnpinnedRootKeys = true
}

func TestVerifyRootKeysStage(t *testing.T) {
	runVerifyCases(t, []verifyCase{
		{
			name:   "fetched root keys verify the log",
			setup:  unpinned,
			wantOK: true,
		},
		{
			name: "root keys fetch fails with 500",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.rootKeysErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusInternalServerError}
			},
			wantErr: "ans root-keys: transparency log returned HTTP 500",
		},
		{
			name: "root keys fetch fails on the connection",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.rootKeysErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}
			},
			wantErr: "ans root-keys: transparency log unreachable",
		},
		{
			name: "malformed root keys",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.rootKeys = []string{"not-a-root-key"}
			},
			wantErr: "ans root-keys: transparency log served malformed root keys",
		},
		{
			name: "empty root keys",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.rootKeys = nil
			},
			wantErr: "ans root-keys: transparency log served no root keys",
		},
		{
			name: "pinned root key line with surrounding whitespace",
			setup: func(f *fixture) {
				f.cfg.RootKeys = []string{" " + f.log.rootKeyLine(f.t) + " "}
			},
			wantOK: true,
		},
		{
			name: "pinned keys reject a log signing with another key",
			setup: func(f *fixture) {
				f.client.token = mintLog(f.t).statusToken(f.t, f.claims())
			},
			wantErr: "ans status-token: signed by unknown key id",
		},
	})
}

func TestVerifyStatusTokenStage(t *testing.T) {
	withClaims := func(mutate func(claims *tokenClaims)) func(f *fixture) {
		return func(f *fixture) {
			claims := f.claims()
			mutate(&claims)
			f.setToken(claims)
		}
	}

	runVerifyCases(t, []verifyCase{
		{
			name:    "token names another agent",
			setup:   withClaims(func(claims *tokenClaims) { claims.agentID = otherAgentID }),
			wantErr: "ans status-token: token names agent " + otherAgentID + ", expected " + testAgentID,
		},
		{
			name:    "token names another host",
			setup:   withClaims(func(claims *tokenClaims) { claims.ansName = "ans://v1.0.0.other.example.com" }),
			wantErr: "ans status-token: token names ans://v1.0.0.other.example.com, expected ans://v1.0.0.agent.example.com",
		},
		{
			name:    "token names another version",
			setup:   withClaims(func(claims *tokenClaims) { claims.ansName = "ans://v2.0.0.agent.example.com" }),
			wantErr: "ans status-token: token names ans://v2.0.0.agent.example.com, expected ans://v1.0.0.agent.example.com",
		},
		{
			name:   "uppercase token name matches the subject",
			setup:  withClaims(func(claims *tokenClaims) { claims.ansName = "ans://v1.0.0.AGENT.example.com" }),
			wantOK: true,
		},
		{
			name:   "uppercase token agent id matches the badge",
			setup:  withClaims(func(claims *tokenClaims) { claims.agentID = strings.ToUpper(testAgentID) }),
			wantOK: true,
		},
		{
			name: "certificate not attested",
			setup: withClaims(func(claims *tokenClaims) {
				claims.identityCerts = []string{"SHA256:" + strings.Repeat("00", 32)}
			}),
			wantErr: "ans certificate: certificate is not attested for this agent",
		},
		{
			name: "certificate attested among others",
			setup: func(f *fixture) {
				claims := f.claims()
				claims.identityCerts = []string{f.mintCert(testAnsName).fingerprint, f.cert.fingerprint, f.mintCert(testAnsName).fingerprint}
				f.setToken(claims)
			},
			wantOK: true,
		},
		{
			name:   "deprecated agent is accepted",
			setup:  withClaims(func(claims *tokenClaims) { claims.status = "DEPRECATED" }),
			wantOK: true,
		},
		{
			name:   "warning agent is accepted",
			setup:  withClaims(func(claims *tokenClaims) { claims.status = "WARNING" }),
			wantOK: true,
		},
		{
			name:    "revoked agent",
			setup:   withClaims(func(claims *tokenClaims) { claims.status = "REVOKED" }),
			wantErr: "ans status-token: agent status REVOKED is terminal",
		},
		{
			name:    "pending agent",
			setup:   withClaims(func(claims *tokenClaims) { claims.status = "PENDING_DNS" }),
			wantErr: "ans status-token: agent status PENDING_DNS does not allow connections",
		},
		{
			name:    "unknown status",
			setup:   withClaims(func(claims *tokenClaims) { claims.status = "SOMETHING_NEW" }),
			wantErr: "ans status-token: agent status SOMETHING_NEW does not allow connections",
		},
		{
			name: "status token 410",
			setup: func(f *fixture) {
				f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrAgentTerminal, StatusCode: http.StatusGone}
			},
			wantErr: "ans status-token: agent is in a terminal state (HTTP 410)",
		},
		{
			name: "status token 404",
			setup: func(f *fixture) {
				f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrNotFound, StatusCode: http.StatusNotFound}
			},
			wantErr: "ans status-token: not found on the transparency log (HTTP 404)",
		},
		{
			name: "status token 501",
			setup: func(f *fixture) {
				f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrNotSupported, StatusCode: http.StatusNotImplemented}
			},
			wantErr: "ans status-token: transparency log returned HTTP 501",
		},
		{
			name:    "status token expired",
			setup:   withClaims(func(claims *tokenClaims) { claims.exp = testNow.Unix() - 60 }),
			wantErr: "ans status-token: status token expired at",
		},
		{
			name: "status token tampered",
			setup: func(f *fixture) {
				f.client.token[len(f.client.token)-1] ^= 0x01
			},
			wantErr: "ans status-token: signature verification failed",
		},
		{
			name: "status token is not cose",
			setup: func(f *fixture) {
				f.client.token = []byte("nope")
			},
			wantErr: "ans status-token: malformed COSE_Sign1 structure",
		},
		{
			name: "total time budget exhausted",
			setup: func(f *fixture) {
				f.cfg.Timeout = 50 * time.Millisecond
				f.client.block = true
			},
			wantErr: "ans status-token: timed out",
		},
	})
}

func TestVerifyReceiptStage(t *testing.T) {
	runVerifyCases(t, []verifyCase{
		{
			name: "receipt 503",
			setup: func(f *fixture) {
				f.client.receiptErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}
			},
			wantErr: "ans receipt: transparency log returned HTTP 503",
		},
		{
			name: "receipt 404",
			setup: func(f *fixture) {
				f.client.receiptErr = &scitt.TransportError{Type: scitt.TransportErrNotFound, StatusCode: http.StatusNotFound}
			},
			wantErr: "ans receipt: not found on the transparency log (HTTP 404)",
		},
		{
			name: "receipt 410",
			setup: func(f *fixture) {
				f.client.receiptErr = &scitt.TransportError{Type: scitt.TransportErrAgentTerminal, StatusCode: http.StatusGone}
			},
			wantErr: "ans receipt: agent is in a terminal state (HTTP 410)",
		},
		{
			name: "receipt names another agent",
			setup: func(f *fixture) {
				f.setReceipt(eventJSON(f.t, "ansId", otherAgentID, testAnsName))
			},
			wantErr: "ans receipt: receipt event names agent " + otherAgentID + ", expected " + testAgentID,
		},
		{
			name: "receipt names another host",
			setup: func(f *fixture) {
				f.setReceipt(eventJSON(f.t, "ansId", testAgentID, "ans://v1.0.0.other.example.com"))
			},
			wantErr: "ans receipt: receipt event names ans://v1.0.0.other.example.com, expected ans://v1.0.0.agent.example.com",
		},
		{
			name: "receipt event keyed by agentId",
			setup: func(f *fixture) {
				f.setReceipt(eventJSON(f.t, "agentId", testAgentID, testAnsName))
			},
			wantOK: true,
		},
		{
			name: "uppercase receipt agent id matches the badge",
			setup: func(f *fixture) {
				f.setReceipt(eventJSON(f.t, "ansId", strings.ToUpper(testAgentID), testAnsName))
			},
			wantOK: true,
		},
		{
			name: "receipt payload is not an envelope",
			setup: func(f *fixture) {
				f.setReceipt([]byte("not json"))
			},
			wantErr: "ans receipt: event payload is not an ANS event envelope",
		},
		{
			name: "receipt envelope without an event",
			setup: func(f *fixture) {
				f.setReceipt([]byte(`{"payload":{"producer":{}}}`))
			},
			wantErr: "ans receipt: event payload is not an ANS event envelope",
		},
		{
			name: "receipt signed by an unknown key",
			setup: func(f *fixture) {
				f.client.receipt = mintLog(f.t).receipt(f.t, eventJSON(f.t, "ansId", testAgentID, testAnsName), 1, 0, nil, testNow.Unix())
			},
			wantErr: "ans receipt: signed by unknown key id",
		},
		{
			name: "receipt tampered",
			setup: func(f *fixture) {
				f.client.receipt[len(f.client.receipt)-1] ^= 0x01
			},
			wantErr: "ans receipt: signature verification failed",
		},
		{
			name: "receipt issuer differs from the root key origin",
			setup: func(f *fixture) {
				f.log.origin = "someone-else"
				f.setReceipt(eventJSON(f.t, "ansId", testAgentID, testAnsName))
			},
			wantErr: "ans receipt: issuer does not match the signing key",
		},
		{
			name: "receipt with an inclusion path",
			setup: func(f *fixture) {
				sibling := scitt.ComputeLeafHash([]byte("another event"))
				f.client.receipt = f.log.receipt(f.t, eventJSON(f.t, "ansId", testAgentID, testAnsName), 2, 1, [][]byte{sibling[:]}, testNow.Unix())
			},
			wantOK: true,
		},
	})
}

// TestVerifyReachesTheBadgeLog checks that the log named by the badge is the
// one asked, for the badge's agent, once per artifact.
func TestVerifyReachesTheBadgeLog(t *testing.T) {
	f := newFixture(t)

	ok, err := f.verify()
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, []string{testLogBase}, f.logBases)
	assert.Equal(t, []string{testAgentID, testAgentID}, f.client.agentIDs)
	assert.Equal(t, 2, f.client.callCount())
	assert.Equal(t, 1, f.dns.callCount())
	assert.Equal(t, "ans", f.build().Scheme())
}

// TestVerifyReturnsTheCallerContextError checks that a verification cut
// short by the caller reports the caller's context error, not a stage.
func TestVerifyReturnsTheCallerContextError(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *fixture) context.Context
		wantErr error
	}{
		{
			name: "canceled before the lookup",
			setup: func(f *fixture) context.Context {
				f.client.block = true

				return canceledContext(f.t)
			},
			wantErr: context.Canceled,
		},
		{
			name: "canceled while the log is fetched",
			setup: func(f *fixture) context.Context {
				ctx, cancel := context.WithCancel(f.t.Context())
				f.client.block = true
				f.client.onCall = cancel

				return ctx
			},
			wantErr: context.Canceled,
		},
		{
			name: "caller deadline shorter than the budget",
			setup: func(f *fixture) context.Context {
				f.client.block = true

				ctx, cancel := context.WithTimeout(f.t.Context(), 20*time.Millisecond)
				f.t.Cleanup(cancel)

				return ctx
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			ctx := tt.setup(f)

			ok, err := f.build().Verify(ctx, f.subject, []byte(f.signature), f.payload)
			require.ErrorIs(t, err, tt.wantErr)
			assert.False(t, ok)

			var stage *stageError

			require.NotErrorAs(t, err, &stage, "the caller's context error must be returned as is")
		})
	}
}

func TestVerifyCircuitBreaker(t *testing.T) {
	logs := captureLogs(t)
	f := newFixture(t)
	f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}

	for i := range breakerThreshold {
		ok, err := f.verify()
		require.ErrorContains(t, err, "ans status-token: transparency log unreachable", "verification %d", i+1)
		assert.False(t, ok)
	}

	assert.Contains(t, logs.String(), "Transparency log circuit opened")

	ok, err := f.verify()
	require.EqualError(t, err, "ans log: transparency log log.example.com is unavailable until 2026-09-11T12:02:00Z; circuit open")
	assert.False(t, ok)
	assert.Equal(t, breakerThreshold, f.client.callCount(), "an open circuit must fail fast")
	assert.Equal(t, breakerThreshold+1, f.dns.callCount(), "DNS is consulted before the breaker")

	f.clock = testNow.Add(breakerCooldown - time.Second)

	_, err = f.verify()
	require.ErrorContains(t, err, "circuit open")

	f.clock = testNow.Add(breakerCooldown)
	f.client.tokenErr = nil

	ok, err = f.verify()
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, breakerThreshold+2, f.client.callCount())
	assert.Equal(t, 1, strings.Count(logs.String(), "Transparency log circuit closed"))
}

func TestVerifyCircuitBreakerIgnoresHTTPStatus(t *testing.T) {
	f := newFixture(t)
	f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}

	for range breakerThreshold + 1 {
		_, err := f.verify()
		require.ErrorContains(t, err, "ans status-token: transparency log returned HTTP 503")
	}

	assert.Equal(t, breakerThreshold+1, f.client.callCount())
}

func TestVerifyCircuitBreakerResetsOnSuccess(t *testing.T) {
	f := newFixture(t)
	connErr := &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}

	f.client.tokenErr = connErr

	for range breakerThreshold - 1 {
		_, err := f.verify()
		require.ErrorContains(t, err, "transparency log unreachable")
	}

	f.client.tokenErr = nil

	ok, err := f.verify()
	require.NoError(t, err)
	assert.True(t, ok)

	f.client.tokenErr = connErr

	for range breakerThreshold - 1 {
		_, err := f.verify()
		require.ErrorContains(t, err, "transparency log unreachable", "the success did not reset the strike count")
	}

	assert.Equal(t, 2*(breakerThreshold-1)+2, f.client.callCount())
}

// TestVerifyCircuitBreakerIsolatesHosts checks that strikes against one
// transparency log leave claims anchored at another log unaffected.
func TestVerifyCircuitBreakerIsolatesHosts(t *testing.T) {
	const (
		otherHost    = "ans://v1.0.0.other.example.com"
		otherLogHost = "other-log.example.com"
		otherLogBase = "https://" + otherLogHost
	)

	f := newFixture(t)
	f.cfg.TrustedLogHosts = []string{testLogHost, otherLogHost}
	f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}

	otherCert := f.mintCert(otherHost)
	otherClient := &fakeLogClient{
		token:   f.log.statusToken(t, tokenClaims{agentID: otherAgentID, ansName: otherHost, status: "ACTIVE", iat: testNow.Unix() - 60, exp: testNow.Unix() + 3600, identityCerts: []string{otherCert.fingerprint}}),
		receipt: f.log.receipt(t, eventJSON(t, "ansId", otherAgentID, otherHost), 1, 0, nil, testNow.Unix()),
	}

	version, err := models.ParseVersion(testVersion)
	require.NoError(t, err)

	f.dns = &countingBadges{badgeFinder: verify.NewMockDNSResolver().
		WithRecords(testHost, []verify.AnsBadgeRecord{{FormatVersion: "ans-badge1", Version: &version, URL: testBadgeURL}}).
		WithRecords("other.example.com", []verify.AnsBadgeRecord{{FormatVersion: "ans-badge1", Version: &version, URL: otherLogBase + "/v1/agents/" + otherAgentID}})}

	r := f.build()
	r.newLogClient = func(base string) (scitt.Client, error) {
		if base == otherLogBase {
			return otherClient, nil
		}

		return f.client, nil
	}

	for range breakerThreshold {
		_, err := f.verify()
		require.ErrorContains(t, err, "transparency log unreachable")
	}

	_, err = f.verify()
	require.ErrorContains(t, err, "circuit open")

	otherPayload := identityv1.CanonicalBytes(testRecordCID, otherHost, testSignedAt)

	ok, err := r.Verify(t.Context(), otherHost, []byte(signWithChain(t, otherCert.key, otherCert.der, otherPayload)), otherPayload)
	require.NoError(t, err)
	assert.True(t, ok, "the open circuit of one log must not affect another")
	assert.Equal(t, 2, otherClient.callCount())
}

// TestVerifyBreakerAttribution checks which fetch failures the breaker holds
// against the log: only a fetch that had its own budget and did not answer.
func TestVerifyBreakerAttribution(t *testing.T) {
	const timeout = 100 * time.Millisecond

	tests := []struct {
		name       string
		setup      func(f *fixture) context.Context
		wantErr    string
		wantCtxErr error
		wantOpen   bool
	}{
		{
			name: "slow dns leaves the log no budget",
			setup: func(f *fixture) context.Context {
				f.dns.delay = timeout - 20*time.Millisecond
				f.client.block = true

				return f.t.Context()
			},
			wantErr: "ans status-token: timed out",
		},
		{
			name: "hanging log takes a strike per fetch",
			setup: func(f *fixture) context.Context {
				f.client.block = true

				return f.t.Context()
			},
			wantErr:  "ans status-token: timed out",
			wantOpen: true,
		},
		{
			name: "caller cancellation never counts",
			setup: func(f *fixture) context.Context {
				ctx, cancel := context.WithCancel(f.t.Context())
				f.client.block = true
				f.client.onCall = cancel

				return ctx
			},
			wantCtxErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.cfg.Timeout = timeout
			ctx := tt.setup(f)
			r := f.build()

			for range breakerThreshold {
				_, err := r.Verify(ctx, f.subject, []byte(f.signature), f.payload)

				if tt.wantCtxErr != nil {
					require.ErrorIs(t, err, tt.wantCtxErr)
				} else {
					require.ErrorContains(t, err, tt.wantErr)
				}
			}

			_, open := r.breaker.openUntil(testLogHost, f.clock)
			assert.Equal(t, tt.wantOpen, open)
			assert.Empty(t, r.breaker.failures, "the breaker holds strikes against the log")
		})
	}
}

// TestVerifyRootKeyCache checks that fetched root keys serve an origin for
// rootKeysTTL, and that a key the cached set does not hold refreshes them.
func TestVerifyRootKeyCache(t *testing.T) {
	f := newFixture(t)
	unpinned(f)

	verifyOK := func(wantCalls int) {
		t.Helper()

		ok, err := f.verify()
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, wantCalls, f.client.callCount())
	}

	verifyOK(3) // root keys, token, receipt
	verifyOK(5) // cache hit: token, receipt

	f.clock = testNow.Add(rootKeysTTL - time.Second)

	verifyOK(7) // still cached

	f.clock = testNow.Add(rootKeysTTL)

	verifyOK(10) // expired: root keys again

	rotated := mintLog(t)
	f.log = rotated
	f.client.rootKeys = []string{rotated.rootKeyLine(t)}
	f.setToken(f.claims())
	f.setReceipt(eventJSON(t, "ansId", testAgentID, testAnsName))

	verifyOK(13) // token, unknown kid: root keys again, receipt
	verifyOK(15) // the refreshed keys are cached
}

func TestVerifyUnknownKeyIDWithoutRotation(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f *fixture)
		wantErr   string
		wantCalls int
	}{
		{
			name: "unpinned token by an unknown key is fetched again then fails",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.token = mintLog(f.t).statusToken(f.t, f.claims())
			},
			wantErr:   "ans status-token: signed by unknown key id",
			wantCalls: 3, // root keys, token, root keys again
		},
		{
			name: "unpinned receipt by an unknown key is fetched again then fails",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.receipt = mintLog(f.t).receipt(f.t, eventJSON(f.t, "ansId", testAgentID, testAnsName), 1, 0, nil, testNow.Unix())
			},
			wantErr:   "ans receipt: signed by unknown key id",
			wantCalls: 4, // root keys, token, receipt, root keys again
		},
		{
			name: "unpinned refetch that fails is reported as root keys",
			setup: func(f *fixture) {
				unpinned(f)
				f.client.token = mintLog(f.t).statusToken(f.t, f.claims())
				f.client.onCall = func() {
					if f.client.callCount() > 2 {
						f.client.rootKeysErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusBadGateway}
					}
				}
			},
			wantErr:   "ans root-keys: transparency log returned HTTP 502",
			wantCalls: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			tt.setup(f)

			ok, err := f.verify()
			require.ErrorContains(t, err, tt.wantErr)
			assert.False(t, ok)
			assert.Equal(t, tt.wantCalls, f.client.callCount())
		})
	}
}

func TestVerifyPinnedUnknownKeyIDWarns(t *testing.T) {
	logs := captureLogs(t)
	f := newFixture(t)
	other := mintLog(t)
	f.client.token = other.statusToken(t, f.claims())

	ok, err := f.verify()
	require.ErrorContains(t, err, "ans status-token: signed by unknown key id")
	assert.False(t, ok)
	assert.Equal(t, 1, f.client.callCount(), "pinned mode never fetches root keys")

	out := logs.String()
	assert.Contains(t, out, `msg="Transparency log signed with a key that is not pinned"`)
	assert.Contains(t, out, "level=WARN")
	assert.Contains(t, out, "kid="+hex.EncodeToString(other.kid[:]))
	assert.Contains(t, out, "logHost="+testLogHost)
}

// tlsLog serves the fixture's artifacts over TLS the way a real log would.
type tlsLog struct {
	server   *httptest.Server
	mu       sync.Mutex
	redirect bool
	moved    int
}

func startTLSLog(t *testing.T, f *fixture) *tlsLog {
	t.Helper()

	l := &tlsLog{}
	mux := http.NewServeMux()

	mux.HandleFunc("/root-keys", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(f.log.rootKeyLine(t) + "\n"))
	})
	mux.HandleFunc("/v1/agents/"+testAgentID+"/status-token", func(w http.ResponseWriter, r *http.Request) {
		if l.redirect {
			http.Redirect(w, r, "/v1/agents/"+testAgentID+"/status-token-moved", http.StatusFound)

			return
		}

		_, _ = w.Write(f.client.token)
	})
	mux.HandleFunc("/v1/agents/"+testAgentID+"/status-token-moved", func(w http.ResponseWriter, _ *http.Request) {
		l.mu.Lock()
		l.moved++
		l.mu.Unlock()

		_, _ = w.Write(f.client.token)
	})
	mux.HandleFunc("/v1/agents/"+testAgentID+"/receipt", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(f.client.receipt)
	})

	l.server = httptest.NewTLSServer(mux)
	t.Cleanup(l.server.Close)

	host := strings.TrimPrefix(l.server.URL, "https://")

	f.realClient = true
	f.cfg.TrustedLogHosts = []string{host}
	f.cfg.CAFile = writeFile(t, "ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: l.server.Certificate().Raw}))
	f.setBadgeURL(l.server.URL + "/v1/agents/" + testAgentID)

	return l
}

func writeFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	return path
}

func TestVerifyOverTLS(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *fixture, l *tlsLog)
		wantErr string
	}{
		{
			name: "pinned keys",
		},
		{
			name: "fetched keys",
			setup: func(f *fixture, _ *tlsLog) {
				unpinned(f)
			},
		},
		{
			name: "redirect from a trusted host is not followed",
			setup: func(_ *fixture, l *tlsLog) {
				l.redirect = true
			},
			wantErr: "ans status-token: transparency log returned HTTP 302",
		},
		{
			name: "untrusted server certificate",
			setup: func(f *fixture, _ *tlsLog) {
				f.cfg.CAFile = ""
			},
			wantErr: "ans status-token: TLS handshake failed",
		},
		{
			name: "connection refused",
			setup: func(_ *fixture, l *tlsLog) {
				l.server.Close()
			},
			wantErr: "ans status-token: transparency log unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			l := startTLSLog(t, f)

			if tt.setup != nil {
				tt.setup(f, l)
			}

			ok, err := f.verify()

			assert.Equal(t, 0, l.moved, "the redirect target was fetched")

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.False(t, ok)

				return
			}

			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}

func TestNew(t *testing.T) {
	line := mintLog(t).rootKeyLine(t)
	cert := mintIdentityCert(t, testAnsName, testNow, testNow.Add(time.Hour))
	caFile := writeFile(t, "ca.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.der}))
	junkFile := writeFile(t, "junk.pem", []byte("junk"))
	missingFile := filepath.Join(t.TempDir(), "missing.pem")

	pinned := func(mutate func(cfg *ansconfig.Config)) ansconfig.Config {
		cfg := ansconfig.Config{Enabled: true, TrustedLogHosts: []string{testLogHost}, RootKeys: []string{line}}
		if mutate != nil {
			mutate(&cfg)
		}

		return cfg
	}

	tests := []struct {
		name         string
		cfg          ansconfig.Config
		wantErr      string
		wantHosts    []string
		wantRootKeys []string
		wantPinned   int
	}{
		{name: "pinned keys", cfg: pinned(nil), wantHosts: []string{testLogHost}, wantRootKeys: []string{line}, wantPinned: 1},
		{name: "root key lines are trimmed", cfg: pinned(func(cfg *ansconfig.Config) { cfg.RootKeys = []string{" " + line + "\t"} }), wantRootKeys: []string{line}, wantPinned: 1},
		{name: "unpinned keys with opt-in", cfg: pinned(func(cfg *ansconfig.Config) { cfg.RootKeys = nil; cfg.AllowUnpinnedRootKeys = true })},
		{
			name: "hosts are normalized",
			cfg: pinned(func(cfg *ansconfig.Config) {
				cfg.TrustedLogHosts = []string{" LOG.Example.com:443 ", "Other.example.com:8443"}
			}),
			wantHosts:  []string{"log.example.com", "other.example.com:8443"},
			wantPinned: 1,
		},
		{name: "ca file", cfg: pinned(func(cfg *ansconfig.Config) { cfg.CAFile = caFile }), wantPinned: 1},
		{name: "dns server", cfg: pinned(func(cfg *ansconfig.Config) { cfg.DNSServer = "127.0.0.1:1" }), wantPinned: 1},
		{name: "disabled configuration", cfg: pinned(func(cfg *ansconfig.Config) { cfg.Enabled = false }), wantErr: "enabled configuration"},
		{name: "malformed root key", cfg: pinned(func(cfg *ansconfig.Config) { cfg.RootKeys = []string{"not-a-key"} }), wantErr: "ans: root_keys"},
		{name: "no trusted hosts", cfg: pinned(func(cfg *ansconfig.Config) { cfg.TrustedLogHosts = nil }), wantErr: "ans: trusted_log_hosts"},
		{name: "malformed trusted host", cfg: pinned(func(cfg *ansconfig.Config) { cfg.TrustedLogHosts = []string{"https://log.example.com"} }), wantErr: "must not contain a scheme or path"},
		{name: "unpinned keys without opt-in", cfg: pinned(func(cfg *ansconfig.Config) { cfg.RootKeys = nil }), wantErr: "root_keys is empty"},
		{name: "negative timeout", cfg: pinned(func(cfg *ansconfig.Config) { cfg.Timeout = -time.Second }), wantErr: "timeout must not be negative"},
		{name: "dns server without port", cfg: pinned(func(cfg *ansconfig.Config) { cfg.DNSServer = "127.0.0.1" }), wantErr: "dns_server must be host:port"},
		{name: "missing ca file", cfg: pinned(func(cfg *ansconfig.Config) { cfg.CAFile = missingFile }), wantErr: "ans: ca_file"},
		{name: "ca file without certificates", cfg: pinned(func(cfg *ansconfig.Config) { cfg.CAFile = junkFile }), wantErr: "contains no PEM certificates"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)

			r, err := New(tt.cfg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Empty(t, logs.String(), "a rejected configuration must not be logged as configured")

				return
			}

			require.NoError(t, err)
			require.NotNil(t, r)
			assert.Equal(t, "ans", r.Scheme())

			if tt.wantRootKeys != nil {
				assert.Equal(t, tt.wantRootKeys, r.cfg.RootKeys)
			}

			if tt.wantHosts != nil {
				hosts := make([]string, 0, len(r.trustedHosts))
				for host := range r.trustedHosts {
					hosts = append(hosts, host)
				}

				assert.ElementsMatch(t, tt.wantHosts, hosts)
			}

			out := logs.String()
			assert.Contains(t, out, `msg="ANS identity resolver configured"`)
			assert.Contains(t, out, "level=INFO")
			assert.Contains(t, out, "trustedLogHosts=")
			assert.Contains(t, out, "pinnedRootKeys="+strconv.Itoa(tt.wantPinned))
			assert.Contains(t, out, "timeout=10s")
			assert.NotContains(t, out, strings.Split(line, "+")[2], "key material must not be logged")
		})
	}
}

// TestHTTPLogClientRequiresHTTPS pins the SDK client's refusal of a plain
// HTTP origin; the badge URL gate never produces one, so this is the only
// path to that branch.
func TestHTTPLogClientRequiresHTTPS(t *testing.T) {
	f := newFixture(t)
	r := f.build()

	client, err := r.httpLogClient("http://" + testLogHost)
	require.ErrorContains(t, err, "ans log client:")
	assert.Nil(t, client)

	client, err = r.httpLogClient(testLogBase)
	require.NoError(t, err)
	assert.NotNil(t, client)
}

// TestLogClientKeepsTheFirstClientBuilt pins the outcome of two
// verifications building a client for the same origin at once: the one
// stored first serves both, so the breaker sees one client per host.
func TestLogClientKeepsTheFirstClientBuilt(t *testing.T) {
	f := newFixture(t)
	r := f.build()
	first := &breakerClient{inner: f.client, host: testLogHost}

	r.newLogClient = func(base string) (scitt.Client, error) {
		r.mu.Lock()
		r.clients[base] = first
		r.mu.Unlock()

		return &fakeLogClient{}, nil
	}

	target, err := parseBadgeURL(testBadgeURL)
	require.NoError(t, err)

	client, err := r.logClient(target)
	require.NoError(t, err)
	assert.Same(t, first, client)

	client, err = r.logClient(target)
	require.NoError(t, err)
	assert.Same(t, first, client)
}

func TestVerifyWithConfiguredDNSServer(t *testing.T) {
	tests := []struct {
		name   string
		server string
	}{
		{name: "server does not answer", server: "127.0.0.1:1"},
		{name: "server cannot be dialed", server: "127.0.0.1:99999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			f.cfg.DNSServer = tt.server
			f.cfg.Timeout = time.Second

			r, err := New(f.cfg)
			require.NoError(t, err)

			r.clock = func() time.Time { return f.clock }

			ok, err := r.Verify(t.Context(), f.subject, []byte(f.signature), f.payload)
			require.ErrorContains(t, err, "ans dns:")
			assert.False(t, ok)
		})
	}
}

// verifyOutcome is what one concurrent Verify call returned.
type verifyOutcome struct {
	ok  bool
	err error
}

// runConcurrentVerifies calls Verify on the fixture's resolver from n
// goroutines at once and returns every outcome in call order.
func runConcurrentVerifies(t *testing.T, f *fixture, n int) []verifyOutcome {
	t.Helper()

	r := f.build()
	outcomes := make([]verifyOutcome, n)

	var wg sync.WaitGroup

	for i := range n {
		wg.Go(func() {
			ok, err := r.Verify(t.Context(), f.subject, []byte(f.signature), f.payload)
			outcomes[i] = verifyOutcome{ok: ok, err: err}
		})
	}

	wg.Wait()

	return outcomes
}

// TestVerifyConcurrent runs many verifications against one Resolver at the
// same time: the race detector must stay quiet and every call must see the
// same outcome.
func TestVerifyConcurrent(t *testing.T) {
	const verifications = 32

	tests := []struct {
		name  string
		setup func(f *fixture)
		check func(t *testing.T, f *fixture, outcomes []verifyOutcome)
	}{
		{
			name: "every verification passes",
			check: func(t *testing.T, f *fixture, outcomes []verifyOutcome) {
				t.Helper()

				for i, outcome := range outcomes {
					require.NoError(t, outcome.err, "verification %d", i)
					assert.True(t, outcome.ok, "verification %d", i)
				}

				assert.Equal(t, 2*verifications, f.client.callCount())
				assert.Equal(t, verifications, f.dns.callCount())
			},
		},
		{
			name:  "every verification passes with fetched root keys",
			setup: unpinned,
			check: func(t *testing.T, f *fixture, outcomes []verifyOutcome) {
				t.Helper()

				for i, outcome := range outcomes {
					require.NoError(t, outcome.err, "verification %d", i)
					assert.True(t, outcome.ok, "verification %d", i)
				}

				assert.LessOrEqual(t, f.client.callCount(), 3*verifications)
				assert.GreaterOrEqual(t, f.client.callCount(), 2*verifications+1)
			},
		},
		{
			name: "every verification fails while the log is unreachable",
			setup: func(f *fixture) {
				f.client.tokenErr = &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}
			},
			check: func(t *testing.T, f *fixture, outcomes []verifyOutcome) {
				t.Helper()

				for i, outcome := range outcomes {
					require.Error(t, outcome.err, "verification %d", i)
					assert.False(t, outcome.ok, "verification %d", i)
				}

				_, open := f.resolver.breaker.openUntil(testLogHost, testNow)
				assert.True(t, open, "the circuit is closed after the log failed every verification")
				assert.LessOrEqual(t, f.client.callCount(), verifications)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t)
			if tt.setup != nil {
				tt.setup(f)
			}

			tt.check(t, f, runConcurrentVerifies(t, f, verifications))
		})
	}
}
