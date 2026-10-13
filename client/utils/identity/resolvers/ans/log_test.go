// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tokenURL    = testBadgeURL + "/status-token"
	rootKeysURL = testLogOrigin + "/root-keys"
)

// logFixture is a synthetic log served by a fake fetcher, and a client that
// trusts it, pinned or not.
type logFixture struct {
	t       *testing.T
	log     *testLog
	cert    identityCert
	fetcher *fakeFetcher
	clock   *fakeClock
	client  *scittLogClient
	target  TrustedLog
}

func newLogFixture(t *testing.T, pinned bool) *logFixture {
	t.Helper()

	f := &logFixture{
		t:       t,
		log:     mintLog(t),
		cert:    mintTestCert(t),
		fetcher: newFakeFetcher(),
		clock:   newFakeClock(testNow),
		target:  TrustedLog{Origin: testLogOrigin, AgentID: testAgentID},
	}

	f.fetcher.serve(rootKeysURL, []byte(f.log.rootKeyLine(t)+"\n"))
	f.serveToken(f.log.claims(f.cert))

	var store *scitt.KeyStore

	if pinned {
		var err error

		store, err = scitt.NewKeyStore([]string{f.log.rootKeyLine(t)})
		require.NoError(t, err)
	}

	f.client = newScittLogClient(f.fetcher, store, DefaultRootKeysTTL, DefaultClockSkew, f.clock.Now)

	return f
}

func (f *logFixture) serveToken(claims tokenClaims) {
	f.fetcher.serve(tokenURL, f.log.statusToken(f.t, claims))
}

func (f *logFixture) status() (*Status, error) {
	return f.client.Status(f.t.Context(), f.target)
}

func TestLogClientStatusMapsTheToken(t *testing.T) {
	for _, pinned := range []bool{true, false} {
		t.Run(map[bool]string{true: "pinned", false: "unpinned"}[pinned], func(t *testing.T) {
			f := newLogFixture(t, pinned)

			claims := f.log.claims(f.cert, mintTestCert(t))
			claims.status = string(StateDeprecated)
			f.serveToken(claims)

			status, err := f.status()
			require.NoError(t, err)

			assert.Equal(t, testAgentID, status.AgentID)
			assert.Equal(t, testAnsName, status.Name)
			assert.Equal(t, StateDeprecated, status.State)
			assert.Equal(t, time.Unix(claims.exp, 0), status.ExpiresAt)
			assert.Len(t, status.IdentityCertificates, 2)
			assert.True(t, status.attests(fingerprintOf(f.cert.der)))
			assert.False(t, status.attests([32]byte{}))

			wantRootKeyFetches := 0
			if !pinned {
				wantRootKeyFetches = 1
			}

			assert.Equal(t, wantRootKeyFetches, f.fetcher.count(rootKeysURL))
			assert.Equal(t, 1, f.fetcher.count(tokenURL))
		})
	}
}

func TestLogClientStatusFailures(t *testing.T) {
	tests := []struct {
		name         string
		pinned       bool
		setup        func(f *logFixture)
		wantErr      string
		wantStatus   int
		wantFinal    bool
		wantRootKeys int
	}{
		{
			name:   "status token 503 is left to the caller",
			pinned: true,
			setup: func(f *logFixture) {
				f.fetcher.fail(tokenURL, &safefetch.StatusError{URL: tokenURL, Code: http.StatusServiceUnavailable})
			},
			wantErr:    "ans log: fetch status token: ",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:   "status token 429 is left to the caller",
			pinned: true,
			setup: func(f *logFixture) {
				f.fetcher.fail(tokenURL, &safefetch.StatusError{URL: tokenURL, Code: http.StatusTooManyRequests})
			},
			wantErr:    "ans log: fetch status token: ",
			wantStatus: http.StatusTooManyRequests,
		},
		{
			name:   "status token 301 is left to the caller",
			pinned: true,
			setup: func(f *logFixture) {
				f.fetcher.fail(tokenURL, &safefetch.StatusError{URL: tokenURL, Code: http.StatusMovedPermanently})
			},
			wantErr:    "ans log: fetch status token: ",
			wantStatus: http.StatusMovedPermanently,
		},
		{
			name:   "status token 404 is a verdict",
			pinned: true,
			setup: func(f *logFixture) {
				f.fetcher.fail(tokenURL, &safefetch.StatusError{URL: tokenURL, Code: http.StatusNotFound})
			},
			wantErr:    "ans log: fetch status token: ",
			wantStatus: http.StatusNotFound,
			wantFinal:  true,
		},
		{
			name:   "status token 403 is a verdict",
			pinned: true,
			setup: func(f *logFixture) {
				f.fetcher.fail(tokenURL, &safefetch.StatusError{URL: tokenURL, Code: http.StatusForbidden})
			},
			wantErr:    "ans log: fetch status token: ",
			wantStatus: http.StatusForbidden,
			wantFinal:  true,
		},
		{
			name: "root keys 500 is left to the caller",
			setup: func(f *logFixture) {
				f.fetcher.fail(rootKeysURL, &safefetch.StatusError{URL: rootKeysURL, Code: http.StatusInternalServerError})
			},
			wantErr:      "ans log: fetch root keys: ",
			wantStatus:   http.StatusInternalServerError,
			wantRootKeys: 1,
		},
		{
			name: "root keys 404 is a verdict",
			setup: func(f *logFixture) {
				f.fetcher.fail(rootKeysURL, &safefetch.StatusError{URL: rootKeysURL, Code: http.StatusNotFound})
			},
			wantErr:      "ans log: fetch root keys: ",
			wantStatus:   http.StatusNotFound,
			wantFinal:    true,
			wantRootKeys: 1,
		},
		{
			name:         "empty root keys",
			setup:        func(f *logFixture) { f.fetcher.serve(rootKeysURL, []byte("\n  \n")) },
			wantErr:      "ans log: transparency log served no root keys",
			wantFinal:    true,
			wantRootKeys: 1,
		},
		{
			name:         "malformed root keys",
			setup:        func(f *logFixture) { f.fetcher.serve(rootKeysURL, []byte("not-a-root-key\n")) },
			wantErr:      "ans log: transparency log served malformed root keys",
			wantFinal:    true,
			wantRootKeys: 1,
		},
		{
			name:      "pinned keys reject a token signed by another log",
			pinned:    true,
			setup:     func(f *logFixture) { f.fetcher.serve(tokenURL, mintLog(f.t).statusToken(f.t, f.log.claims(f.cert))) },
			wantErr:   "ans log: status token signed by a key that is not in root_keys: key id ",
			wantFinal: true,
		},
		{
			name:   "expired token",
			pinned: true,
			setup: func(f *logFixture) {
				claims := f.log.claims(f.cert)
				claims.exp = testNow.Unix() - 120
				f.serveToken(claims)
			},
			wantErr:   "ans log: status token did not verify: token expired",
			wantFinal: true,
		},
		{
			name:   "revoked agent",
			pinned: true,
			setup: func(f *logFixture) {
				claims := f.log.claims(f.cert)
				claims.status = "REVOKED"
				f.serveToken(claims)
			},
			wantErr:   "ans log: status token did not verify: terminal status [REVOKED]",
			wantFinal: true,
		},
		{
			name:      "token that is not COSE",
			pinned:    true,
			setup:     func(f *logFixture) { f.fetcher.serve(tokenURL, []byte("nope")) },
			wantErr:   "ans log: status token did not verify: ",
			wantFinal: true,
		},
		{
			name:   "forged signature under the trusted key id",
			pinned: true,
			setup: func(f *logFixture) {
				token := f.log.statusToken(f.t, f.log.claims(f.cert))
				token[len(token)-1] ^= 0x01
				f.fetcher.serve(tokenURL, token)
			},
			wantErr:   "ans log: status token did not verify: ",
			wantFinal: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLogFixture(t, tt.pinned)
			tt.setup(f)

			status, err := f.status()
			require.ErrorContains(t, err, tt.wantErr)
			assert.Nil(t, status)
			assert.Equal(t, tt.wantRootKeys, f.fetcher.count(rootKeysURL))

			var statusErr *safefetch.StatusError

			if tt.wantStatus != 0 {
				// The cause of a fetch failure stays readable, verdict or not.
				require.ErrorAs(t, err, &statusErr)
				assert.Equal(t, tt.wantStatus, statusErr.Code)
			} else {
				assert.NotErrorAs(t, err, &statusErr, "a verdict about the token or the keys must not carry a fetch error")
			}

			if tt.wantFinal {
				require.ErrorIs(t, err, resolvers.ErrFinal)
			} else {
				require.NotErrorIs(t, err, resolvers.ErrFinal)
			}
		})
	}
}

// An answer about the resource is a verdict; no answer, a server-side failure,
// rate limiting or a redirect is left for the caller.
func TestFetchError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantFinal bool
	}{
		{name: "connection failure", err: errors.New("dial tcp: connection refused")},
		{name: "deadline", err: context.DeadlineExceeded},
		{name: "500", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusInternalServerError}},
		{name: "503", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusServiceUnavailable}},
		{name: "408", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusRequestTimeout}},
		{name: "429", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusTooManyRequests}},
		{name: "301", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusMovedPermanently}},
		{name: "307", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusTemporaryRedirect}},
		{name: "204", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusNoContent}, wantFinal: true},
		{name: "400", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusBadRequest}, wantFinal: true},
		{name: "401", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusUnauthorized}, wantFinal: true},
		{name: "403", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusForbidden}, wantFinal: true},
		{name: "404", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusNotFound}, wantFinal: true},
		{name: "410", err: &safefetch.StatusError{URL: tokenURL, Code: http.StatusGone}, wantFinal: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fetchError("status token", tt.err)
			require.ErrorIs(t, err, tt.err)
			assert.Equal(t, "ans log: fetch status token: "+tt.err.Error(), err.Error())
			assert.Equal(t, tt.wantFinal, errors.Is(err, resolvers.ErrFinal))
		})
	}
}

// root_keys may hold several keys, so a log's next key can be pinned beside
// its current one and a rotation has no gap.
func TestLogClientAcceptsAnyPinnedKey(t *testing.T) {
	f := newLogFixture(t, true)
	next := mintLog(t)

	store, err := scitt.NewKeyStore([]string{f.log.rootKeyLine(t), next.rootKeyLine(t)})
	require.NoError(t, err)

	f.client = newScittLogClient(f.fetcher, store, DefaultRootKeysTTL, DefaultClockSkew, f.clock.Now)

	_, err = f.status()
	require.NoError(t, err, "the current key")

	f.fetcher.serve(tokenURL, next.statusToken(t, f.log.claims(f.cert)))

	status, err := f.status()
	require.NoError(t, err, "the next key")
	assert.Equal(t, testAgentID, status.AgentID)
}

// A key that is not pinned is named by a cause a caller can test for, through
// the verdict mark.
func TestLogClientNamesTheUnpinnedKey(t *testing.T) {
	f := newLogFixture(t, true)
	f.fetcher.serve(tokenURL, mintLog(t).statusToken(t, f.log.claims(f.cert)))

	_, err := f.status()
	require.ErrorIs(t, err, ErrUnpinnedRootKey)
	require.ErrorIs(t, err, resolvers.ErrFinal)
}

func TestLogClientCachesFetchedRootKeys(t *testing.T) {
	f := newLogFixture(t, false)

	for range 3 {
		_, err := f.status()
		require.NoError(t, err)
	}

	assert.Equal(t, 1, f.fetcher.count(rootKeysURL), "cached within the ttl")

	f.clock.advance(DefaultRootKeysTTL - time.Second)

	_, err := f.status()
	require.NoError(t, err)
	assert.Equal(t, 1, f.fetcher.count(rootKeysURL), "still cached just before the ttl")

	f.clock.advance(time.Second)

	_, err = f.status()
	require.NoError(t, err)
	assert.Equal(t, 2, f.fetcher.count(rootKeysURL), "fetched again once the ttl passed")
}

func TestLogClientFollowsAKeyRotationOnce(t *testing.T) {
	f := newLogFixture(t, false)

	_, err := f.status()
	require.NoError(t, err)

	// The log rotates its key: the token is signed by a key the cached set
	// does not hold, and /root-keys now serves the new key.
	rotated := mintLog(t)
	f.fetcher.serve(rootKeysURL, []byte(rotated.rootKeyLine(t)+"\n"))
	f.fetcher.serve(tokenURL, rotated.statusToken(t, f.log.claims(f.cert)))

	status, err := f.status()
	require.NoError(t, err)
	assert.Equal(t, testAgentID, status.AgentID)
	assert.Equal(t, 2, f.fetcher.count(rootKeysURL), "one forced refresh")

	// A token signed by yet another key the log does not publish is refused
	// without another fetch until the forced set expires.
	stranger := mintLog(t)
	f.fetcher.serve(tokenURL, stranger.statusToken(t, f.log.claims(f.cert)))

	for range 2 {
		_, err = f.status()
		require.ErrorContains(t, err, "which the log's root keys do not list")
	}

	assert.Equal(t, 2, f.fetcher.count(rootKeysURL), "a forced set is not refreshed again")

	f.clock.advance(DefaultRootKeysTTL)

	_, err = f.status()
	require.ErrorContains(t, err, "which the log's root keys do not list")
	assert.Equal(t, 4, f.fetcher.count(rootKeysURL), "expired set is fetched, then refreshed once more")
}

func TestLogClientForcedRefreshKeepsFetchErrors(t *testing.T) {
	f := newLogFixture(t, false)

	_, err := f.status()
	require.NoError(t, err)

	f.fetcher.serve(tokenURL, mintLog(t).statusToken(t, f.log.claims(f.cert)))
	f.fetcher.fail(rootKeysURL, &safefetch.StatusError{URL: rootKeysURL, Code: http.StatusBadGateway})

	_, err = f.status()
	require.ErrorContains(t, err, "ans log: fetch root keys: ")

	var statusErr *safefetch.StatusError

	require.ErrorAs(t, err, &statusErr)
	assert.Equal(t, http.StatusBadGateway, statusErr.Code)
}

func TestStateAllowsUse(t *testing.T) {
	tests := []struct {
		state State
		want  bool
	}{
		{StateActive, true},
		{StateWarning, true},
		{StateDeprecated, true},
		{State("REVOKED"), false},
		{State("EXPIRED"), false},
		{State("PENDING_DNS"), false},
		{State("active"), false},
		{State(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.allowsUse())
		})
	}
}
