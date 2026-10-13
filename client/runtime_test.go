// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"io"
	"testing"

	runtimev1 "github.com/agntcy/dir/api/runtime/v1"
	"google.golang.org/grpc"
)

// mockWorkloadStream returns the given workloads, then io.EOF.
type mockWorkloadStream struct {
	workloads []*runtimev1.Workload
	index     int
	grpc.ClientStream
}

func (m *mockWorkloadStream) Recv() (*runtimev1.Workload, error) {
	if m.index >= len(m.workloads) {
		return nil, io.EOF
	}

	workload := m.workloads[m.index]
	m.index++

	return workload, nil
}

// mockDiscoveryClient records the ListWorkloads request it receives.
type mockDiscoveryClient struct {
	runtimev1.DiscoveryServiceClient

	workloads []*runtimev1.Workload
	got       *runtimev1.ListWorkloadsRequest
}

func (m *mockDiscoveryClient) ListWorkloads(_ context.Context, req *runtimev1.ListWorkloadsRequest, _ ...grpc.CallOption) (runtimev1.DiscoveryService_ListWorkloadsClient, error) {
	m.got = req

	return &mockWorkloadStream{workloads: m.workloads}, nil
}

func TestListWorkloadsWithRequest_ForwardsAllFilters(t *testing.T) {
	discovery := &mockDiscoveryClient{
		workloads: []*runtimev1.Workload{{Id: "w1"}, {Id: "w2"}},
	}
	c := &Client{DiscoveryServiceClient: discovery}

	req := &runtimev1.ListWorkloadsRequest{
		Labels:           map[string]string{"org.agntcy/discover": "true"},
		Skills:           []string{"natural_language_processing/*"},
		Domains:          []string{"technology/software_engineering"},
		LocatorProtocols: []string{"slim"},
		Names:            []string{"network-*"},
		Authors:          []string{"did:key:zHuman"},
		SchemaVersions:   []string{"1.*"},
	}

	workloads, err := c.ListWorkloadsWithRequest(t.Context(), req)
	if err != nil {
		t.Fatalf("ListWorkloadsWithRequest: %v", err)
	}

	if len(workloads) != 2 {
		t.Fatalf("got %d workloads, want 2", len(workloads))
	}

	if discovery.got != req {
		t.Fatalf("request was not forwarded unchanged: got %v, want %v", discovery.got, req)
	}
}

func TestListWorkloads_SendsOnlyLabels(t *testing.T) {
	discovery := &mockDiscoveryClient{}
	c := &Client{DiscoveryServiceClient: discovery}

	labels := map[string]string{"org.agntcy/agent-type": "a2a"}

	if _, err := c.ListWorkloads(t.Context(), labels); err != nil {
		t.Fatalf("ListWorkloads: %v", err)
	}

	if discovery.got.GetLabels()["org.agntcy/agent-type"] != "a2a" {
		t.Fatalf("labels not forwarded: %v", discovery.got.GetLabels())
	}

	if len(discovery.got.GetSkills()) != 0 || len(discovery.got.GetDomains()) != 0 || len(discovery.got.GetLocatorProtocols()) != 0 ||
		len(discovery.got.GetNames()) != 0 || len(discovery.got.GetAuthors()) != 0 || len(discovery.got.GetSchemaVersions()) != 0 {
		t.Fatalf("unexpected filters set: %v", discovery.got)
	}
}
