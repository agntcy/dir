// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/agentnameservice/ans-sdk-go/models"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
)

// TrustedLog is the transparency log a badge record points at, once the
// record has passed the allow-list: the log's origin and the agent's id on it.
type TrustedLog struct {
	// Origin is the log's https origin without a trailing slash, e.g.
	// "https://log.example.com:8443".
	Origin string

	// AgentID is the agent's lowercase UUID on that log.
	AgentID string
}

const (
	// badgeRecordPrefix is prepended to the agent's host to form the badge
	// record's DNS name, e.g. "_ans-badge.agent.example.com".
	badgeRecordPrefix = "_ans-badge."

	// badgeFormat is the only record format accepted.
	badgeFormat = "ans-badge1"

	// badgePathSegments is the segment count of /v1/agents/{agentId} split on
	// "/": the empty segment before the leading slash plus three.
	badgePathSegments = 4
)

// agentIDPattern matches a lowercase RFC 4122 UUID, the only spelling that
// keeps the fetched URL canonical.
var agentIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// badgeRecord is one parsed "_ans-badge" TXT record. A nil version applies
// to every version of the agent.
type badgeRecord struct {
	version *models.Version
	url     string
}

// parseBadgeRecord reads a TXT record of the form
// "v=ans-badge1; version=v1.0.0; url=https://..." (the version is optional).
// A record of any other shape is not a badge record. That is stricter than
// the reference parser in two ways, both deliberate: the legacy "ra-badge1"
// format is not accepted, and a version that does not parse makes the record
// no badge record rather than one that applies to every version.
func parseBadgeRecord(txt string) (badgeRecord, bool) {
	var (
		record badgeRecord
		format string
	)

	for part := range strings.SplitSeq(txt, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}

		switch key {
		case "v":
			format = value
		case "version":
			parsed, err := models.ParseVersion(value)
			if err != nil {
				return badgeRecord{}, false
			}

			record.version = &parsed
		case "url":
			record.url = value
		}
	}

	if format != badgeFormat || record.url == "" {
		return badgeRecord{}, false
	}

	return record, true
}

// lookupBadge finds the one badge record for the agent's version, an exact
// match or else a record without a version, and gates its URL. What the zone
// says is the publisher's doing, so a missing, ambiguous or misdirected
// record is a verdict, and so is an answer that the name does not exist. A
// lookup that gets no answer is left for the caller to classify by its cause:
// the resolver cannot tell the publisher's zone from its own DNS being down.
func (r *Resolver) lookupBadge(ctx context.Context, name agentName) (TrustedLog, error) {
	recordName := badgeRecordPrefix + name.host

	txts, err := r.lookupTXT(ctx, recordName)
	if err != nil {
		err = fmt.Errorf("ans badge: lookup %s: %w", recordName, err)

		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) && dnsErr.IsNotFound {
			return TrustedLog{}, resolvers.Final(err)
		}

		return TrustedLog{}, err
	}

	var exact, versionless []string

	for _, txt := range txts {
		record, ok := parseBadgeRecord(txt)

		switch {
		case !ok:
		case record.version == nil:
			versionless = append(versionless, record.url)
		case record.version.Equal(name.version):
			exact = append(exact, record.url)
		}
	}

	candidates := exact
	if len(candidates) == 0 {
		candidates = versionless
	}

	switch len(candidates) {
	case 0:
		return TrustedLog{}, resolvers.Final(fmt.Errorf("ans badge: no %s record names version %s", recordName, name.version))
	case 1:
		target, err := parseBadgeURL(candidates[0], r.trusted)
		if err != nil {
			return TrustedLog{}, resolvers.Final(err)
		}

		return target, nil
	default:
		return TrustedLog{}, resolvers.Final(fmt.Errorf("ans badge: %d %s records name version %s; expected one", len(candidates), recordName, name.version))
	}
}

// parseBadgeURL is the gate between a DNS record anyone can publish and the
// HTTPS requests the resolver makes. The URL must be https, name a trusted
// log host, and be exactly /v1/agents/{agentId} with a lowercase UUID;
// userinfo, a query, a fragment, dot or empty segments and trailing slashes
// are refused.
func parseBadgeURL(raw string, trusted map[string]struct{}) (TrustedLog, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return TrustedLog{}, fmt.Errorf("ans badge: badge URL %q is not a valid URL", truncate(raw))
	}

	if u.Scheme != "https" {
		return TrustedLog{}, fmt.Errorf("ans badge: badge URL scheme %q is not https", truncate(u.Scheme))
	}

	if u.User != nil {
		return TrustedLog{}, errors.New("ans badge: badge URL must not carry userinfo")
	}

	if u.RawQuery != "" || u.ForceQuery {
		return TrustedLog{}, errors.New("ans badge: badge URL must not carry a query")
	}

	if u.Fragment != "" {
		return TrustedLog{}, errors.New("ans badge: badge URL must not carry a fragment")
	}

	if u.Host == "" {
		return TrustedLog{}, errors.New("ans badge: badge URL has no host")
	}

	host, err := normalizeHost(u.Host)
	if err != nil {
		return TrustedLog{}, fmt.Errorf("ans badge: badge URL host is malformed: %q", truncate(u.Host))
	}

	if _, ok := trusted[host]; !ok {
		return TrustedLog{}, fmt.Errorf("ans badge: log host %q is not a trusted transparency log", truncate(host))
	}

	// The escaped path is checked so percent-encoding cannot spell the segments.
	segments := strings.Split(u.EscapedPath(), "/")
	if len(segments) != badgePathSegments || segments[0] != "" || segments[1] != "v1" || segments[2] != "agents" {
		return TrustedLog{}, errors.New("ans badge: badge URL path must be /v1/agents/{agentId}")
	}

	if !agentIDPattern.MatchString(segments[3]) {
		return TrustedLog{}, errors.New("ans badge: badge URL agent id is not a lowercase UUID")
	}

	return TrustedLog{Origin: "https://" + host, AgentID: segments[3]}, nil
}

// truncate bounds text taken from DNS or a transparency log before an error
// echoes it.
func truncate(text string) string {
	if len(text) <= maxHostLength {
		return text
	}

	return text[:maxHostLength]
}
