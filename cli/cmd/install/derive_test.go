// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"os"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func loadRecord(t *testing.T, name string) *corev1.Record {
	t.Helper()

	raw, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)

	var data structpb.Struct
	require.NoError(t, protojson.Unmarshal(raw, &data))

	return &corev1.Record{Data: &data}
}

func TestDeriveSkillOnly(t *testing.T) {
	arts, err := deriveArtifacts(loadRecord(t, "skill.json"))
	require.NoError(t, err)
	require.Equal(t, "code-review", arts.slug)
	require.NotEmpty(t, arts.skill)
	require.Empty(t, arts.mcpServers)
}

func TestDeriveMCPOnly(t *testing.T) {
	arts, err := deriveArtifacts(loadRecord(t, "mcp.json"))
	require.NoError(t, err)
	require.Equal(t, "io.example-code-review-server", arts.slug)
	require.Empty(t, arts.skill)
	require.Len(t, arts.mcpServers, 1)
	require.NotEmpty(t, arts.mcpServers[0].name)
	_, isAnySlice := arts.mcpServers[0].entry["args"].([]any)
	require.True(t, isAnySlice, "args must be []any")

	_, isAnyMap := arts.mcpServers[0].entry["env"].(map[string]any)
	require.True(t, isAnyMap, "env must be map[string]any")
}

func TestDeriveMulti(t *testing.T) {
	arts, err := deriveArtifacts(loadRecord(t, "multi.json"))
	require.NoError(t, err)
	require.NotEmpty(t, arts.skill)
	require.NotEmpty(t, arts.mcpServers)
}

func TestDeriveA2AErrors(t *testing.T) {
	_, err := deriveArtifacts(loadRecord(t, "a2a.json"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "dirctl export")
}

func TestDeriveBareErrors(t *testing.T) {
	_, err := deriveArtifacts(loadRecord(t, "bare.json"))
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "no installable")
}

// --- moduleNames tests ---

func TestModuleNamesNoModulesField(t *testing.T) {
	// A struct with no "modules" key returns ["none"].
	data, err := structpb.NewStruct(map[string]any{"name": "foo"})
	require.NoError(t, err)

	got := moduleNames(data)
	require.Equal(t, []string{"none"}, got)
}

func TestModuleNamesWithModules(t *testing.T) {
	// Build a structpb.Struct that has a modules list with named entries.
	data, err := structpb.NewStruct(map[string]any{
		"modules": []any{
			map[string]any{"name": "skills"},
			map[string]any{"name": "mcp"},
		},
	})
	require.NoError(t, err)

	got := moduleNames(data)
	require.Equal(t, []string{"skills", "mcp"}, got)
}

func TestModuleNamesEmptyModulesList(t *testing.T) {
	// modules list exists but is empty → ["none"].
	data, err := structpb.NewStruct(map[string]any{
		"modules": []any{},
	})
	require.NoError(t, err)

	got := moduleNames(data)
	require.Equal(t, []string{"none"}, got)
}

func TestSanitizeSlug(t *testing.T) {
	require.Equal(t, "io.example-code-review-server", sanitizeSlug("io.example/code-review-server"))
	require.Equal(t, "my-agent", sanitizeSlug("my agent"))

	// Path-traversal hardening: leading/trailing dots are trimmed so a slug
	// cannot be "." or ".." or escape a parent directory when used as a path.
	require.Empty(t, sanitizeSlug(".."))
	require.Empty(t, sanitizeSlug("."))
	require.Equal(t, "etc", sanitizeSlug("../../etc"))
}
