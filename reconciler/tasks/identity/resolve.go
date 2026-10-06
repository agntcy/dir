// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	didresolver "github.com/agntcy/dir/client/utils/identity/resolvers/did"
	dnsresolver "github.com/agntcy/dir/client/utils/identity/resolvers/dns"
	wellknownresolver "github.com/agntcy/dir/client/utils/identity/resolvers/wellknown"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/opencontainers/go-digest"
)

// resolverSet holds one key resolver per subject scheme.
type resolverSet struct {
	dns       resolvers.Resolver
	did       resolvers.Resolver
	wellknown resolvers.Resolver
	spiffe    resolvers.Resolver
	agntcy    resolvers.Resolver
}

// newNetworkResolvers returns the resolvers that need no per-run state. A nil
// fetcher makes the DID and well-known resolvers use an SSRF-guarded client.
func newNetworkResolvers() resolverSet {
	return resolverSet{
		dns:       dnsresolver.New(),
		did:       didresolver.New(nil),
		wellknown: wellknownresolver.New(nil),
	}
}

// cached returns the set with every resolver remembering its answers, so a
// subject shared by many records is looked up once per run. Not safe for
// concurrent use: a run is sequential.
func (s resolverSet) cached() resolverSet {
	wrap := func(r resolvers.Resolver) resolvers.Resolver {
		if r == nil {
			return nil
		}

		return &cachedResolver{next: r, seen: map[digest.Digest]lookup{}}
	}

	return resolverSet{dns: wrap(s.dns), did: wrap(s.did), wellknown: wrap(s.wellknown), spiffe: wrap(s.spiffe), agntcy: wrap(s.agntcy)}
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
		until := c.ResolutionValidUntil(subject, certificate)
		if until.IsZero() || until.After(time.Now()) {
			return l.keys, l.err
		}
	}

	keys, err := c.next.Resolve(ctx, subject, certificate)
	c.seen[key] = lookup{keys, err}

	return keys, err //nolint:wrapcheck // the answer is passed through as it came
}

func (c *cachedResolver) ResolutionValidUntil(subject string, certificate []byte) time.Time {
	if expiring, ok := c.next.(resolvers.ExpiringResolver); ok {
		return expiring.ResolutionValidUntil(subject, certificate)
	}

	return time.Time{}
}

// failedResolver lets a rotated authority configuration fail its own scheme
// without preventing the other identity schemes from being reconciled.
type failedResolver struct{ err error }

func (r failedResolver) Resolve(context.Context, string, []byte) ([]crypto.PublicKey, error) {
	return nil, r.err
}

// isTransient reports whether a failed key lookup says nothing about the claim:
// a timeout, a refused or dropped connection, a DNS failure other than "no such
// name", or a 5xx, 408 or 429 response. A subject that answers and publishes no
// usable key is not transient.
func isTransient(err error) bool {
	var (
		statusErr *safefetch.StatusError
		dnsErr    *net.DNSError
		opErr     *net.OpError
		netErr    net.Error
	)

	switch {
	case errors.Is(err, safefetch.ErrDisallowedAddress):
		return false
	case errors.Is(err, context.DeadlineExceeded):
		return true
	case errors.As(err, &statusErr):
		return statusErr.Code >= http.StatusInternalServerError ||
			statusErr.Code == http.StatusTooManyRequests || statusErr.Code == http.StatusRequestTimeout
	case errors.As(err, &dnsErr):
		return !dnsErr.IsNotFound
	case errors.As(err, &opErr):
		return true
	case errors.As(err, &netErr):
		return netErr.Timeout()
	}

	return false
}

// forSubject picks the resolver for a claim's subject by its scheme:
//
//	did:web:..., did:key:...   did
//	spiffe://...               spiffe
//	agntcy://...               agntcy
//	https://...                wellknown
//	dns:acme.com, acme.com     dns
func (s resolverSet) forSubject(subject string) (resolvers.Resolver, error) {
	var r resolvers.Resolver

	switch {
	case strings.HasPrefix(subject, "did:"):
		r = s.did
	case strings.HasPrefix(subject, "spiffe://"):
		r = s.spiffe
	case strings.HasPrefix(subject, "agntcy://"):
		r = s.agntcy
	case strings.HasPrefix(subject, "https://"):
		r = s.wellknown
	case strings.HasPrefix(subject, "dns:"), !strings.Contains(subject, ":"):
		r = s.dns
	}

	if r == nil {
		return nil, fmt.Errorf("unsupported subject scheme in %q", subject)
	}

	return r, nil
}
