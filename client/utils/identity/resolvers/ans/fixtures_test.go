// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/agntcy/dir/utils/safefetch"
	"github.com/stretchr/testify/require"
)

// The synthetic agent and log every test starts from.
const (
	testHost      = "agent.example.com"
	testVersion   = "v1.0.0"
	testAnsName   = "ans://v1.0.0.agent.example.com"
	testAgentID   = "5b1b6cc4-4b3e-4d4e-9a7d-2c1e7a6f9a10"
	otherAgentID  = "0e4c1d2a-7f6b-4c3d-8e9f-a1b2c3d4e5f6"
	testLogHost   = "log.example.com"
	testLogOrigin = "https://" + testLogHost
	testBadgeURL  = testLogOrigin + "/v1/agents/" + testAgentID
	testBadgeName = badgeRecordPrefix + testHost
)

var testNow = time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

// badgeTXT renders a badge TXT record.
func badgeTXT(version, url string) string {
	return "v=" + badgeFormat + "; version=" + version + "; url=" + url
}

// unpinnedConfig trusts the test log over TLS alone, which is enough when the
// log client is a fake.
func unpinnedConfig() Config {
	return Config{TrustedLogHosts: []string{testLogHost}, AllowUnpinnedRootKeys: true}
}

// fakeClock is a settable clock.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(now time.Time) *fakeClock {
	return &fakeClock{now: now}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(d)
}

// fakeTXT scripts DNS TXT answers by name and counts lookups. Like the
// system resolver it fails at once under a done context, and it can hold an
// answer back for a while or until the context ends.
type fakeTXT struct {
	mu      sync.Mutex
	records map[string][]string
	err     error
	delay   time.Duration
	block   bool
	calls   atomic.Int64
}

func newFakeTXT(records ...string) *fakeTXT {
	return &fakeTXT{records: map[string][]string{testBadgeName: records}}
}

func (f *fakeTXT) lookup(ctx context.Context, name string) ([]string, error) {
	f.calls.Add(1)

	if err := ctx.Err(); err != nil {
		return nil, dnsError(name, err)
	}

	f.mu.Lock()
	delay, block, err := f.delay, f.block, f.err
	records, known := f.records[name]
	f.mu.Unlock()

	if block {
		<-ctx.Done()

		return nil, dnsError(name, ctx.Err())
	}

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, dnsError(name, ctx.Err())
		}
	}

	if err != nil {
		return nil, err
	}

	if !known {
		return nil, &net.DNSError{Err: "no such host", Name: name, IsNotFound: true}
	}

	return records, nil
}

// dnsError is the error the system resolver returns when its context ends.
func dnsError(name string, cause error) error {
	return &net.DNSError{Err: cause.Error(), Name: name, IsTimeout: errors.Is(cause, context.DeadlineExceeded), UnwrapErr: cause}
}

// set makes name answer with records; with none, the name exists but holds
// no badge record.
func (f *fakeTXT) set(name string, records ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.records[name] = append([]string{}, records...)
}

// fakeLog scripts a trusted log's statement and counts calls. It can block
// until the context ends, and it records the budget the caller gave it.
type fakeLog struct {
	mu        sync.Mutex
	status    *Status
	err       error
	block     bool
	calls     atomic.Int64
	remaining time.Duration
	target    TrustedLog
}

func (f *fakeLog) Status(ctx context.Context, log TrustedLog) (*Status, error) {
	f.calls.Add(1)

	f.mu.Lock()

	f.target = log
	if deadline, ok := ctx.Deadline(); ok {
		f.remaining = time.Until(deadline)
	}

	status, err, block := f.status, f.err, f.block

	f.mu.Unlock()

	if block {
		<-ctx.Done()

		return nil, fmt.Errorf("ans log: fetch status token: %w", ctx.Err())
	}

	if err != nil {
		return nil, err
	}

	copied := *status
	copied.IdentityCertificates = append([][32]byte(nil), status.IdentityCertificates...)

	return &copied, nil
}

func (f *fakeLog) seen() (TrustedLog, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.target, f.remaining
}

// activeStatus is the log's statement about the test agent, attesting the
// given certificates.
func activeStatus(fingerprints ...[32]byte) *Status {
	return &Status{
		AgentID:              testAgentID,
		Name:                 testAnsName,
		State:                StateActive,
		ExpiresAt:            testNow.Add(time.Hour),
		IdentityCertificates: fingerprints,
	}
}

// fakeFetcher serves canned responses by URL, a StatusError for any other,
// and counts requests.
type fakeFetcher struct {
	mu        sync.Mutex
	docs      map[string][]byte
	errs      map[string]error
	requested []string
}

func newFakeFetcher() *fakeFetcher {
	return &fakeFetcher{docs: map[string][]byte{}, errs: map[string]error{}}
}

func (f *fakeFetcher) Get(_ context.Context, url string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.requested = append(f.requested, url)

	if err, ok := f.errs[url]; ok {
		return nil, err
	}

	if doc, ok := f.docs[url]; ok {
		return doc, nil
	}

	return nil, &safefetch.StatusError{URL: url, Code: 404}
}

func (f *fakeFetcher) serve(url string, doc []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.errs, url)

	f.docs[url] = doc
}

func (f *fakeFetcher) fail(url string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.errs[url] = err
}

// count returns how often url was requested.
func (f *fakeFetcher) count(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	n := 0

	for _, requested := range f.requested {
		if requested == url {
			n++
		}
	}

	return n
}

// newTestResolver builds a resolver over the fakes with a clock pinned to testNow.
func newTestResolver(t *testing.T, cfg Config, dns *fakeTXT, log TrustedLogClient) *Resolver {
	t.Helper()

	r, err := New(cfg, WithLookupTXT(dns.lookup), WithLogClient(log), WithClock(newFakeClock(testNow).Now))
	require.NoError(t, err)

	return r
}
