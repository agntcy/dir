// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expiredContext returns a context whose deadline has passed.
func expiredContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	t.Cleanup(cancel)

	<-ctx.Done()

	return ctx
}

// liveContext returns the test's own context, which stays live.
func liveContext(t *testing.T) context.Context {
	t.Helper()

	return t.Context()
}

// canceledContext returns a context its caller has canceled.
func canceledContext(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	return ctx
}

// childOf returns a fetch context derived from verifyCtx with a deadline far
// away, so it ends only when verifyCtx does.
func childOf(t *testing.T, verifyCtx context.Context) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(verifyCtx, time.Hour)
	t.Cleanup(cancel)

	return ctx
}

func requestErr(cause error) error {
	return &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: &url.Error{
		Op: "Get", URL: testBadgeURL + "/status-token", Err: cause,
	}}
}

func TestCountsAsStrike(t *testing.T) {
	live := func(t *testing.T) (context.Context, context.Context) {
		t.Helper()

		return t.Context(), t.Context()
	}
	fetchExpired := func(t *testing.T) (context.Context, context.Context) {
		t.Helper()

		return expiredContext(t), t.Context()
	}
	verifyExpired := func(t *testing.T) (context.Context, context.Context) {
		t.Helper()

		verifyCtx := expiredContext(t)

		return childOf(t, verifyCtx), verifyCtx
	}
	verifyCanceled := func(t *testing.T) (context.Context, context.Context) {
		t.Helper()

		verifyCtx := canceledContext(t)

		return childOf(t, verifyCtx), verifyCtx
	}

	tests := []struct {
		name     string
		err      error
		contexts func(t *testing.T) (context.Context, context.Context)
		want     bool
	}{
		{name: "fetch deadline fired while the verification had budget", err: requestErr(context.DeadlineExceeded), contexts: fetchExpired, want: true},
		{name: "bare deadline from the fetch deadline", err: fmt.Errorf("fake: %w", context.DeadlineExceeded), contexts: fetchExpired, want: true},
		{name: "verification deadline fired", err: requestErr(context.DeadlineExceeded), contexts: verifyExpired},
		{name: "deadline without an expired context", err: requestErr(context.DeadlineExceeded), contexts: live},
		{name: "caller canceled", err: requestErr(context.Canceled), contexts: verifyCanceled},
		{name: "bare cancellation", err: context.Canceled, contexts: verifyCanceled},
		{name: "connection refused", err: requestErr(errBoom), contexts: live, want: true},
		{name: "tls handshake failed", err: requestErr(&tls.CertificateVerificationError{Err: x509.UnknownAuthorityError{}}), contexts: live, want: true},
		{name: "http 503", err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}, contexts: live},
		{name: "http 404", err: &scitt.TransportError{Type: scitt.TransportErrNotFound, StatusCode: http.StatusNotFound}, contexts: live},
		{name: "invalid response", err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "response body exceeds maximum size"}, contexts: live},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fetchCtx, verifyCtx := tt.contexts(t)

			assert.Equal(t, tt.want, countsAsStrike(tt.err, fetchCtx, verifyCtx))
		})
	}
}

func TestCauseText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "transport error exposes its cause",
			err:  &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom},
			want: "boom",
		},
		{
			name: "transport error without a cause prints itself",
			err:  &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable, Message: "unexpected status code 503"},
			want: "HTTP error (503): unexpected status code 503",
		},
		{
			name: "other errors print themselves",
			err:  fmt.Errorf("fake log client: %w", context.DeadlineExceeded),
			want: "fake log client: context deadline exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, causeText(tt.err))
		})
	}
}

// TestLogFailureLevel checks which failed fetches are worth a WARN: those
// that say the log is down or broken, not those that state a verdict about
// the agent.
func TestLogFailureLevel(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		strike    bool
		wantLevel string
		wantAttrs []string
	}{
		{name: "connection failure", err: requestErr(errBoom), strike: true, wantLevel: "WARN", wantAttrs: []string{"strike=true", "cause="}},
		{name: "http 503", err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}, wantLevel: "WARN", wantAttrs: []string{"statusCode=503"}},
		{name: "http 500", err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusInternalServerError}, wantLevel: "WARN", wantAttrs: []string{"statusCode=500"}},
		{name: "http 404", err: &scitt.TransportError{Type: scitt.TransportErrNotFound, StatusCode: http.StatusNotFound}, wantLevel: "DEBUG", wantAttrs: []string{"statusCode=404"}},
		{name: "http 410", err: &scitt.TransportError{Type: scitt.TransportErrAgentTerminal, StatusCode: http.StatusGone}, wantLevel: "DEBUG", wantAttrs: []string{"statusCode=410"}},
		{name: "http 302", err: &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusFound}, wantLevel: "DEBUG", wantAttrs: []string{"statusCode=302"}},
		{name: "caller cancellation", err: requestErr(context.Canceled), wantLevel: "DEBUG", wantAttrs: []string{"strike=false"}},
		{name: "verification budget exhausted", err: requestErr(context.DeadlineExceeded), wantLevel: "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)
			c := &breakerClient{host: testLogHost}

			c.logFailure(stageStatusToken, testAgentID, tt.err, time.Second, tt.strike)

			out := logs.String()
			assert.Contains(t, out, "level="+tt.wantLevel)
			assert.Contains(t, out, `msg="Transparency log fetch failed"`)
			assert.Contains(t, out, "logHost="+testLogHost)
			assert.Contains(t, out, "agentId="+testAgentID)
			assert.Contains(t, out, "stage=\"ans status-token\"")

			for _, attr := range tt.wantAttrs {
				assert.Contains(t, out, attr)
			}
		})
	}
}

func TestLogFailureWithoutAnAgent(t *testing.T) {
	logs := captureLogs(t)
	c := &breakerClient{host: testLogHost}

	c.logFailure(stageRootKeys, "", requestErr(errBoom), time.Second, true)

	assert.NotContains(t, logs.String(), "agentId=")
	assert.Contains(t, logs.String(), "stage=\"ans root-keys\"")
}

// fakeLogClient is a scripted scitt.Client that counts calls, can run a hook
// on each call, and can hold a request open until its context ends.
type fakeLogClient struct {
	mu          sync.Mutex
	rootKeys    []string
	rootKeysErr error
	token       []byte
	tokenErr    error
	receipt     []byte
	receiptErr  error
	block       bool
	onCall      func()
	calls       int
	agentIDs    []string
}

func (c *fakeLogClient) FetchRootKeys(ctx context.Context) ([]string, error) {
	if err := c.enter(ctx, ""); err != nil {
		return nil, err
	}

	if c.rootKeysErr != nil {
		return nil, c.rootKeysErr
	}

	return c.rootKeys, nil
}

func (c *fakeLogClient) FetchStatusToken(ctx context.Context, agentID string) ([]byte, error) {
	if err := c.enter(ctx, agentID); err != nil {
		return nil, err
	}

	if c.tokenErr != nil {
		return nil, c.tokenErr
	}

	return c.token, nil
}

func (c *fakeLogClient) FetchReceipt(ctx context.Context, agentID string) ([]byte, error) {
	if err := c.enter(ctx, agentID); err != nil {
		return nil, err
	}

	if c.receiptErr != nil {
		return nil, c.receiptErr
	}

	return c.receipt, nil
}

func (c *fakeLogClient) enter(ctx context.Context, agentID string) error {
	c.mu.Lock()
	c.calls++

	if agentID != "" {
		c.agentIDs = append(c.agentIDs, agentID)
	}

	block, onCall := c.block, c.onCall
	c.mu.Unlock()

	if onCall != nil {
		onCall()
	}

	if block {
		<-ctx.Done()

		return fmt.Errorf("fake log client: %w", ctx.Err())
	}

	return nil
}

func (c *fakeLogClient) callCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.calls
}

// newBreakerClient wraps inner the way the resolver does, with a short
// per-fetch deadline and a fixed clock.
func newBreakerClient(inner scitt.Client, b *breaker, timeout time.Duration) *breakerClient {
	return &breakerClient{
		inner:   inner,
		host:    testLogHost,
		breaker: b,
		clock:   func() time.Time { return testNow },
		timeout: timeout,
	}
}

func TestBreakerClientReturnsWhatTheLogAnswers(t *testing.T) {
	inner := &fakeLogClient{rootKeys: []string{"line"}, token: []byte("token"), receipt: []byte("receipt")}
	c := newBreakerClient(inner, newBreaker(), time.Second)

	keys, err := c.FetchRootKeys(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"line"}, keys)

	token, err := c.FetchStatusToken(t.Context(), testAgentID)
	require.NoError(t, err)
	assert.Equal(t, []byte("token"), token)

	receipt, err := c.FetchReceipt(t.Context(), testAgentID)
	require.NoError(t, err)
	assert.Equal(t, []byte("receipt"), receipt)

	assert.Equal(t, []string{testAgentID, testAgentID}, inner.agentIDs)
	assert.Equal(t, 3, inner.callCount())
}

// TestBreakerClientObservesFailures checks which fetch outcomes reach the
// breaker as strikes and that the circuit opens with a WARN once they add up.
func TestBreakerClientObservesFailures(t *testing.T) {
	connErr := &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}
	httpErr := &scitt.TransportError{Type: scitt.TransportErrHTTPError, StatusCode: http.StatusServiceUnavailable}

	tests := []struct {
		name     string
		inner    *fakeLogClient
		ctx      func(t *testing.T) context.Context
		wantErr  error
		wantOpen bool
	}{
		{
			name:     "connection failures open the circuit",
			inner:    &fakeLogClient{tokenErr: connErr},
			ctx:      liveContext,
			wantErr:  connErr,
			wantOpen: true,
		},
		{
			name:     "a hanging log takes a strike per fetch",
			inner:    &fakeLogClient{block: true},
			ctx:      liveContext,
			wantErr:  context.DeadlineExceeded,
			wantOpen: true,
		},
		{
			name:    "http errors never open the circuit",
			inner:   &fakeLogClient{tokenErr: httpErr},
			ctx:     liveContext,
			wantErr: httpErr,
		},
		{
			name:    "the caller's cancellation never counts",
			inner:   &fakeLogClient{block: true},
			ctx:     canceledContext,
			wantErr: context.Canceled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)
			b := newBreaker()
			c := newBreakerClient(tt.inner, b, 20*time.Millisecond)
			ctx := tt.ctx(t)

			for range breakerThreshold {
				_, err := c.FetchStatusToken(ctx, testAgentID)
				require.ErrorIs(t, err, tt.wantErr)
			}

			until, open := b.openUntil(testLogHost, testNow)
			assert.Equal(t, tt.wantOpen, open)
			assert.Equal(t, tt.wantOpen, strings.Contains(logs.String(), "Transparency log circuit opened"))
			assert.Empty(t, b.failures, "a closed circuit after non-strikes, or an open one, keeps no count")

			if tt.wantOpen {
				assert.Equal(t, testNow.Add(breakerCooldown), until)
				assert.Contains(t, logs.String(), "level=WARN")
			}

			assert.Equal(t, breakerThreshold, tt.inner.callCount())
		})
	}
}

func TestBreakerClientSuccessResetsStrikes(t *testing.T) {
	connErr := &scitt.TransportError{Type: scitt.TransportErrHTTPError, Message: "request failed", Cause: errBoom}
	inner := &fakeLogClient{tokenErr: connErr, receipt: []byte("receipt")}
	b := newBreaker()
	c := newBreakerClient(inner, b, time.Second)

	for range breakerThreshold - 1 {
		_, err := c.FetchStatusToken(t.Context(), testAgentID)
		require.ErrorIs(t, err, connErr)
	}

	assert.Equal(t, breakerThreshold-1, b.failures[testLogHost])

	_, err := c.FetchReceipt(t.Context(), testAgentID)
	require.NoError(t, err)
	assert.Empty(t, b.failures)

	_, open := b.openUntil(testLogHost, testNow)
	assert.False(t, open)
}
