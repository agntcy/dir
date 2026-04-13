// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"testing"

	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSourceItem_NameVersion_MCP(t *testing.T) {
	t.Parallel()

	s := MCPSourceItem(mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "n", Version: "v"}})
	if got := s.NameVersion(); got != "n@v" {
		t.Errorf("NameVersion = %q, want n@v", got)
	}

	if MCPSourceItem(mcpapiv0.ServerResponse{}).NameVersion() != "" {
		t.Error("empty MCP item should return empty NameVersion")
	}
}

func TestSourceItem_NameVersion_A2A(t *testing.T) {
	t.Parallel()

	st, err := structpb.NewStruct(map[string]any{"name": "agent", "version": "2.0.0"})
	if err != nil {
		t.Fatal(err)
	}

	s := A2ASourceItem(st)
	if got := s.NameVersion(); got != "agent@2.0.0" {
		t.Errorf("NameVersion = %q, want agent@2.0.0", got)
	}

	st2, err := structpb.NewStruct(map[string]any{"name": "onlyname"})
	if err != nil {
		t.Fatal(err)
	}

	s2 := A2ASourceItem(st2)
	if got := s2.NameVersion(); got != "onlyname@v1.0.0" {
		t.Errorf("default version: got %q, want onlyname@v1.0.0", got)
	}

	if A2ASourceItem(nil).NameVersion() != "" {
		t.Error("nil struct should give empty NameVersion")
	}
}

func TestSourceItem_NameVersion_AgentSkill(t *testing.T) {
	t.Parallel()

	st, err := structpb.NewStruct(map[string]any{
		"name": "my-skill",
		"metadata": map[string]any{
			"version": "3.0.0",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	s := AgentSkillSourceItem(st)
	if got := s.NameVersion(); got != "my-skill@3.0.0" {
		t.Errorf("NameVersion = %q, want my-skill@3.0.0", got)
	}

	st2, err := structpb.NewStruct(map[string]any{"name": "only"})
	if err != nil {
		t.Fatal(err)
	}

	if got := AgentSkillSourceItem(st2).NameVersion(); got != "only@v1.0.0" {
		t.Errorf("default version: got %q", got)
	}
}
