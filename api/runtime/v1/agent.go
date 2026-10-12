// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"regexp"
	"slices"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// DefaultOASFSchemaURL is the OASF schema server whose extensions are accepted
// when no other schema servers are configured.
const DefaultOASFSchemaURL = "https://schema.oasf.outshift.com"

const (
	discoveryObjectPath = "/objects/discovery"
	identityObjectPath  = "/objects/identity"
)

// OASFClass is an OASF skill or domain reference.
type OASFClass struct {
	Name string
	ID   uint32
}

// AgentInfo is what a workload says about the agent it runs.
//
// It is read from the workload's A2A card (services.a2a): the card's name, and the OASF
// discovery and identity extensions served by an accepted schema server. Whatever the card
// does not provide is read from the OASF record the workload references (services.oasf).
// The values are claims made by whoever deployed the workload; nothing here is verified.
type AgentInfo struct {
	// Name is the agent's name.
	Name string
	// SchemaVersion is the OASF schema version of Skills and Domains.
	SchemaVersion string
	// Skills are the agent's OASF skills.
	Skills []OASFClass
	// Domains are the agent's OASF domains.
	Domains []OASFClass
	// Authors are the agent's authors, e.g. the DID of the human vouching for it.
	Authors []string
}

// AgentInfo returns what the workload says about the agent it runs.
// acceptedSchemaURLs lists the OASF schema servers whose extensions are read from the card;
// when empty, only DefaultOASFSchemaURL is accepted.
func (x *Workload) AgentInfo(acceptedSchemaURLs []string) AgentInfo {
	if len(acceptedSchemaURLs) == 0 {
		acceptedSchemaURLs = []string{DefaultOASFSchemaURL}
	}

	card := x.GetServices().GetA2A().GetFields()
	record := x.GetServices().GetOasf().GetFields()["record"].GetStructValue().GetFields()

	var info AgentInfo

	info.Name = card["name"].GetStringValue()
	if info.Name == "" {
		info.Name = record["name"].GetStringValue()
	}

	if discovery := cardExtension(card, discoveryObjectPath, acceptedSchemaURLs); discovery != nil {
		info.SchemaVersion = discovery["schema_version"].GetStringValue()
		info.Skills = classes(discovery["skills"])
		info.Domains = classes(discovery["domains"])
	} else {
		info.SchemaVersion = record["schema_version"].GetStringValue()
		info.Skills = classes(record["skills"])
		info.Domains = classes(record["domains"])
	}

	if identity := cardExtension(card, identityObjectPath, acceptedSchemaURLs); identity != nil {
		if human := identity["human_did"].GetStringValue(); human != "" {
			info.Authors = []string{human}
		}
	}

	if len(info.Authors) == 0 {
		for _, author := range record["authors"].GetListValue().GetValues() {
			if s := author.GetStringValue(); s != "" {
				info.Authors = append(info.Authors, s)
			}
		}
	}

	return info
}

// FilterOptions holds what MatchesFilters needs besides the request itself.
type FilterOptions struct {
	// AcceptedSchemaURLs is passed to AgentInfo.
	AcceptedSchemaURLs []string
	// ReachableFrom is the workload named by the request's reachable_from_workload_id,
	// looked up by the caller. When the request sets that field and ReachableFrom is nil
	// or has another ID, no workload matches.
	ReachableFrom *Workload
}

// MatchesFilters reports whether the workload matches every filter of req, as documented
// on ListWorkloadsRequest.
func (x *Workload) MatchesFilters(req *ListWorkloadsRequest, opts FilterOptions) bool {
	return x.matchesRuntimeFilters(req, opts) && x.matchesAgentFilters(req, opts)
}

// matchesRuntimeFilters evaluates the filters on the workload itself: labels, runtime type,
// reachability and locator protocols.
func (x *Workload) matchesRuntimeFilters(req *ListWorkloadsRequest, opts FilterOptions) bool {
	if !x.matchesLabels(req.GetLabels()) {
		return false
	}

	if len(req.GetRuntimeTypes()) > 0 && !containsFold(req.GetRuntimeTypes(), x.GetRuntime()) {
		return false
	}

	if req.ReachableFromWorkloadId != nil && !x.reachableFrom(req.GetReachableFromWorkloadId(), opts.ReachableFrom) {
		return false
	}

	return len(req.GetLocatorProtocols()) == 0 || x.hasLocatorProtocol(req.GetLocatorProtocols())
}

// matchesAgentFilters evaluates the filters on the agent the workload runs: names, skills,
// domains, authors and schema versions.
func (x *Workload) matchesAgentFilters(req *ListWorkloadsRequest, opts FilterOptions) bool {
	if len(req.GetNames()) == 0 && len(req.GetSkills()) == 0 && len(req.GetDomains()) == 0 &&
		len(req.GetAuthors()) == 0 && len(req.GetSchemaVersions()) == 0 {
		return true
	}

	info := x.AgentInfo(opts.AcceptedSchemaURLs)

	return matchesAny(req.GetNames(), []string{info.Name}) &&
		matchesAny(req.GetSkills(), classNames(info.Skills)) &&
		matchesAny(req.GetDomains(), classNames(info.Domains)) &&
		matchesAny(req.GetAuthors(), info.Authors) &&
		matchesAny(req.GetSchemaVersions(), []string{info.SchemaVersion})
}

// matchesLabels requires every label to be present with a value matching its pattern:
// a regular expression matched anywhere in the value, or the exact value when the pattern
// is not a valid regular expression.
func (x *Workload) matchesLabels(filters map[string]string) bool {
	for key, pattern := range filters {
		value, exists := x.GetLabels()[key]
		if !exists {
			return false
		}

		matched, err := regexp.MatchString(pattern, value)
		if err != nil {
			matched = value == pattern
		}

		if !matched {
			return false
		}
	}

	return true
}

// reachableFrom reports whether the workload shares an isolation group with from,
// the workload with ID fromID. A workload is not reachable from itself.
func (x *Workload) reachableFrom(fromID string, from *Workload) bool {
	if from == nil || from.GetId() != fromID || x.GetId() == fromID {
		return false
	}

	for _, group := range x.GetIsolationGroups() {
		if slices.Contains(from.GetIsolationGroups(), group) {
			return true
		}
	}

	return false
}

func containsFold(values []string, value string) bool {
	for _, v := range values {
		if strings.EqualFold(v, value) {
			return true
		}
	}

	return false
}

func (x *Workload) hasLocatorProtocol(protocols []string) bool {
	for _, locator := range x.GetLocators() {
		if containsFold(protocols, locator.GetProtocol()) {
			return true
		}
	}

	return false
}

// cardExtension returns the params of the card's extension whose URI is
// <accepted schema URL>[/<version>]<objectPath>, or nil if there is none.
func cardExtension(card map[string]*structpb.Value, objectPath string, acceptedSchemaURLs []string) map[string]*structpb.Value {
	extensions := card["capabilities"].GetStructValue().GetFields()["extensions"].GetListValue().GetValues()

	for _, extension := range extensions {
		fields := extension.GetStructValue().GetFields()
		if isAcceptedObjectURI(fields["uri"].GetStringValue(), objectPath, acceptedSchemaURLs) {
			return fields["params"].GetStructValue().GetFields()
		}
	}

	return nil
}

// versionSegment matches the optional schema version between the schema URL and the object path.
var versionSegment = regexp.MustCompile(`^(/[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?)?$`)

func isAcceptedObjectURI(uri, objectPath string, acceptedSchemaURLs []string) bool {
	rest, ok := strings.CutSuffix(uri, objectPath)
	if !ok {
		return false
	}

	for _, base := range acceptedSchemaURLs {
		base = strings.TrimSuffix(base, "/")
		if version, ok := strings.CutPrefix(rest, base); ok && versionSegment.MatchString(version) {
			return true
		}
	}

	return false
}

func classes(value *structpb.Value) []OASFClass {
	items := value.GetListValue().GetValues()
	if len(items) == 0 {
		return nil
	}

	out := make([]OASFClass, 0, len(items))

	for _, item := range items {
		fields := item.GetStructValue().GetFields()

		name := fields["name"].GetStringValue()
		if name == "" {
			continue
		}

		out = append(out, OASFClass{Name: name, ID: uint32(fields["id"].GetNumberValue())})
	}

	return out
}

func classNames(classes []OASFClass) []string {
	names := make([]string, 0, len(classes))
	for _, class := range classes {
		names = append(names, class.Name)
	}

	return names
}

// matchesAny reports whether any value matches any pattern. An empty pattern list matches.
func matchesAny(patterns, values []string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		for _, value := range values {
			if value != "" && matchPattern(pattern, value) {
				return true
			}
		}
	}

	return false
}

// matchPattern matches value against pattern as record search does: exactly (case-insensitive),
// or with '*' matching any sequence of characters and '?' matching a single character.
func matchPattern(pattern, value string) bool {
	if !strings.ContainsAny(pattern, "*?") {
		return strings.EqualFold(pattern, value)
	}

	var expr strings.Builder

	expr.WriteString("(?is)^")

	for _, r := range pattern {
		switch r {
		case '*':
			expr.WriteString(".*")
		case '?':
			expr.WriteString(".")
		default:
			expr.WriteString(regexp.QuoteMeta(string(r)))
		}
	}

	expr.WriteString("$")

	return regexp.MustCompile(expr.String()).MatchString(value)
}
