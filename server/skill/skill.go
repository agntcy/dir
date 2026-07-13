// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package skill builds and publishes an OASF record describing how an AI
// agent can interact with this Directory instance.
package skill

import (
	_ "embed"
	"fmt"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/version"
	ocidigest "github.com/opencontainers/go-digest"
	"google.golang.org/protobuf/types/known/structpb"
)

//go:embed SKILL.md
var DirSkill string

const (
	RecordName    = "org.agntcy/directory"
	SchemaVersion = "1.1.0"

	MCPModuleName         = "integration/mcp"
	AgentSkillsModuleName = "core/language_model/agentskills"
	MCPServerName         = "agntcy-dir-mcp"

	// OASF taxonomy IDs (verified against the 1.1.0 JSON schema). Mismatching
	// these is a hard validation error, missing them is only a warning.
	skillContextualComprehensionID uint32 = 10101
	agentSkillsModuleID            uint32 = 10302
	mcpModuleID                    uint32 = 202
	domainSoftwareEngineeringID    uint32 = 102

	// Used when the binary lacks an ldflags-stamped version (e.g. `go run`).
	fallbackVersion = "0.0.0-dev"

	// OASF requires `skills` to be non-empty; pick the closest taxonomy
	// node for an instructional record.
	skillContextualComprehensionName = "language_processing/language_understanding/contextual_comprehension"

	domainSoftwareEngineeringName = "technology/software_engineering"
)

// RecordVersion returns the record's `version`, stripping the commit suffix
// that version.String() appends so it stays semver-shaped.
func RecordVersion() string {
	if version.Version != "" {
		return version.Version
	}

	return fallbackVersion
}

// ContentDigest returns the OCI-form sha256 digest of the embedded SKILL.md.
func ContentDigest() string {
	return ocidigest.FromString(DirSkill).String()
}

// BuildRecord constructs the typed OASF record. `now` is injected so tests
// can pin time; production callers should pass time.Now().UTC().
func BuildRecord(now time.Time) (*corev1.Record, error) {
	rv := RecordVersion()
	skillBytes := []byte(DirSkill)

	skillModule, err := buildAgentSkillsModule(rv, skillBytes)
	if err != nil {
		return nil, err
	}

	mcpModule, err := buildMCPModule()
	if err != nil {
		return nil, err
	}

	return corev1.New(&typesv1.Record{
		Name:          RecordName,
		SchemaVersion: SchemaVersion,
		Version:       rv,
		Description:   "Self-describing usage guide for this AGNTCY Directory (DIR) instance: how an AI agent can publish, search, pull, sign, and verify OASF agent records.",
		Authors:       []string{"https://github.com/agntcy/dir"},
		CreatedAt:     now.UTC().Format(time.RFC3339),
		Skills: []*typesv1.Skill{
			{Name: skillContextualComprehensionName, Id: skillContextualComprehensionID},
		},
		Domains: []*typesv1.Domain{
			{Name: domainSoftwareEngineeringName, Id: domainSoftwareEngineeringID},
		},
		Modules: []*typesv1.Module{skillModule, mcpModule},
	}), nil
}

// Descriptor.Data is proto `bytes`, which the JSON wire layer base64-encodes
// automatically — pass raw bytes, not a pre-encoded string.
func buildAgentSkillsModule(recordVer string, skillBytes []byte) (*typesv1.Module, error) {
	manifest, err := structpb.NewStruct(map[string]any{
		"name":        "agntcy-dir",
		"description": "Use this skill to interact with an AGNTCY Directory (DIR) instance.",
		"version":     recordVer,
		"frontmatter_metadata": map[string]any{
			"author": "AGNTCY Contributors",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build skill_manifest struct: %w", err)
	}

	data, err := structpb.NewStruct(map[string]any{
		"skill_file": "SKILL.md",
	})
	if err != nil {
		return nil, fmt.Errorf("build agentskills data struct: %w", err)
	}

	data.GetFields()["skill_manifest"] = structpb.NewStructValue(manifest)

	return &typesv1.Module{
		Name: AgentSkillsModuleName,
		Id:   agentSkillsModuleID,
		Data: data,
		Artifact: &typesv1.Descriptor{
			MediaType: catalogv1.ProtocolAgentSkillsMdMediaType,
			Size:      uint64(len(skillBytes)),
			Digest:    ocidigest.FromBytes(skillBytes).String(),
			Data:      skillBytes,
		},
	}, nil
}

func buildMCPModule() (*typesv1.Module, error) {
	// `dirctl mcp serve` is the embedded MCP server that fronts this Directory.
	connection, err := structpb.NewStruct(map[string]any{
		"type":    "stdio",
		"command": "dirctl",
		"args":    []any{"mcp", "serve"},
	})
	if err != nil {
		return nil, fmt.Errorf("build mcp connection struct: %w", err)
	}

	data, err := structpb.NewStruct(map[string]any{
		"name":        MCPServerName,
		"description": "MCP server fronting this AGNTCY Directory instance.",
	})
	if err != nil {
		return nil, fmt.Errorf("build mcp data struct: %w", err)
	}

	data.GetFields()["connections"] = structpb.NewListValue(&structpb.ListValue{
		Values: []*structpb.Value{structpb.NewStructValue(connection)},
	})

	return &typesv1.Module{
		Name: MCPModuleName,
		Id:   mcpModuleID,
		Data: data,
	}, nil
}
