// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

type fakeStream struct {
	grpc.ClientStream

	workloads []*runtimev1.Workload
	i         int
}

func (s *fakeStream) Recv() (*runtimev1.Workload, error) {
	if s.i >= len(s.workloads) {
		return nil, io.EOF
	}

	s.i++

	return s.workloads[s.i-1], nil
}

// fakeDiscovery ignores every filter, like a server that predates them.
type fakeDiscovery struct {
	runtimev1.DiscoveryServiceClient

	workloads []*runtimev1.Workload
	got       *runtimev1.ListWorkloadsRequest
}

func (f *fakeDiscovery) ListWorkloads(_ context.Context, req *runtimev1.ListWorkloadsRequest, _ ...grpc.CallOption) (runtimev1.DiscoveryService_ListWorkloadsClient, error) {
	f.got = req

	return &fakeStream{workloads: f.workloads}, nil
}

func card(t *testing.T, name, skill string) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(map[string]any{
		"name":    name,
		"version": "1.0.0",
		"capabilities": map[string]any{"extensions": []any{
			map[string]any{
				"uri":    runtimev1.DefaultOASFSchemaURL + "/1.1.0/objects/discovery",
				"params": map[string]any{"schema_version": "1.1.0", "skills": []any{map[string]any{"name": skill, "id": 1}}},
			},
			map[string]any{
				"uri":    runtimev1.DefaultOASFSchemaURL + "/1.1.0/objects/identity",
				"params": map[string]any{"agent_did": "did:key:zAgent", "human_did": "did:key:zHuman"},
			},
		}},
	})
	require.NoError(t, err)

	return s
}

func resetOpts(t *testing.T) {
	t.Helper()

	saved := listOpts

	t.Cleanup(func() { listOpts = saved })

	listOpts.Names, listOpts.Skills, listOpts.Domains, listOpts.Authors = nil, nil, nil, nil
	listOpts.SchemaVersions, listOpts.LocatorProtocols, listOpts.RuntimeTypes, listOpts.Labels = nil, nil, nil, nil
	listOpts.ReachableFrom = ""
	listOpts.SchemaURLs = []string{runtimev1.DefaultOASFSchemaURL}
}

func run(t *testing.T, discovery *fakeDiscovery, args ...string) []listedWorkload {
	t.Helper()

	c := &client.Client{DiscoveryServiceClient: discovery}

	var out bytes.Buffer

	listCmd.SetOut(&out)
	listCmd.SetContext(ctxUtils.SetClientForContext(context.Background(), c))
	require.NoError(t, listCmd.ParseFlags(append([]string{"--output", "json"}, args...)))
	require.NoError(t, runList(listCmd))

	var listed []listedWorkload
	require.NoError(t, json.Unmarshal(out.Bytes(), &listed), out.String())

	return listed
}

func TestRunList_SendsFiltersAndFiltersAgainOnTheClient(t *testing.T) {
	resetOpts(t)

	discovery := &fakeDiscovery{workloads: []*runtimev1.Workload{
		{
			Id: "debugger", Name: "debugger", Runtime: "process", Type: "process",
			Labels:   map[string]string{"org.agntcy/agent-record": "bafy-source"},
			Locators: []*runtimev1.WorkloadLocator{{Protocol: "slim", Url: "slim://org/ns/debugger"}},
			Services: &runtimev1.WorkloadServices{A2A: card(t, "debugger", "software_engineering/code_debugging/error_handling")},
		},
		{
			Id: "summarizer", Runtime: "docker",
			Services: &runtimev1.WorkloadServices{A2A: card(t, "summarizer", "natural_language_processing/summarization")},
		},
	}}

	listed := run(t, discovery, "--skill", "software_engineering/*", "--locator-protocol", "slim", "--label", "org.agntcy/agent-record=bafy")

	require.Equal(t, []string{"software_engineering/*"}, discovery.got.GetSkills())
	require.Equal(t, []string{"slim"}, discovery.got.GetLocatorProtocols())
	require.Equal(t, map[string]string{"org.agntcy/agent-record": "bafy"}, discovery.got.GetLabels())

	require.Len(t, listed, 1, "the client must drop what an older server didn't filter")

	got := listed[0]
	require.Equal(t, "debugger", got.ID)
	require.Equal(t, "debugger", got.Agent.Name)
	require.Equal(t, "1.0.0", got.Agent.Version)
	require.Equal(t, "1.1.0", got.Agent.SchemaVersion)
	require.Equal(t, []string{"did:key:zHuman"}, got.Agent.Authors)
	require.Equal(t, "software_engineering/code_debugging/error_handling", got.Agent.Skills[0].Name)
	require.Equal(t, "bafy-source", got.SourceRecord)
	require.Equal(t, "slim://org/ns/debugger", got.Locators[0].URL)
	require.NotEmpty(t, got.Workload)
}

func TestBuildRequest_RejectsMalformedLabel(t *testing.T) {
	resetOpts(t)

	listOpts.Labels = []string{"no-equals-sign"}

	_, err := buildRequest()
	require.Error(t, err)
}

func TestBuildRequest_Reachability(t *testing.T) {
	resetOpts(t)

	req, err := buildRequest()
	require.NoError(t, err)
	require.Nil(t, req.ReachableFromWorkloadId)

	listOpts.ReachableFrom = "w1"

	req, err = buildRequest()
	require.NoError(t, err)
	require.Equal(t, "w1", req.GetReachableFromWorkloadId())
}
