// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/agentnameservice/ans-sdk-go/models"
)

const (
	// ansScheme prefixes every ANS name.
	ansScheme = "ans://"

	// nameParts is the number of dot-separated parts an ANS name splits into:
	// v<major>, <minor>, <patch> and the host.
	nameParts = 4

	// maxHostLength is the longest name DNS carries. It bounds the host of a
	// subject, and any text taken from a DNS record before an error echoes it.
	maxHostLength = 253

	canonicalForm = "ans://v<major>.<minor>.<patch>.<lowercase host>"
)

var (
	errNotANSName   = errors.New("ans name: subject is not an ANS name (" + canonicalForm + ")")
	errNotCanonical = errors.New("ans name: subject is not in canonical form (" + canonicalForm + ")")
)

// agentName is the structural identity of an ANS name: the lowercase agent
// host and the parsed version. Names are compared structurally, so
// "ans://v1.0.0.Agent.Example.COM" and "ans://v1.0.0.agent.example.com" agree.
type agentName struct {
	host    string
	version models.Version
}

// parseAgentName parses a claim subject and requires the canonical spelling,
// the one the registration authority issues: a lowercase DNS host and a
// numeric version without leading zeros. One attested identity therefore
// verifies under one subject only.
func parseAgentName(subject string) (agentName, error) {
	name, err := parseLooseName(subject)
	if err != nil {
		return agentName{}, err
	}

	if name.String() != subject {
		return agentName{}, errNotCanonical
	}

	return name, nil
}

// parseLooseName parses an ANS name in any spelling: mixed case, leading
// zeros in the version, a trailing dot. parseAgentName adds the canonical
// check; matches uses the loose form for names the log spells.
func parseLooseName(raw string) (agentName, error) {
	rest, ok := strings.CutPrefix(raw, ansScheme)
	if !ok {
		return agentName{}, errNotANSName
	}

	parts := strings.SplitN(rest, ".", nameParts)
	if len(parts) != nameParts || !strings.HasPrefix(parts[0], "v") {
		return agentName{}, errNotANSName
	}

	version, err := models.ParseVersion(parts[0] + "." + parts[1] + "." + parts[2])
	if err != nil {
		return agentName{}, errNotANSName
	}

	host := parts[3]
	if len(host) > maxHostLength {
		return agentName{}, errors.New("ans name: host is longer than a DNS name allows")
	}

	if net.ParseIP(strings.TrimSuffix(host, ".")) != nil {
		return agentName{}, fmt.Errorf("ans name: host %q is an IP address, not a DNS name", host)
	}

	fqdn, err := models.NewFqdn(host)
	if err != nil {
		return agentName{}, fmt.Errorf("ans name: host %q is not a valid DNS name", host)
	}

	return agentName{host: fqdn.String(), version: version}, nil
}

// matches reports whether raw names the same host and version in any spelling.
func (n agentName) matches(raw string) bool {
	other, err := parseLooseName(raw)

	return err == nil && other.host == n.host && other.version.Equal(n.version)
}

// String renders the canonical spelling of the name.
func (n agentName) String() string {
	return ansScheme + n.version.String() + "." + n.host
}
