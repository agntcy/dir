// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"fmt"

	"github.com/agentnameservice/ans-sdk-go/models"
	"github.com/agentnameservice/ans-sdk-go/verify"
)

// ansScheme prefixes every ANS name.
const ansScheme = "ans://"

// agentName is the structural identity of an ANS name: the lowercase agent
// host, its validated DNS form, and the parsed version. Names are compared
// structurally so that "ans://v1.0.0.Agent.Example.COM" and
// "ans://v1.0.0.agent.example.com" agree.
type agentName struct {
	host    string
	fqdn    models.Fqdn
	version models.Version
}

// parseAgentName parses a claim subject and requires the canonical spelling,
// the one the registration authority issues: a lowercase DNS host and a
// numeric version without leading zeros. One attested identity therefore
// verifies under one subject only.
func parseAgentName(subject string) (agentName, error) {
	parsed, err := verify.ParseAnsName(subject)
	if err != nil {
		return agentName{}, failWith(stageName, err, "subject is not an ANS name (ans://v<major>.<minor>.<patch>.<host>)")
	}

	fqdn, err := models.NewFqdn(parsed.Host)
	if err != nil {
		return agentName{}, failWith(stageName, err, fmt.Sprintf("host %q is not a valid DNS name", truncateHost(parsed.Host)))
	}

	name := agentName{host: fqdn.String(), fqdn: fqdn, version: parsed.Version}
	if name.String() != subject {
		return agentName{}, fail(stageName, "subject is not in canonical form (ans://v<major>.<minor>.<patch>.<lowercase host>)")
	}

	return name, nil
}

// matches reports whether raw parses to the same host and version.
func (n agentName) matches(raw string) bool {
	parsed, err := verify.ParseAnsName(raw)

	return err == nil && n.equals(parsed)
}

func (n agentName) equals(parsed *verify.AnsName) bool {
	return parsed != nil && parsed.Host == n.host && parsed.Version.Equal(n.version)
}

// String renders the canonical spelling of the name. It is the one place the
// spelling is constructed; the SDK's AnsName.String returns its raw input.
func (n agentName) String() string {
	return ansScheme + n.version.String() + "." + n.host
}
