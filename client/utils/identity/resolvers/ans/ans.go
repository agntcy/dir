// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ansresolver resolves "ans://" subjects. The key is not published
// anywhere: it is the public key of the identity certificate the claim
// carries, trusted once the agent's transparency log, found through a DNS
// record and checked against an allow-list, attests the certificate's
// fingerprint for the agent the subject names.
//
// An error Resolve returns is marked with resolvers.Final when it is a
// verdict about the claim: the subject, the certificate, what the badge
// record says, and what the transparency log states or refuses to state. A
// failure to reach the publisher's DNS or the trusted log, the caller's own
// deadline included, is left unmarked for the caller to classify by its
// cause, as for every other scheme.
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

// maxResponseBytes bounds a transparency log answer: a status token is a few
// hundred bytes and a root-keys set a few lines.
const maxResponseBytes = 64 << 10

// Resolver resolves "ans://" subjects. It is safe for concurrent use, and
// concurrent claims for one subject share one lookup; see attest.
type Resolver struct {
	cfg       Config
	trusted   map[string]struct{}
	pinned    []PinnedKey
	lookupTXT func(ctx context.Context, name string) ([]string, error)
	fetch     resolvers.Fetcher
	log       TrustedLogClient
	clock     func() time.Time

	// mu guards memo, inflight and sweepAt, and is never held across a
	// network call.
	mu       sync.Mutex
	memo     map[string]attestation
	inflight map[string]*lookup
	sweepAt  time.Time
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

// lookup is an attestation in progress. The claims of a subject that arrive
// while one runs wait for its result instead of starting their own, each
// under its own context. kept says whether the result was remembered, which
// it is not when the leading caller had given up before it came.
type lookup struct {
	done chan struct{}
	att  attestation
	kept bool
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
		inflight:  make(map[string]*lookup),
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
// one subject cost one DNS lookup and one log fetch between them, whether
// they arrive in turn or at once, and a log that is down is not asked again
// for the same subject within the lifetime. An answer obtained after the
// caller gave up is returned but not kept, since it says nothing about the
// subject.
func (r *Resolver) attest(ctx context.Context, name agentName) attestation {
	subject := name.String()

	for {
		att, call, leads := r.join(subject)
		if call == nil {
			return att
		}

		if leads {
			att = r.attestUncached(ctx, name)
			r.finish(subject, call, att, ctx.Err() == nil)

			return att
		}

		select {
		case <-call.done:
			if call.kept {
				return call.att
			}
		case <-ctx.Done():
		}

		// The lookup ended without an answer to keep, or this caller's own
		// context did: a caller still waiting runs its own lookup, one that
		// gave up reports its own cause.
		if err := ctx.Err(); err != nil {
			return attestation{err: fmt.Errorf("ans: waiting for the lookup of %s: %w", subject, err)}
		}
	}
}

// join returns the kept attestation of subject, or the lookup in progress to
// wait for, or a new lookup the caller leads.
func (r *Resolver) join(subject string) (attestation, *lookup, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if att, ok := r.recallLocked(subject); ok {
		return att, nil, false
	}

	if call, ok := r.inflight[subject]; ok {
		return attestation{}, call, false
	}

	call := &lookup{done: make(chan struct{})}
	r.inflight[subject] = call

	return attestation{}, call, true
}

// finish hands the result of a lookup to its waiters and, when keep says the
// leading caller was still waiting for it, remembers it.
func (r *Resolver) finish(subject string, call *lookup, att attestation, keep bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.inflight, subject)

	if keep {
		r.rememberLocked(subject, att)
	}

	call.att, call.kept = att, keep

	close(call.done)
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

// recallLocked returns the kept attestation of subject while it is fresh and,
// for a statement, while the statement itself has not expired. One that has
// is dropped. The caller holds mu.
func (r *Resolver) recallLocked(subject string) (attestation, bool) {
	att, ok := r.memo[subject]
	if !ok {
		return attestation{}, false
	}

	now := r.clock()
	if !now.Before(att.expires) || (att.status != nil && now.After(att.status.ExpiresAt.Add(r.cfg.GetClockSkew()))) {
		delete(r.memo, subject)

		return attestation{}, false
	}

	return att, true
}

// rememberLocked keeps att for subject for the lifetime and, once per
// lifetime, drops every entry whose lifetime has passed, so the memo stays
// bounded without a scan on every claim. The caller holds mu.
func (r *Resolver) rememberLocked(subject string, att attestation) {
	now := r.clock()
	lifetime := r.cfg.GetStatusCacheTTL()
	att.expires = now.Add(lifetime)

	if !now.Before(r.sweepAt) {
		for key, entry := range r.memo {
			if !now.Before(entry.expires) {
				delete(r.memo, key)
			}
		}

		r.sweepAt = now.Add(lifetime)
	}

	r.memo[subject] = att
}
