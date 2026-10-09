// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"maps"
	"net"
	"time"

	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	ansresolver "github.com/agntcy/dir/client/utils/identity/resolvers/ans"
	didresolver "github.com/agntcy/dir/client/utils/identity/resolvers/did"
	dnsresolver "github.com/agntcy/dir/client/utils/identity/resolvers/dns"
	wellknownresolver "github.com/agntcy/dir/client/utils/identity/resolvers/wellknown"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/opencontainers/go-digest"
)

// schemeResolver is what the task knows about one subject scheme: the resolver
// that finds its keys, and how long a stored result survives lookups that get
// no answer.
type schemeResolver struct {
	resolver resolvers.Resolver
	grace    time.Duration
}

// resolverSet holds one entry per subject scheme the task serves; the schemes
// themselves are the client package's (clientidentity.SchemeOf).
type resolverSet map[clientidentity.Scheme]schemeResolver

// newNetworkResolvers returns the resolvers that need no per-run state. A nil
// fetcher makes the DID and well-known resolvers use an SSRF-guarded client.
// The ans resolver exists only when its block is enabled, and an invalid block
// is an error, so a mistake fails startup rather than every ans:// claim.
func newNetworkResolvers(cfg Config) (resolverSet, error) {
	set := resolverSet{
		clientidentity.SchemeDNS:       {resolver: dnsresolver.New(), grace: staleGrace},
		clientidentity.SchemeDID:       {resolver: didresolver.New(nil), grace: staleGrace},
		clientidentity.SchemeWellKnown: {resolver: wellknownresolver.New(nil), grace: staleGrace},
	}

	if !cfg.ANS.Enabled {
		return set, nil
	}

	ans, err := ansresolver.New(cfg.ANS.Config)
	if err != nil {
		return resolverSet{}, fmt.Errorf("configure the ans resolver: %w", err)
	}

	pinned := ans.PinnedKeys()

	keys := make([]string, 0, len(pinned))
	for _, key := range pinned {
		keys = append(keys, key.Name+":"+key.KeyID)
	}

	logger.Info("ans:// claims are verified against the Agent Name Service",
		"trustedLogHosts", cfg.ANS.TrustedLogHosts,
		"pinnedRootKeys", keys,
		"timeout", cfg.ANS.GetTimeout(),
		"statusCacheTTL", cfg.ANS.GetStatusCacheTTL(),
		"staleGrace", cfg.ANS.GetStaleGrace())

	if cfg.ANS.AllowUnpinnedRootKeys {
		logger.Warn("ans root keys are not pinned: trust in the transparency logs rests on TLS to the trusted hosts alone")
	}

	set[clientidentity.SchemeANS] = schemeResolver{resolver: ans, grace: cfg.ANS.GetStaleGrace()}

	return set, nil
}

// with returns a copy of the set in which scheme is served by resolver.
func (s resolverSet) with(scheme clientidentity.Scheme, resolver resolvers.Resolver, grace time.Duration) resolverSet {
	out := make(resolverSet, len(s)+1)
	maps.Copy(out, s)
	out[scheme] = schemeResolver{resolver: resolver, grace: grace}

	return out
}

// cached returns the set with every resolver remembering its answers, so a
// subject shared by many records is looked up once per run. Not safe for
// concurrent use: a run is sequential.
func (s resolverSet) cached() resolverSet {
	out := make(resolverSet, len(s))

	for scheme, entry := range s {
		if entry.resolver != nil {
			entry.resolver = &cachedResolver{next: entry.resolver, seen: map[digest.Digest]lookup{}}
		}

		out[scheme] = entry
	}

	return out
}

type lookup struct {
	keys []crypto.PublicKey
	err  error
}

type cachedResolver struct {
	next resolvers.Resolver
	seen map[digest.Digest]lookup
}

// cacheKey digests the subject and certificate.
func cacheKey(subject string, certificate []byte) digest.Digest {
	b := make([]byte, 0, len(subject)+len(certificate))
	b = append(b, subject...)
	b = append(b, certificate...)

	return digest.FromBytes(b)
}

func (c *cachedResolver) Resolve(ctx context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error) {
	key := cacheKey(subject, certificate)
	if l, ok := c.seen[key]; ok {
		return l.keys, l.err
	}

	keys, err := c.next.Resolve(ctx, subject, certificate)
	c.seen[key] = lookup{keys, err}

	return keys, err //nolint:wrapcheck // the answer is passed through as it came
}

// isTransient reports whether a failed key lookup says nothing about the claim:
// a timeout, a refused or dropped connection, a DNS failure other than "no such
// name", or a 5xx, 408, 429 or 3xx response (a 3xx only reaches here from a
// client that refuses redirects, and a redirect says nothing about the claim
// either). A subject that answers and publishes no usable key is not transient,
// and neither is a failure the resolver marked as a verdict with
// resolvers.Final, whatever cause it carries.
func isTransient(err error) bool {
	var (
		statusErr *safefetch.StatusError
		dnsErr    *net.DNSError
		opErr     *net.OpError
		netErr    net.Error
	)

	switch {
	case errors.Is(err, resolvers.ErrFinal), errors.Is(err, safefetch.ErrDisallowedAddress):
		return false
	case errors.Is(err, context.DeadlineExceeded):
		return true
	case errors.As(err, &statusErr):
		return statusErr.Transient()
	case errors.As(err, &dnsErr):
		return !dnsErr.IsNotFound
	case errors.As(err, &opErr):
		return true
	case errors.As(err, &netErr):
		return netErr.Timeout()
	}

	return false
}

// graceFor returns how long a stored result for subject survives lookups that
// get no answer; a scheme the set does not serve has none.
func (s resolverSet) graceFor(subject string) time.Duration {
	scheme, _ := clientidentity.SchemeOf(subject)

	return s[scheme].grace
}

// forSubject picks the resolver for a claim's subject by its scheme. A scheme
// the set does not serve, the ans block being off for one, is unsupported.
func (s resolverSet) forSubject(subject string) (resolvers.Resolver, error) {
	scheme, ok := clientidentity.SchemeOf(subject)

	entry := s[scheme]
	if !ok || entry.resolver == nil {
		return nil, fmt.Errorf("unsupported subject scheme in %q", subject)
	}

	return entry.resolver, nil
}
