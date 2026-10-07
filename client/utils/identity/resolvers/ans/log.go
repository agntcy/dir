// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/utils/safefetch"
)

// State is an agent's lifecycle status as its transparency log reports it.
type State string

// The states in which an agent may be relied on.
const (
	StateActive     State = "ACTIVE"
	StateWarning    State = "WARNING"
	StateDeprecated State = "DEPRECATED"
)

// allowsUse reports whether an agent in this state may be relied on. Every
// other state, known or not, fails closed.
func (s State) allowsUse() bool {
	switch s {
	case StateActive, StateWarning, StateDeprecated:
		return true
	default:
		return false
	}
}

// Status is what a transparency log states about an agent in a status token
// it signed.
type Status struct {
	// AgentID is the agent's id on the log.
	AgentID string

	// Name is the agent's ANS name as the log spells it.
	Name string

	// State is the agent's lifecycle status.
	State State

	// ExpiresAt is when the statement stops being valid. The log's tokens
	// always carry one, so a zero value counts as already expired.
	ExpiresAt time.Time

	// IdentityCertificates are the SHA-256 fingerprints of the agent's valid
	// identity certificates.
	IdentityCertificates [][32]byte
}

// attests reports whether fingerprint is one of the agent's valid identity
// certificates.
func (s *Status) attests(fingerprint [32]byte) bool {
	for _, known := range s.IdentityCertificates {
		if subtle.ConstantTimeCompare(known[:], fingerprint[:]) == 1 {
			return true
		}
	}

	return false
}

// TrustedLogClient fetches what a trusted transparency log states about an
// agent and verifies that the log signed it. The default client talks to the
// log over HTTPS and verifies the status token with the configured keys; a
// test scripts it.
type TrustedLogClient interface {
	Status(ctx context.Context, log TrustedLog) (*Status, error)
}

// scittLogClient is the default TrustedLogClient: it fetches the agent's
// COSE_Sign1 status token and verifies it against the pinned root keys, or
// against the keys the log publishes when none are pinned.
//
// Fetch errors are returned as they are, so the caller can tell a log outage
// from a verdict by their cause; a verdict about the token or the keys is
// marked resolvers.Final. scitt.RefreshableKeyStore is not reused: it fetches
// with the SDK's own HTTP client rather than the SSRF-guarded one here, and
// its refresh only merges keys, so a rotated-out key would stay trusted.
type scittLogClient struct {
	fetch  resolvers.Fetcher
	pinned *scitt.KeyStore
	ttl    time.Duration
	skew   time.Duration
	clock  func() time.Time

	// mu guards cache and is never held across a fetch.
	mu    sync.Mutex
	cache map[string]cachedKeys
}

// cachedKeys is one log origin's fetched key set. forced marks a set fetched
// because a token named a key the previous set did not hold; such a set is
// kept until it expires, so a log signing with a key it does not publish
// costs one extra fetch rather than one per claim.
type cachedKeys struct {
	keys    *scitt.KeyStore
	expires time.Time
	forced  bool
}

// newScittLogClient returns a client fetching through fetch and trusting the
// pinned keys, or the keys the log publishes when pinned is nil.
func newScittLogClient(fetch resolvers.Fetcher, pinned *scitt.KeyStore, ttl, skew time.Duration, clock func() time.Time) *scittLogClient {
	return &scittLogClient{
		fetch:  fetch,
		pinned: pinned,
		ttl:    ttl,
		skew:   skew,
		clock:  clock,
		cache:  make(map[string]cachedKeys),
	}
}

// Status implements TrustedLogClient.
func (c *scittLogClient) Status(ctx context.Context, log TrustedLog) (*Status, error) {
	body, err := c.fetch.Get(ctx, log.Origin+"/v1/agents/"+log.AgentID+"/status-token")
	if err != nil {
		return nil, fetchError("status token", err)
	}

	keys, err := c.keysFor(ctx, log.Origin)
	if err != nil {
		return nil, err
	}

	token, err := scitt.VerifyStatusTokenAt(body, keys, c.skew, c.clock().Unix())

	if kid, unknown := unknownKeyID(err); unknown {
		if c.pinned != nil {
			return nil, final(fmt.Errorf("ans log: status token signed by key id %x, which is not in root_keys: %w", kid, err))
		}

		fresh, refreshed, refreshErr := c.refreshKeys(ctx, log.Origin)
		if refreshErr != nil {
			return nil, refreshErr
		}

		if refreshed {
			token, err = scitt.VerifyStatusTokenAt(body, fresh, c.skew, c.clock().Unix())
		}

		if _, stillUnknown := unknownKeyID(err); stillUnknown {
			return nil, final(fmt.Errorf("ans log: status token signed by key id %x, which the log's root keys do not list: %w", kid, err))
		}
	}

	if err != nil {
		return nil, final(fmt.Errorf("ans log: status token did not verify: %w", err))
	}

	return statusFrom(token), nil
}

// keysFor returns the keys that authenticate the log at origin: the pinned
// set, or the set fetched from the log and cached for ttl.
func (c *scittLogClient) keysFor(ctx context.Context, origin string) (*scitt.KeyStore, error) {
	if c.pinned != nil {
		return c.pinned, nil
	}

	c.mu.Lock()
	entry, ok := c.cache[origin]
	c.mu.Unlock()

	if ok && c.clock().Before(entry.expires) {
		return entry.keys, nil
	}

	keys, err := c.fetchKeys(ctx, origin)
	if err != nil {
		return nil, err
	}

	c.store(origin, keys, false)

	return keys, nil
}

// refreshKeys fetches the log's keys again after a token named a key the
// cached set did not hold. It reports false, and fetches nothing, when the
// cached set was itself the product of such a refresh.
func (c *scittLogClient) refreshKeys(ctx context.Context, origin string) (*scitt.KeyStore, bool, error) {
	c.mu.Lock()
	entry, ok := c.cache[origin]
	c.mu.Unlock()

	if ok && entry.forced && c.clock().Before(entry.expires) {
		return nil, false, nil
	}

	keys, err := c.fetchKeys(ctx, origin)
	if err != nil {
		return nil, false, err
	}

	c.store(origin, keys, true)

	return keys, true, nil
}

// fetchKeys fetches and parses the log's /root-keys lines.
func (c *scittLogClient) fetchKeys(ctx context.Context, origin string) (*scitt.KeyStore, error) {
	body, err := c.fetch.Get(ctx, origin+"/root-keys")
	if err != nil {
		return nil, fetchError("root keys", err)
	}

	lines := trimmed(strings.Split(string(body), "\n"))
	if len(lines) == 0 {
		return nil, final(errors.New("ans log: transparency log served no root keys"))
	}

	keys, err := scitt.NewKeyStore(lines)
	if err != nil {
		return nil, final(fmt.Errorf("ans log: transparency log served malformed root keys: %w", err))
	}

	return keys, nil
}

func (c *scittLogClient) store(origin string, keys *scitt.KeyStore, forced bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[origin] = cachedKeys{keys: keys, expires: c.clock().Add(c.ttl), forced: forced}
}

// fetchError wraps a failure to fetch what from a trusted log. An answer that
// refuses the request or says the resource is not there is a verdict. An
// answer that asks for another try later, a redirect the client does not
// follow, or no answer at all is left for the caller to classify by its
// cause, as for every other scheme.
func fetchError(what string, err error) error {
	err = fmt.Errorf("ans log: fetch %s: %w", what, err)

	var statusErr *safefetch.StatusError
	if errors.As(err, &statusErr) && !retryLater(statusErr.Code) {
		return final(err)
	}

	return err
}

// retryLater reports whether an HTTP status says nothing about the resource
// itself: a server-side failure, a timeout, rate limiting, or a redirect.
func retryLater(code int) bool {
	return code >= http.StatusInternalServerError ||
		code == http.StatusRequestTimeout ||
		code == http.StatusTooManyRequests ||
		(code >= http.StatusMultipleChoices && code < http.StatusBadRequest)
}

// unknownKeyID reports whether err says the token was signed by a key the key
// set does not hold, and which one.
func unknownKeyID(err error) ([4]byte, bool) {
	var sigErr *scitt.SignatureError
	if !errors.As(err, &sigErr) || sigErr.Type != scitt.SigErrUnknownKeyID {
		return [4]byte{}, false
	}

	return sigErr.Kid, true
}

// statusFrom maps a verified token's payload to a Status.
func statusFrom(token *scitt.VerifiedStatusToken) *Status {
	payload := token.Payload

	certificates := make([][32]byte, 0, len(payload.ValidIdentityCerts))
	for _, entry := range payload.ValidIdentityCerts {
		certificates = append(certificates, entry.Fingerprint)
	}

	return &Status{
		AgentID:              payload.AgentID,
		Name:                 payload.AnsName,
		State:                State(payload.Status),
		ExpiresAt:            time.Unix(payload.Exp, 0),
		IdentityCertificates: certificates,
	}
}
