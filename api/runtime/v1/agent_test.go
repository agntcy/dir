// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func mustStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	return s
}

func card(extensions ...any) map[string]any {
	return map[string]any{
		"name":         "network-debugger",
		"capabilities": map[string]any{"extensions": extensions},
	}
}

func discoveryExtension(uri string) map[string]any {
	return map[string]any{
		"uri": uri,
		"params": map[string]any{
			"schema_version": "1.2.0",
			"skills":         []any{map[string]any{"name": "software_engineering/code_debugging/error_handling", "id": 60602}},
			"domains":        []any{map[string]any{"name": "technology/networking/network_security", "id": 10305}},
		},
	}
}

func identityExtension(uri string) map[string]any {
	return map[string]any{
		"uri":    uri,
		"params": map[string]any{"agent_did": "did:key:zAgent", "human_did": "did:key:zHuman"},
	}
}

func oasfService() map[string]any {
	return map[string]any{
		"cid": "bafy-record",
		"record": map[string]any{
			"name":           "record-agent",
			"schema_version": "1.1.0",
			"skills":         []any{map[string]any{"name": "natural_language_processing/summarization", "id": 10201}},
			"domains":        []any{map[string]any{"name": "technology/software_engineering", "id": 102}},
			"authors":        []any{"Alice <alice@example.com>"},
		},
	}
}

func workload(t *testing.T, a2a, oasf map[string]any) *Workload {
	t.Helper()

	services := &WorkloadServices{}
	if a2a != nil {
		services.A2A = mustStruct(t, a2a)
	}

	if oasf != nil {
		services.Oasf = mustStruct(t, oasf)
	}

	return &Workload{Id: "w1", Name: "runtime-name", Services: services}
}

func TestAgentInfo_FromCardExtensions(t *testing.T) {
	w := workload(t, card(
		discoveryExtension("https://schema.oasf.outshift.com/1.2.0/objects/discovery"),
		identityExtension("https://schema.oasf.outshift.com/objects/identity"),
	), oasfService())

	info := w.AgentInfo(nil)

	if info.Name != "network-debugger" {
		t.Errorf("Name = %q, want the card's name", info.Name)
	}

	if info.SchemaVersion != "1.2.0" {
		t.Errorf("SchemaVersion = %q, want 1.2.0", info.SchemaVersion)
	}

	if len(info.Skills) != 1 || info.Skills[0] != (OASFClass{Name: "software_engineering/code_debugging/error_handling", ID: 60602}) {
		t.Errorf("Skills = %v, want the card's skill", info.Skills)
	}

	if len(info.Domains) != 1 || info.Domains[0].Name != "technology/networking/network_security" {
		t.Errorf("Domains = %v, want the card's domain", info.Domains)
	}

	if len(info.Authors) != 1 || info.Authors[0] != "did:key:zHuman" {
		t.Errorf("Authors = %v, want the identity extension's human DID", info.Authors)
	}
}

func TestAgentInfo_FallsBackToRecord(t *testing.T) {
	w := workload(t, map[string]any{"description": "card without name or extensions"}, oasfService())

	info := w.AgentInfo(nil)

	if info.Name != "record-agent" || info.SchemaVersion != "1.1.0" {
		t.Errorf("Name, SchemaVersion = %q, %q; want the record's", info.Name, info.SchemaVersion)
	}

	if len(info.Skills) != 1 || info.Skills[0].Name != "natural_language_processing/summarization" {
		t.Errorf("Skills = %v, want the record's skill", info.Skills)
	}

	if len(info.Authors) != 1 || info.Authors[0] != "Alice <alice@example.com>" {
		t.Errorf("Authors = %v, want the record's authors", info.Authors)
	}
}

func TestAgentInfo_IgnoresExtensionsFromOtherSchemaServers(t *testing.T) {
	w := workload(t, card(
		discoveryExtension("https://evil.example.com/1.2.0/objects/discovery"),
		discoveryExtension("https://schema.oasf.outshift.com/latest/objects/discovery"),
	), nil)

	if info := w.AgentInfo(nil); len(info.Skills) != 0 {
		t.Errorf("Skills = %v, want none from an unaccepted server or a malformed version", info.Skills)
	}
}

func TestAgentInfo_AcceptsConfiguredSchemaServer(t *testing.T) {
	w := workload(t, card(discoveryExtension("http://localhost:8080/1.2.0/objects/discovery")), nil)

	if info := w.AgentInfo(nil); len(info.Skills) != 0 {
		t.Errorf("Skills = %v, want none when localhost is not accepted", info.Skills)
	}

	if info := w.AgentInfo([]string{"http://localhost:8080/"}); len(info.Skills) != 1 {
		t.Errorf("Skills = %v, want the skill from the configured server", info.Skills)
	}
}

func TestAgentInfo_EmptyWorkload(t *testing.T) {
	var w *Workload

	info := w.AgentInfo(nil)
	if info.Name != "" || len(info.Skills) != 0 || len(info.Authors) != 0 {
		t.Errorf("AgentInfo of nil workload = %+v, want zero value", info)
	}
}

func TestMatchesFilters(t *testing.T) {
	w := workload(t, card(
		discoveryExtension("https://schema.oasf.outshift.com/1.2.0/objects/discovery"),
		identityExtension("https://schema.oasf.outshift.com/1.2.0/objects/identity"),
	), nil)
	w.Locators = []*WorkloadLocator{{Protocol: "slim", Url: "slim://org/ns/agent"}}
	w.Runtime = "docker"
	w.Labels = map[string]string{"org.agntcy/agent-type": "a2a", "team": "net-ops"}

	tests := []struct {
		name string
		req  *ListWorkloadsRequest
		want bool
	}{
		{"no filters", &ListWorkloadsRequest{}, true},
		{"exact skill, other case", &ListWorkloadsRequest{Skills: []string{"Software_Engineering/Code_Debugging/Error_Handling"}}, true},
		{"skill wildcard", &ListWorkloadsRequest{Skills: []string{"software_engineering/*"}}, true},
		{"skill prefix without wildcard", &ListWorkloadsRequest{Skills: []string{"software_engineering"}}, false},
		{"one of several skills", &ListWorkloadsRequest{Skills: []string{"other", "*error_handling"}}, true},
		{"domain", &ListWorkloadsRequest{Domains: []string{"technology/networking/*"}}, true},
		{"wrong domain", &ListWorkloadsRequest{Domains: []string{"finance/*"}}, false},
		{"name", &ListWorkloadsRequest{Names: []string{"network-*"}}, true},
		{"runtime name is not the agent name", &ListWorkloadsRequest{Names: []string{"runtime-name"}}, false},
		{"author", &ListWorkloadsRequest{Authors: []string{"did:key:zHuman"}}, true},
		{"schema version wildcard", &ListWorkloadsRequest{SchemaVersions: []string{"1.?.*"}}, true},
		{"schema version mismatch", &ListWorkloadsRequest{SchemaVersions: []string{"1.1.0"}}, false},
		{"locator protocol", &ListWorkloadsRequest{LocatorProtocols: []string{"SLIM"}}, true},
		{"missing locator protocol", &ListWorkloadsRequest{LocatorProtocols: []string{"grpc"}}, false},
		{"filters combine with AND", &ListWorkloadsRequest{Skills: []string{"software_engineering/*"}, Authors: []string{"did:key:zOther"}}, false},
		{"label regex", &ListWorkloadsRequest{Labels: map[string]string{"team": "^net-"}}, true},
		{"label regex matches anywhere", &ListWorkloadsRequest{Labels: map[string]string{"org.agntcy/agent-type": "2"}}, true},
		{"label value mismatch", &ListWorkloadsRequest{Labels: map[string]string{"team": "^ops$"}}, false},
		{"missing label", &ListWorkloadsRequest{Labels: map[string]string{"absent": ".*"}}, false},
		{"invalid label regex falls back to exact", &ListWorkloadsRequest{Labels: map[string]string{"team": "net-ops("}}, false},
		{"all labels must match", &ListWorkloadsRequest{Labels: map[string]string{"team": "net-ops", "org.agntcy/agent-type": "mcp"}}, false},
		{"runtime type", &ListWorkloadsRequest{RuntimeTypes: []string{"kubernetes", "Docker"}}, true},
		{"other runtime type", &ListWorkloadsRequest{RuntimeTypes: []string{"process"}}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := w.MatchesFilters(tt.req, FilterOptions{}); got != tt.want {
				t.Errorf("MatchesFilters = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesFilters_WorkloadWithoutAgentInfo(t *testing.T) {
	w := &Workload{Id: "w1"}

	if !w.MatchesFilters(&ListWorkloadsRequest{}, FilterOptions{}) {
		t.Error("a workload without agent info must match an empty request")
	}

	if w.MatchesFilters(&ListWorkloadsRequest{Skills: []string{"*"}}, FilterOptions{}) {
		t.Error("a workload without skills must not match a skill filter")
	}
}

func TestMatchesFilters_InvalidLabelRegexMatchesExactValue(t *testing.T) {
	w := &Workload{Id: "w1", Labels: map[string]string{"pattern": "a("}}

	if !w.MatchesFilters(&ListWorkloadsRequest{Labels: map[string]string{"pattern": "a("}}, FilterOptions{}) {
		t.Error("an invalid regular expression must match its exact value")
	}
}

func TestMatchesFilters_ReachableFrom(t *testing.T) {
	from := &Workload{Id: "caller", Runtime: "docker", IsolationGroups: []string{"team-a", "shared"}}
	sameGroup := &Workload{Id: "w1", Runtime: "docker", IsolationGroups: []string{"shared"}}
	otherGroup := &Workload{Id: "w2", Runtime: "docker", IsolationGroups: []string{"team-b"}}

	id := "caller"
	req := &ListWorkloadsRequest{ReachableFromWorkloadId: &id}
	opts := FilterOptions{ReachableFrom: from}

	if !sameGroup.MatchesFilters(req, opts) {
		t.Error("a workload sharing an isolation group must be reachable")
	}

	if otherGroup.MatchesFilters(req, opts) {
		t.Error("a workload in another isolation group must not be reachable")
	}

	if from.MatchesFilters(req, opts) {
		t.Error("the reference workload itself must not be returned")
	}

	if sameGroup.MatchesFilters(req, FilterOptions{}) {
		t.Error("without the reference workload nothing matches")
	}

	other := "someone-else"
	if sameGroup.MatchesFilters(&ListWorkloadsRequest{ReachableFromWorkloadId: &other}, opts) {
		t.Error("a reference workload with another ID must not be used")
	}
}

func TestMatchesFilters_ReachabilityIsScopedToRuntimeAndDiscoveryInstance(t *testing.T) {
	instance := func(id string) map[string]string { return map[string]string{DiscoveryInstanceAnnotation: id} }

	from := &Workload{Id: "caller", Runtime: "process", IsolationGroups: []string{"host"}, Annotations: instance("laptop-a")}
	sameHost := &Workload{Id: "w1", Runtime: "process", IsolationGroups: []string{"host"}, Annotations: instance("laptop-a")}
	otherHost := &Workload{Id: "w2", Runtime: "process", IsolationGroups: []string{"host"}, Annotations: instance("laptop-b")}
	otherRuntime := &Workload{Id: "w3", Runtime: "docker", IsolationGroups: []string{"host"}, Annotations: instance("laptop-a")}

	id := "caller"
	req := &ListWorkloadsRequest{ReachableFromWorkloadId: &id}
	opts := FilterOptions{ReachableFrom: from}

	if !sameHost.MatchesFilters(req, opts) {
		t.Error("a process on the same host must be reachable")
	}

	if otherHost.MatchesFilters(req, opts) {
		t.Error("a process from another discovery instance must not be reachable through a same-named group")
	}

	if otherRuntime.MatchesFilters(req, opts) {
		t.Error("a workload of another runtime must not be reachable through a same-named group")
	}
}

func TestAgentInfo_PartialExtensionKeepsRecordValues(t *testing.T) {
	partial := map[string]any{
		"uri": "https://schema.oasf.outshift.com/1.2.0/objects/discovery",
		"params": map[string]any{
			"skills": []any{map[string]any{"name": "software_engineering/code_debugging/error_handling", "id": 60602}},
		},
	}

	info := workload(t, card(partial), oasfService()).AgentInfo(nil)

	if len(info.Skills) != 1 || info.Skills[0].Name != "software_engineering/code_debugging/error_handling" {
		t.Errorf("Skills = %v, want the card's", info.Skills)
	}

	if len(info.Domains) != 1 || info.Domains[0].Name != "technology/software_engineering" {
		t.Errorf("Domains = %v, want the record's, since the card has none", info.Domains)
	}

	if info.SchemaVersion != "1.1.0" {
		t.Errorf("SchemaVersion = %q, want the record's, since the card has none", info.SchemaVersion)
	}
}

func TestMatchesFilters_NilRequestMatchesEveryWorkload(t *testing.T) {
	workloads := []*Workload{
		{Id: "bare"},
		{Id: "labelled", Runtime: "docker", Labels: map[string]string{"app": "x"}},
	}

	for _, w := range workloads {
		if !w.MatchesFilters(nil, FilterOptions{}) {
			t.Errorf("MatchesFilters(nil) = false for %q, want true", w.GetId())
		}
	}
}
