// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"crypto"
	"fmt"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resolveFixture is a complete, passing verification scenario that each case
// mutates before calling Resolve.
type resolveFixture struct {
	t           *testing.T
	cert        identityCert
	dns         *fakeTXT
	log         *fakeLog
	cfg         Config
	subject     string
	certificate []byte
	ctx         context.Context //nolint:containedctx // the case decides which context Resolve gets
	clock       *fakeClock
	resolver    *Resolver
}

func newResolveFixture(t *testing.T) *resolveFixture {
	t.Helper()

	cert := mintTestCert(t)

	return &resolveFixture{
		t:           t,
		cert:        cert,
		dns:         newFakeTXT(badgeTXT(testVersion, testBadgeURL)),
		log:         &fakeLog{status: activeStatus(fingerprintOf(cert.der))},
		cfg:         unpinnedConfig(),
		subject:     testAnsName,
		certificate: cert.der,
		ctx:         t.Context(),
		clock:       newFakeClock(testNow),
	}
}

func (f *resolveFixture) build() *Resolver {
	if f.resolver == nil {
		r, err := New(f.cfg, WithLookupTXT(f.dns.lookup), WithLogClient(f.log), WithClock(f.clock.Now))
		require.NoError(f.t, err)

		f.resolver = r
	}

	return f.resolver
}

func (f *resolveFixture) resolve() ([]crypto.PublicKey, error) {
	return f.build().Resolve(f.ctx, f.subject, f.certificate)
}

// requireCertificateKey asserts keys holds exactly the certificate's key.
func (f *resolveFixture) requireCertificateKey(keys []crypto.PublicKey) {
	f.t.Helper()

	require.Len(f.t, keys, 1)

	equaler, ok := keys[0].(interface{ Equal(crypto.PublicKey) bool })
	require.True(f.t, ok)
	assert.True(f.t, equaler.Equal(f.cert.key.Public()))
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(f *resolveFixture)
		wantErr   string
		wantNoDNS bool
		wantNoLog bool
		// Every error is marked as a verdict unless wantLive says it is left for
		// the caller to classify, with wantStatus the HTTP status its cause must
		// still carry.
		wantLive   bool
		wantStatus int
	}{
		{name: "valid claim"},
		{
			name:  "warning and deprecated agents are accepted",
			setup: func(f *resolveFixture) { f.log.status.State = StateDeprecated },
		},
		{
			name: "the log's spelling of the name and id may differ",
			setup: func(f *resolveFixture) {
				f.log.status.Name = "ans://v01.0.0.AGENT.example.com"
				f.log.status.AgentID = "5B1B6CC4-4B3E-4D4E-9A7D-2C1E7A6F9A10"
			},
		},
		{
			name: "certificate attested among others",
			setup: func(f *resolveFixture) {
				f.log.status.IdentityCertificates = [][32]byte{{1}, fingerprintOf(f.cert.der), {2}}
			},
		},
		{
			name:      "non-canonical subject",
			setup:     func(f *resolveFixture) { f.subject = "ans://v1.0.0.Agent.Example.com" },
			wantErr:   "ans name: subject is not in canonical form",
			wantNoDNS: true,
		},
		{
			name:      "not an ans subject",
			setup:     func(f *resolveFixture) { f.subject = "dns:agent.example.com" },
			wantErr:   "ans name: subject is not an ANS name",
			wantNoDNS: true,
		},
		{
			name:      "no certificate",
			setup:     func(f *resolveFixture) { f.certificate = nil },
			wantErr:   "ans certificate: claim carries no certificate",
			wantNoDNS: true,
		},
		{
			name:      "garbage certificate",
			setup:     func(f *resolveFixture) { f.certificate = []byte("not a certificate") },
			wantErr:   "ans certificate: parse certificate: ",
			wantNoDNS: true,
		},
		{
			name: "certificate for another agent",
			setup: func(f *resolveFixture) {
				f.certificate = mintIdentityCert(f.t, "ans://v1.0.0.other.example.com", testNow.Add(-time.Hour), testNow.Add(time.Hour)).der
			},
			wantErr:   "ans certificate: certificate does not name",
			wantNoDNS: true,
		},
		{
			name: "expired certificate",
			setup: func(f *resolveFixture) {
				f.certificate = mintIdentityCert(f.t, testAnsName, testNow.Add(-2*time.Hour), testNow.Add(-time.Hour)).der
			},
			wantErr:   "ans certificate: certificate is not valid at",
			wantNoDNS: true,
		},
		{
			name:      "no badge record",
			setup:     func(f *resolveFixture) { f.dns.set(testBadgeName) },
			wantErr:   "ans badge: no _ans-badge.agent.example.com record names version v1.0.0",
			wantNoLog: true,
		},
		{
			name: "dns server failure is left to the caller",
			setup: func(f *resolveFixture) {
				f.dns.err = &net.DNSError{Err: "server misbehaving", Name: testBadgeName, IsTemporary: true}
			},
			wantErr:   "ans badge: lookup _ans-badge.agent.example.com: ",
			wantNoLog: true,
			wantLive:  true,
		},
		{
			name: "dns hang ends after one timeout and is left to the caller",
			setup: func(f *resolveFixture) {
				f.cfg.Timeout = 50 * time.Millisecond
				f.dns.block = true
			},
			wantErr:   "ans badge: lookup _ans-badge.agent.example.com: ",
			wantNoLog: true,
			wantLive:  true,
		},
		{
			name: "badge points at an untrusted log",
			setup: func(f *resolveFixture) {
				f.dns.set(testBadgeName, badgeTXT(testVersion, "https://other.example.com/v1/agents/"+testAgentID))
			},
			wantErr:   "ans badge: log host \"other.example.com\" is not a trusted transparency log",
			wantNoLog: true,
		},
		{
			name: "log outage keeps its cause while the caller waits",
			setup: func(f *resolveFixture) {
				f.log.err = fmt.Errorf("ans log: fetch status token: %w", &safefetch.StatusError{URL: tokenURL, Code: http.StatusServiceUnavailable})
			},
			wantErr:    "ans log: fetch status token: ",
			wantLive:   true,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name: "log budget expiry keeps its cause while the caller waits",
			setup: func(f *resolveFixture) {
				f.cfg.Timeout = 50 * time.Millisecond
				f.log.block = true
			},
			wantErr:  "ans log: fetch status token: ",
			wantLive: true,
		},
		{
			name:    "status token for another agent",
			setup:   func(f *resolveFixture) { f.log.status.AgentID = otherAgentID },
			wantErr: "ans log: status token names agent \"" + otherAgentID + "\", expected \"" + testAgentID + "\"",
		},
		{
			name:    "status token for another name",
			setup:   func(f *resolveFixture) { f.log.status.Name = "ans://v2.0.0.agent.example.com" },
			wantErr: "ans log: status token names \"ans://v2.0.0.agent.example.com\", expected \"ans://v1.0.0.agent.example.com\"",
		},
		{
			name:    "revoked agent",
			setup:   func(f *resolveFixture) { f.log.status.State = State("REVOKED") },
			wantErr: "ans log: agent status \"REVOKED\" does not allow use",
		},
		{
			name:    "unknown status",
			setup:   func(f *resolveFixture) { f.log.status.State = State("SOMETHING_NEW") },
			wantErr: "ans log: agent status \"SOMETHING_NEW\" does not allow use",
		},
		{
			name:    "certificate not attested",
			setup:   func(f *resolveFixture) { f.log.status.IdentityCertificates = [][32]byte{{1}, {2}} },
			wantErr: "ans log: the certificate is not among the agent's valid identity certificates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newResolveFixture(t)
			if tt.setup != nil {
				tt.setup(f)
			}

			keys, err := f.resolve()

			if tt.wantNoDNS {
				assert.Zero(t, f.dns.calls.Load(), "DNS was queried")
			}

			if tt.wantNoDNS || tt.wantNoLog {
				assert.Zero(t, f.log.calls.Load(), "the log was called")
			}

			if tt.wantErr == "" {
				require.NoError(t, err)
				f.requireCertificateKey(keys)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, keys)

			if tt.wantLive {
				requireLive(t, err)
			} else {
				requireFinal(t, err)
			}

			if tt.wantStatus != 0 {
				var statusErr *safefetch.StatusError

				require.ErrorAs(t, err, &statusErr)
				assert.Equal(t, tt.wantStatus, statusErr.Code)
			}
		})
	}
}

// requireFinal asserts err is marked as a verdict, so a caller never classifies
// it as transient whatever cause it carries.
func requireFinal(t *testing.T, err error) {
	t.Helper()

	require.ErrorIs(t, err, resolvers.ErrFinal, "not marked final: %v", err)
}

// requireLive asserts err is left for the caller to classify by its cause: a
// failure to reach the publisher's DNS or the trusted log.
func requireLive(t *testing.T, err error) {
	t.Helper()

	require.NotErrorIs(t, err, resolvers.ErrFinal, "marked final: %v", err)
}

// A failure to reach the network is never a verdict, whichever side's deadline
// ended it: the caller sees the cause and keeps its stored result or not.
func TestResolveLeavesNetworkFailuresToTheCaller(t *testing.T) {
	t.Run("log budget expiry", func(t *testing.T) {
		f := newResolveFixture(t)
		f.cfg.Timeout = 50 * time.Millisecond
		f.log.block = true

		_, err := f.resolve()
		require.ErrorIs(t, err, context.DeadlineExceeded)
		requireLive(t, err)
	})

	t.Run("caller gives up during the log stage", func(t *testing.T) {
		f := newResolveFixture(t)
		f.log.block = true

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		f.ctx = ctx

		_, err := f.resolve()
		require.ErrorContains(t, err, "ans log: fetch status token: ")
		require.ErrorIs(t, err, context.DeadlineExceeded)
		requireLive(t, err)
	})

	t.Run("caller already gave up", func(t *testing.T) {
		for name, done := range map[string]func(context.Context) (context.Context, context.CancelFunc){
			"cancelled": func(ctx context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(ctx)
				cancel()

				return ctx, cancel
			},
			"expired": func(ctx context.Context) (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithTimeout(ctx, time.Nanosecond)
				<-ctx.Done()

				return ctx, cancel
			},
		} {
			t.Run(name, func(t *testing.T) {
				f := newResolveFixture(t)

				ctx, cancel := done(t.Context())
				defer cancel()

				f.ctx = ctx

				_, err := f.resolve()
				require.ErrorContains(t, err, "ans badge: lookup")
				require.ErrorIs(t, err, ctx.Err())
				requireLive(t, err)
				assert.Zero(t, f.log.calls.Load(), "the log was called")
			})
		}
	})
}

func TestResolveGivesTheLogItsOwnBudget(t *testing.T) {
	f := newResolveFixture(t)
	f.cfg.Timeout = 200 * time.Millisecond
	f.dns.delay = 120 * time.Millisecond
	f.log.status.State = State("REVOKED")

	_, err := f.resolve()
	require.ErrorContains(t, err, "ans log: agent status \"REVOKED\" does not allow use")
	requireFinal(t, err)

	_, remaining := f.log.seen()
	assert.Greater(t, remaining, 150*time.Millisecond, "slow DNS ate into the log's budget")
}

func TestResolvePassesTheGatedTarget(t *testing.T) {
	f := newResolveFixture(t)
	f.dns.set(testBadgeName, badgeTXT(testVersion, "https://LOG.Example.com:443/v1/agents/"+testAgentID))

	_, err := f.resolve()
	require.NoError(t, err)

	target, _ := f.log.seen()
	assert.Equal(t, TrustedLog{Origin: testLogOrigin, AgentID: testAgentID}, target)
}

// A log that is down is not asked again for the same subject within the
// lifetime, and the kept failure still carries its cause.
func TestResolveMemoizesALogOutage(t *testing.T) {
	f := newResolveFixture(t)
	f.log.err = fmt.Errorf("ans log: fetch status token: %w", &safefetch.StatusError{URL: tokenURL, Code: http.StatusServiceUnavailable})

	for range 3 {
		_, err := f.build().Resolve(f.ctx, testAnsName, mintTestCert(t).der)
		requireLive(t, err)

		var statusErr *safefetch.StatusError

		require.ErrorAs(t, err, &statusErr)
	}

	assert.Equal(t, int64(1), f.log.calls.Load())
}

// Resolve is safe for concurrent use, and concurrent claims for one subject
// cost one lookup: they wait for the one in flight or read the memo.
func TestResolveConcurrently(t *testing.T) {
	f := newResolveFixture(t)
	r := f.build()

	const callers = 16

	results := make(chan error, callers)

	var wg sync.WaitGroup

	for range callers {
		wg.Go(func() {
			keys, err := r.Resolve(f.ctx, testAnsName, f.certificate)
			if err == nil && len(keys) != 1 {
				err = fmt.Errorf("got %d keys, want 1", len(keys))
			}

			results <- err
		})
	}

	wg.Wait()
	close(results)

	for err := range results {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(1), f.dns.calls.Load())
	assert.Equal(t, int64(1), f.log.calls.Load())
}

// Claims that arrive while the subject's lookup is running wait for it rather
// than each asking DNS and the log.
func TestResolveSharesTheLookupInFlight(t *testing.T) {
	f := newResolveFixture(t)
	f.log.hold()
	r := f.build()

	const callers = 16

	results := make(chan error, callers)

	var wg sync.WaitGroup

	for range callers {
		wg.Go(func() {
			keys, err := r.Resolve(f.ctx, testAnsName, f.certificate)
			if err == nil && len(keys) != 1 {
				err = fmt.Errorf("got %d keys, want 1", len(keys))
			}

			results <- err
		})
	}

	require.Eventually(t, func() bool { return f.log.calls.Load() == 1 }, time.Second, time.Millisecond, "no lookup reached the log")
	f.log.release()
	wg.Wait()
	close(results)

	for err := range results {
		require.NoError(t, err)
	}

	assert.Equal(t, int64(1), f.dns.calls.Load())
	assert.Equal(t, int64(1), f.log.calls.Load())
}

// A waiter gives up on its own deadline, not the leader's, without starting a
// lookup of its own, and the leader's answer still serves the next claim.
func TestResolveWaiterHonoursItsOwnContext(t *testing.T) {
	f := newResolveFixture(t)
	f.log.hold()
	r := f.build()

	leader := make(chan error, 1)

	go func() {
		_, err := r.Resolve(f.ctx, testAnsName, f.certificate)
		leader <- err
	}()

	require.Eventually(t, func() bool { return f.log.calls.Load() == 1 }, time.Second, time.Millisecond, "the leader did not reach the log")

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	_, err := r.Resolve(ctx, testAnsName, f.certificate)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorContains(t, err, "ans: waiting for the lookup of "+testAnsName+": ")
	requireLive(t, err)
	assert.Equal(t, int64(1), f.log.calls.Load(), "the waiter started a lookup of its own")

	f.log.release()
	require.NoError(t, <-leader)

	keys, err := r.Resolve(f.ctx, testAnsName, f.certificate)
	require.NoError(t, err)
	f.requireCertificateKey(keys)
	assert.Equal(t, int64(1), f.log.calls.Load(), "the leader's answer was not kept")
}

// When the leading caller gives up, its answer is not kept, and a waiter still
// waiting looks the subject up itself.
func TestResolveWaiterTakesOverWhenTheLeaderGivesUp(t *testing.T) {
	f := newResolveFixture(t)
	f.log.hold()
	r := f.build()

	leaderCtx, cancelLeader := context.WithCancel(t.Context())
	defer cancelLeader()

	leader := make(chan error, 1)

	go func() {
		_, err := r.Resolve(leaderCtx, testAnsName, f.certificate)
		leader <- err
	}()

	require.Eventually(t, func() bool { return f.log.calls.Load() == 1 }, time.Second, time.Millisecond, "the leader did not reach the log")

	waiter := make(chan error, 1)

	go func() {
		keys, err := r.Resolve(f.ctx, testAnsName, f.certificate)
		if err == nil && len(keys) != 1 {
			err = fmt.Errorf("got %d keys, want 1", len(keys))
		}

		waiter <- err
	}()

	cancelLeader()

	err := <-leader
	require.ErrorIs(t, err, context.Canceled)
	requireLive(t, err)

	require.Eventually(t, func() bool { return f.log.calls.Load() == 2 }, time.Second, time.Millisecond, "the waiter did not look the subject up itself")
	f.log.release()
	require.NoError(t, <-waiter)
}

// The claims of one subject cost one DNS lookup and one log fetch between
// them, whatever certificates they carry, so a stack of claims attached by a
// third party cannot exhaust a record's budget.
func TestResolveMemoizesTheAttestationPerSubject(t *testing.T) {
	f := newResolveFixture(t)

	keys, err := f.resolve()
	require.NoError(t, err)
	f.requireCertificateKey(keys)

	for range 20 {
		other := mintTestCert(t)

		_, err := f.build().Resolve(f.ctx, testAnsName, other.der)
		require.ErrorContains(t, err, "not among the agent's valid identity certificates")
	}

	assert.Equal(t, int64(1), f.dns.calls.Load())
	assert.Equal(t, int64(1), f.log.calls.Load())

	// Another subject is its own lookup.
	otherName := "ans://v1.0.0.other.example.com"

	f.dns.set(badgeRecordPrefix+"other.example.com", badgeTXT(testVersion, testBadgeURL))

	_, err = f.build().Resolve(f.ctx, otherName, mintIdentityCert(t, otherName, testNow.Add(-time.Hour), testNow.Add(time.Hour)).der)
	require.ErrorContains(t, err, "status token names")
	assert.Equal(t, int64(2), f.dns.calls.Load())
	assert.Equal(t, int64(2), f.log.calls.Load())
}

func TestResolveMemoExpires(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(f *resolveFixture)
		advance func(f *resolveFixture)
	}{
		{
			name:    "after the configured lifetime",
			advance: func(f *resolveFixture) { f.clock.advance(DefaultStatusCacheTTL) },
		},
		{
			name:    "after a configured lifetime",
			setup:   func(f *resolveFixture) { f.cfg.StatusCacheTTL = MinStatusCacheTTL },
			advance: func(f *resolveFixture) { f.clock.advance(MinStatusCacheTTL) },
		},
		{
			name: "when the log's statement expires before the lifetime does",
			setup: func(f *resolveFixture) {
				f.cfg.StatusCacheTTL = 10 * time.Minute
				f.log.status.ExpiresAt = testNow.Add(time.Second)
			},
			advance: func(f *resolveFixture) { f.clock.advance(time.Second + DefaultClockSkew + time.Second) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newResolveFixture(t)
			if tt.setup != nil {
				tt.setup(f)
			}

			for range 2 {
				_, err := f.resolve()
				require.NoError(t, err)
			}

			assert.Equal(t, int64(1), f.log.calls.Load(), "reused before expiry")

			tt.advance(f)

			_, err := f.resolve()
			require.NoError(t, err)
			assert.Equal(t, int64(2), f.log.calls.Load(), "fetched again after expiry")
		})
	}
}

// The log's tokens always carry an expiry, so a statement without one comes
// from another client and is never reused.
func TestResolveDoesNotReuseAStatementWithoutExpiry(t *testing.T) {
	f := newResolveFixture(t)
	f.log.status.ExpiresAt = time.Time{}

	for range 2 {
		keys, err := f.resolve()
		require.NoError(t, err)
		f.requireCertificateKey(keys)
	}

	assert.Equal(t, int64(2), f.log.calls.Load())
}

// Expired entries are swept once per lifetime, not on every claim, and an
// expired entry a claim asks for is dropped on the spot.
func TestResolveSweepsTheMemoOncePerLifetime(t *testing.T) {
	f := newResolveFixture(t)
	r := f.build()

	resolve := func(host string) {
		name := ansScheme + testVersion + "." + host
		f.dns.set(badgeRecordPrefix+host, badgeTXT(testVersion, testBadgeURL))

		_, _ = r.Resolve(f.ctx, name, mintIdentityCert(t, name, testNow.Add(-time.Hour), testNow.Add(time.Hour)).der)
	}

	resolve("a.example.com")
	f.clock.advance(time.Second)
	resolve("b.example.com")
	assert.Len(t, r.memo, 2)

	f.clock.advance(DefaultStatusCacheTTL - time.Second)
	resolve("c.example.com")
	assert.Len(t, r.memo, 2, "the sweep due with this claim dropped the expired entry")
	assert.NotContains(t, r.memo, ansScheme+testVersion+".a.example.com")

	f.clock.advance(2 * time.Second)
	resolve("d.example.com")
	assert.Len(t, r.memo, 3, "no sweep is due, so the expired entry stays")
	assert.Contains(t, r.memo, ansScheme+testVersion+".b.example.com")

	calls := f.dns.calls.Load()

	resolve("b.example.com")
	assert.Equal(t, calls+1, f.dns.calls.Load(), "the expired entry was not reused")
	assert.Len(t, r.memo, 3, "the expired entry was replaced, not kept beside the new one")
}

func TestResolveMemoizesErrorsAndEvictsExpiredEntries(t *testing.T) {
	f := newResolveFixture(t)
	f.dns.set(testBadgeName)

	for range 3 {
		_, err := f.resolve()
		require.ErrorContains(t, err, "no _ans-badge")
	}

	assert.Equal(t, int64(1), f.dns.calls.Load(), "a final answer is reused too")

	otherName := "ans://v1.0.0.other.example.com"

	f.dns.set(badgeRecordPrefix+"other.example.com", badgeTXT(testVersion, testBadgeURL))

	_, err := f.build().Resolve(f.ctx, otherName, mintIdentityCert(t, otherName, testNow.Add(-time.Hour), testNow.Add(time.Hour)).der)
	require.Error(t, err)
	assert.Len(t, f.resolver.memo, 2)

	f.clock.advance(DefaultStatusCacheTTL)

	_, err = f.resolve()
	require.Error(t, err)
	assert.Len(t, f.resolver.memo, 1, "expired entries are dropped when a new one is kept")
}

func TestResolveDoesNotMemoizeAnAnswerTheCallerDidNotWaitFor(t *testing.T) {
	f := newResolveFixture(t)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	f.ctx = ctx

	_, err := f.resolve()
	require.Error(t, err)

	f.ctx = t.Context()

	keys, err := f.resolve()
	require.NoError(t, err)
	f.requireCertificateKey(keys)
	assert.Equal(t, int64(2), f.dns.calls.Load(), "the live call looked the subject up itself")
	assert.Equal(t, int64(1), f.log.calls.Load())
}

func TestNew(t *testing.T) {
	log := mintLog(t)

	tests := []struct {
		name       string
		cfg        Config
		wantErr    string
		wantPinned []PinnedKey
	}{
		{
			name:       "pinned keys are parsed",
			cfg:        Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{" " + log.rootKeyLine(t) + " "}},
			wantPinned: []PinnedKey{{Name: log.name, KeyID: fmt.Sprintf("%x", log.kid)}},
		},
		{
			name: "unpinned",
			cfg:  unpinnedConfig(),
		},
		{
			name:    "no trusted hosts",
			cfg:     Config{RootKeys: []string{log.rootKeyLine(t)}},
			wantErr: "ans: trusted_log_hosts",
		},
		{
			name:    "pinned and unpinned together",
			cfg:     Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{log.rootKeyLine(t)}, AllowUnpinnedRootKeys: true},
			wantErr: "cannot both be set",
		},
		{
			name:    "malformed root key",
			cfg:     Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{"example-log+zz+AjBZ"}},
			wantErr: "ans: root_keys: ",
		},
		{
			name:    "duplicate root key",
			cfg:     Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{log.rootKeyLine(t), log.rootKeyLine(t)}},
			wantErr: "ans: root_keys: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := New(tt.cfg)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantPinned, r.PinnedKeys())
		})
	}
}

// New wires the configured key policy, lifetimes, fetcher and clock into the
// default log client.
func TestNewWiresTheDefaultLogClient(t *testing.T) {
	log := mintLog(t)
	fetcher := newFakeFetcher()
	clock := newFakeClock(testNow)

	r, err := New(Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{log.rootKeyLine(t)}, RootKeysTTL: time.Minute, ClockSkew: 2 * time.Second},
		WithFetcher(fetcher), WithClock(clock.Now))
	require.NoError(t, err)

	client, ok := r.log.(*scittLogClient)
	require.True(t, ok)
	assert.NotNil(t, client.pinned)
	assert.Equal(t, time.Minute, client.ttl)
	assert.Equal(t, 2*time.Second, client.skew)
	assert.Same(t, fetcher, client.fetch)
	assert.Equal(t, testNow, client.clock(), "the client shares the resolver's clock")

	r, err = New(unpinnedConfig())
	require.NoError(t, err)

	client, ok = r.log.(*scittLogClient)
	require.True(t, ok)
	assert.Nil(t, client.pinned)
	assert.Equal(t, DefaultRootKeysTTL, client.ttl)
	assert.Equal(t, DefaultClockSkew, client.skew)
	assert.NotNil(t, client.fetch, "an SSRF-guarded client by default")
	assert.Empty(t, r.PinnedKeys())
}

func TestResolverSatisfiesContract(t *testing.T) {
	var _ resolvers.Resolver = (*Resolver)(nil)
}
