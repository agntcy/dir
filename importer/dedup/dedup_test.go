// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dedup

import (
	"context"
	"testing"

	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/types"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestFilterDuplicates_SkipsKnownDuplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &DuplicateChecker{
		existingRecords: map[string]string{"dup@1.0.0": "bafycid"},
	}

	in := make(chan types.SourceItem, 2)
	in <- types.MCPSourceItem(mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "dup", Version: "1.0.0"}})

	in <- types.MCPSourceItem(mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "new", Version: "2.0.0"}})

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	var passed int

	for range out {
		passed++
	}

	if passed != 1 {
		t.Errorf("passed through = %d, want 1", passed)
	}

	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}

	// Dedup increments TotalRecords for duplicates; transform would add for non-dupes (not run here).
	if result.TotalRecords != 1 {
		t.Errorf("TotalRecords after dedup = %d, want 1 (duplicate only)", result.TotalRecords)
	}
}

func TestFilterDuplicates_PassThroughWhenUnknown(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &DuplicateChecker{existingRecords: map[string]string{}}

	in := make(chan types.SourceItem, 1)
	in <- types.MCPSourceItem(mcpapiv0.ServerResponse{Server: mcpapiv0.ServerJSON{Name: "only", Version: "1"}})

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	n := 0

	for range out {
		n++
	}

	if n != 1 {
		t.Errorf("count = %d, want 1", n)
	}

	if result.SkippedCount != 0 {
		t.Errorf("SkippedCount = %d", result.SkippedCount)
	}
}

func TestModulesByImportType_MCPTypes(t *testing.T) {
	t.Parallel()

	for _, importType := range []config.ImportType{config.ImportTypeMCPRegistry, config.ImportTypeMCP} {
		modules, ok := modulesByImportType[importType]
		if !ok {
			t.Errorf("import type %q has no entry in modulesByImportType", importType)

			continue
		}

		if len(modules) == 0 {
			t.Errorf("import type %q has empty modules list", importType)
		}

		for _, m := range modules {
			if m != "integration/mcp" && m != "runtime/mcp" {
				t.Errorf("import type %q has unexpected module %q", importType, m)
			}
		}
	}
}

func TestModulesByImportType_A2AType(t *testing.T) {
	t.Parallel()

	modules, ok := modulesByImportType[config.ImportTypeA2A]
	if !ok {
		t.Fatal("ImportTypeA2A has no entry in modulesByImportType")
	}

	expected := []string{"integration/a2a", "runtime/a2a"}
	if len(modules) != len(expected) {
		t.Errorf("ImportTypeA2A modules = %v, want %v", modules, expected)
	} else {
		for i, m := range expected {
			if modules[i] != m {
				t.Errorf("ImportTypeA2A modules[%d] = %q, want %q", i, modules[i], m)
			}
		}
	}
}

func TestModulesByImportType_AgentSkillType(t *testing.T) {
	t.Parallel()

	modules, ok := modulesByImportType[config.ImportTypeAgentSkill]
	if !ok {
		t.Fatal("ImportTypeAgentSkill has no entry in modulesByImportType")
	}

	if len(modules) != 1 || modules[0] != "core/language_model/agentskills" {
		t.Errorf("ImportTypeAgentSkill modules = %v, want [core/language_model/agentskills]", modules)
	}
}

func TestFilterDuplicates_A2ASkipsKnownDuplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &DuplicateChecker{
		importType:      config.ImportTypeA2A,
		existingRecords: map[string]string{"my-agent@v1.0.0": "bafycid123"},
	}

	card, err := structpb.NewStruct(map[string]any{"name": "my-agent", "version": "v1.0.0"})
	if err != nil {
		t.Fatalf("failed to create A2A struct: %v", err)
	}

	newCard, err := structpb.NewStruct(map[string]any{"name": "other-agent", "version": "v2.0.0"})
	if err != nil {
		t.Fatalf("failed to create A2A struct: %v", err)
	}

	in := make(chan types.SourceItem, 2)
	in <- types.A2ASourceItem(card)

	in <- types.A2ASourceItem(newCard)

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	var passed int

	for range out {
		passed++
	}

	if passed != 1 {
		t.Errorf("passed through = %d, want 1", passed)
	}

	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
}

func TestFilterDuplicates_AgentSkillSkipsKnownDuplicate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	result := &types.Result{}

	c := &DuplicateChecker{
		importType:      config.ImportTypeAgentSkill,
		existingRecords: map[string]string{"my-skill@v1.0.0": "bafycid456"},
	}

	skill, err := structpb.NewStruct(map[string]any{
		"name":     "my-skill",
		"metadata": map[string]any{"version": "v1.0.0"},
	})
	if err != nil {
		t.Fatalf("failed to create skill struct: %v", err)
	}

	newSkill, err := structpb.NewStruct(map[string]any{
		"name":     "other-skill",
		"metadata": map[string]any{"version": "v2.0.0"},
	})
	if err != nil {
		t.Fatalf("failed to create skill struct: %v", err)
	}

	in := make(chan types.SourceItem, 2)
	in <- types.AgentSkillSourceItem(skill)

	in <- types.AgentSkillSourceItem(newSkill)

	close(in)

	out := c.FilterDuplicates(ctx, in, result)

	var passed int

	for range out {
		passed++
	}

	if passed != 1 {
		t.Errorf("passed through = %d, want 1", passed)
	}

	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
}
