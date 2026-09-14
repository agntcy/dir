// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/agentnameservice/ans-sdk-go/verify"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
)

// badgeTarget is the transparency log and agent a badge record points at.
type badgeTarget struct {
	// LogBase is the log's origin, scheme://host, without a trailing slash.
	LogBase string

	// LogHost is the normalized host, the key for the allow-list and the breaker.
	LogHost string

	// AgentID is the lowercase agent UUID from the badge path.
	AgentID string
}

// agentIDPattern matches an RFC 4122 UUID in either case.
var agentIDPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

const (
	// badgePathSegments is the segment count of /v1/agents/{agentId} split on
	// "/": the empty segment before the leading slash plus three.
	badgePathSegments = 4

	// maxHostLength bounds a host echoed in an error text: the longest name
	// DNS carries, so a record anyone can publish cannot flood the stored error.
	maxHostLength = 253
)

// parseBadgeURL is the SSRF gate between a DNS record anyone can publish and
// the HTTPS requests the resolver makes. The URL must be https and name
// exactly /v1/agents/{agentId} with a UUID agent id; query, fragment,
// userinfo, dot or empty segments and trailing slashes are rejected. The
// caller checks the normalized host against the allow-list.
//
// The SDK's verify.URLValidator is not reused: it accepts port 443 only,
// which fails the local demo and any non-default port, and it does not check
// the path shape.
func parseBadgeURL(raw string) (badgeTarget, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL is not a valid URL")
	}

	if u.Scheme != "https" {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL scheme is not https")
	}

	if u.User != nil {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL must not carry userinfo")
	}

	if u.RawQuery != "" || u.ForceQuery {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL must not carry a query")
	}

	if u.Fragment != "" {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL must not carry a fragment")
	}

	if u.Host == "" {
		return badgeTarget{}, fail(stageBadgeURL, "badge URL has no host")
	}

	host, err := ansconfig.NormalizeHost(u.Host)
	if err != nil {
		return badgeTarget{}, failWith(stageBadgeURL, err, "badge URL host is malformed")
	}

	agentID, err := parseBadgePath(u.EscapedPath())
	if err != nil {
		return badgeTarget{}, err
	}

	return badgeTarget{LogBase: "https://" + host, LogHost: host, AgentID: agentID}, nil
}

// parseBadgePath returns the agent id from a path that is exactly
// /v1/agents/{agentId}. It works on the escaped path so percent-encoding
// cannot spell the literal segments.
func parseBadgePath(path string) (string, error) {
	segments := strings.Split(path, "/")
	if len(segments) != badgePathSegments || segments[0] != "" || segments[1] != "v1" || segments[2] != "agents" {
		return "", fail(stageBadgeURL, "badge URL path must be /v1/agents/{agentId}")
	}

	if !agentIDPattern.MatchString(segments[3]) {
		return "", fail(stageBadgeURL, "badge URL agent id is not a UUID")
	}

	return strings.ToLower(segments[3]), nil
}

// badgeSourceName names the DNS record a badge was read from.
func badgeSourceName(source verify.BadgeRecordSource) string {
	if source == verify.BadgeRecordSourceRaBadge {
		return "_ra-badge"
	}

	return "_ans-badge"
}

// truncateHost bounds a host taken from DNS or from a claim before an error
// text echoes it.
func truncateHost(host string) string {
	if len(host) <= maxHostLength {
		return host
	}

	return host[:maxHostLength]
}
