// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ansresolver resolves "ans://" subjects. The key is not published
// anywhere: it is the public key of the identity certificate the claim
// carries, trusted once the agent's transparency log, found through a DNS
// record and checked against an allow-list, attests the certificate's
// fingerprint for the agent the subject names.
//
// Every error Resolve returns is marked with resolvers.Final, a verdict about
// the claim whatever its cause, with one exception: a failure to fetch from a
// trusted log while the caller is still waiting is left for the caller to
// classify by its cause, so a log outage can be treated as transient. Failures
// the publisher controls (the subject, the certificate, the DNS record) and
// failures observed after the caller gave up are therefore never transient.
package ansresolver

import (
	"context"
	"crypto"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/utils/safefetch"
)

const (
	// maxResponseBytes bounds a transparency log answer: a status token is a
	// few hundred bytes and a root-keys set a few lines.
	maxResponseBytes = 64 << 10

	// minStatusCacheTTL keeps an attestation reusable across the claims of one
	// record even when the configured lifetime is shorter.
	minStatusCacheTTL = 5 * time.Second
)

// Resolver resolves "ans://" subjects. It is safe for concurrent use.
type Resolver struct {
	cfg       Config
	trusted   map[string]struct{}
	pinned    []PinnedKey
	lookupTXT func(ctx context.Context, name string) ([]string, error)
	fetch     resolvers.Fetcher
	log       TrustedLogClient
	clock     func() time.Time

	// mu guards memo and is never held across a network call.
	mu   sync.Mutex
	memo map[string]attestation
}

// PinnedKey identifies one configured root key, for an operator-facing log line.
type PinnedKey struct {
	Name  string
	KeyID string
}

// final marks err as a verdict about the claim; see resolvers.Final.
func final(err error) error {
	return resolvers.Final(err) //nolint:wrapcheck // the mark is the wrapping
}

// attestation is what the network stages said about one subject: the log's
// statement, or the classified error, kept until expires.
type attestation struct {
	status  *Status
	err     error
	expires time.Time
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithLookupTXT replaces the system DNS lookup, for tests.
func WithLookupTXT(lookup func(ctx context.Context, name string) ([]string, error)) Option {
	return func(r *Resolver) { r.lookupTXT = lookup }
}

// WithLogClient replaces the client that fetches and verifies status tokens,
// for tests. WithFetcher and WithClock then only affect the resolver itself.
func WithLogClient(client TrustedLogClient) Option {
	return func(r *Resolver) { r.log = client }
}

// WithFetcher replaces the HTTPS client the default log client fetches with.
func WithFetcher(fetch resolvers.Fetcher) Option {
	return func(r *Resolver) { r.fetch = fetch }
}

// WithClock replaces the clock, for tests of the time-based behaviour.
func WithClock(clock func() time.Time) Option {
	return func(r *Resolver) { r.clock = clock }
}

// New creates a Resolver. It validates cfg and parses the pinned root keys,
// so a malformed configuration fails here rather than at the first claim. No
// network call is made.
func New(cfg Config, opts ...Option) (*Resolver, error) {
	trusted, err := cfg.trustedSet()
	if err != nil {
		return nil, err
	}

	if err := cfg.validateTrust(); err != nil {
		return nil, err
	}

	store, pinned, err := parsePinnedKeys(trimmed(cfg.RootKeys))
	if err != nil {
		return nil, err
	}

	r := &Resolver{
		cfg:       cfg,
		trusted:   trusted,
		pinned:    pinned,
		lookupTXT: net.DefaultResolver.LookupTXT,
		fetch:     safefetch.New(safefetch.WithTimeout(cfg.GetTimeout()), safefetch.WithMaxBytes(maxResponseBytes), safefetch.WithoutRedirects()),
		clock:     time.Now,
		memo:      make(map[string]attestation),
	}

	for _, opt := range opts {
		opt(r)
	}

	if r.log == nil {
		r.log = newScittLogClient(r.fetch, store, cfg.GetRootKeysTTL(), cfg.GetClockSkew(), func() time.Time { return r.clock() })
	}

	return r, nil
}

// parsePinnedKeys parses the configured root-key lines. A nil store means
// none are pinned.
func parsePinnedKeys(lines []string) (*scitt.KeyStore, []PinnedKey, error) {
	if len(lines) == 0 {
		return nil, nil, nil
	}

	pinned := make([]PinnedKey, 0, len(lines))

	for _, line := range lines {
		key, err := scitt.ParseC2SPKey(line)
		if err != nil {
			return nil, nil, fmt.Errorf("ans: root_keys: %w", err)
		}

		pinned = append(pinned, PinnedKey{Name: key.Name, KeyID: hex.EncodeToString(key.Kid[:])})
	}

	store, err := scitt.NewKeyStore(lines)
	if err != nil {
		return nil, nil, fmt.Errorf("ans: root_keys: %w", err)
	}

	return store, pinned, nil
}

// PinnedKeys returns the configured root keys by name and key id; none in
// unpinned mode.
func (r *Resolver) PinnedKeys() []PinnedKey {
	return slices.Clone(r.pinned)
}

// Resolve implements resolvers.Resolver. certificate is the claim's
// DER-encoded identity certificate, which must name exactly subject as a URI
// SAN and be valid now; its public key is the result once the agent's
// transparency log attests the certificate's fingerprint for an agent whose
// name is subject and whose status allows use. The checks that need no
// network run first, so a claim that cannot verify costs no lookup.
func (r *Resolver) Resolve(ctx context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error) {
	name, err := parseAgentName(subject)
	if err != nil {
		return nil, final(err)
	}

	if len(certificate) == 0 {
		return nil, final(errors.New("ans certificate: claim carries no certificate"))
	}

	cert, err := x509.ParseCertificate(certificate)
	if err != nil {
		return nil, final(fmt.Errorf("ans certificate: parse certificate: %w", err))
	}

	key, err := checkCertificate(cert, subject, r.clock())
	if err != nil {
		return nil, final(err)
	}

	att := r.attest(ctx, name)
	if att.err != nil {
		return nil, att.err
	}

	if !att.status.attests(sha256.Sum256(certificate)) {
		return nil, final(errors.New("ans log: the certificate is not among the agent's valid identity certificates"))
	}

	return []crypto.PublicKey{key}, nil
}

// attest returns what the network stages say about name, reusing a recent
// answer, a failure included: anyone can attach claims to a record, and a
// claim's certificate only matters at the fingerprint check, so the claims of
// one subject cost one DNS lookup and one log fetch between them, and a log
// that is down is not asked again for the same subject within the lifetime.
// An answer obtained after the caller gave up is returned but not kept, since
// it says nothing about the subject.
func (r *Resolver) attest(ctx context.Context, name agentName) attestation {
	subject := name.String()

	if att, ok := r.recall(subject); ok {
		return att
	}

	att := r.attestUncached(ctx, name)

	if ctx.Err() == nil {
		r.remember(subject, att)
	}

	return att
}

// attestUncached runs the two network stages, each under its own budget
// derived from the caller's context: slow DNS cannot shorten the log's
// window, and a DNS hang ends after one timeout.
func (r *Resolver) attestUncached(ctx context.Context, name agentName) attestation {
	dnsCtx, cancelDNS := context.WithTimeout(ctx, r.cfg.GetTimeout())
	target, err := r.lookupBadge(dnsCtx, name)

	cancelDNS()

	if err != nil {
		return attestation{err: err}
	}

	logCtx, cancelLog := context.WithTimeout(ctx, r.cfg.GetTimeout())
	status, err := r.log.Status(logCtx, target)

	cancelLog()

	if err != nil {
		// Once the caller has given up, the failure is the caller's, not the log's.
		if ctx.Err() != nil {
			err = final(err)
		}

		return attestation{err: err}
	}

	if !strings.EqualFold(status.AgentID, target.AgentID) {
		return attestation{err: final(fmt.Errorf("ans log: status token names agent %q, expected %q", truncate(status.AgentID), target.AgentID))}
	}

	if !name.matches(status.Name) {
		return attestation{err: final(fmt.Errorf("ans log: status token names %q, expected %q", truncate(status.Name), name.String()))}
	}

	if !status.State.allowsUse() {
		return attestation{err: final(fmt.Errorf("ans log: agent status %q does not allow use", truncate(string(status.State))))}
	}

	return attestation{status: status}
}

// recall returns the kept attestation of subject while it is fresh and, for
// a statement, while the statement itself has not expired.
func (r *Resolver) recall(subject string) (attestation, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	att, ok := r.memo[subject]
	if !ok {
		return attestation{}, false
	}

	now := r.clock()
	if !now.Before(att.expires) {
		return attestation{}, false
	}

	if att.status != nil && !att.status.ExpiresAt.IsZero() && now.After(att.status.ExpiresAt.Add(r.cfg.GetClockSkew())) {
		return attestation{}, false
	}

	return att, true
}

// remember keeps att for subject and drops every expired entry on the way.
func (r *Resolver) remember(subject string, att attestation) {
	now := r.clock()
	att.expires = now.Add(max(r.cfg.GetStatusCacheTTL(), minStatusCacheTTL))

	r.mu.Lock()
	defer r.mu.Unlock()

	for key, entry := range r.memo {
		if !now.Before(entry.expires) {
			delete(r.memo, key)
		}
	}

	r.memo[subject] = att
}
