// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package skill builds and publishes an OASF record describing how an AI agent
// can interact with this Directory instance. The embedded SKILL.md is carried
// inline at modules[0].artifact.data (base64) so a single Pull returns
// everything needed to consume the skill.
package skill

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/version"
)

//go:embed SKILL.md
var markdown string

const (
	// RecordName is the OASF record name consumers search for.
	RecordName = "agntcy.dir/skill"

	// SchemaVersion is the OASF schema version emitted.
	SchemaVersion = "1.0.0"

	// MediaType is the IANA media type of the embedded payload.
	MediaType = "text/markdown"

	// SkillFileName is the value placed in agentskills.data.skill_file (a path,
	// not the body).
	SkillFileName = "SKILL.md"

	agentskillsModuleName = "core/language_model/agentskills"
	agentskillsModuleID   = 10302

	// OASF requires at least one entry in `skills`. This taxonomy node is the
	// closest fit for a documentation/instructional record.
	skillContextualComprehensionID = 10101

	// fallbackVersion is used when the binary was built without a version
	// stamped in via ldflags (e.g. `go run`).
	fallbackVersion = "0.0.0-dev"
)

// Markdown returns the embedded SKILL.md content.
func Markdown() string { return markdown }

// ContentSHA256 returns the lowercase hex sha256 of the embedded markdown.
func ContentSHA256() string {
	sum := sha256.Sum256([]byte(markdown))

	return hex.EncodeToString(sum[:])
}

// RecordVersion returns the OASF record's `version` field, mirroring the DIR
// build version so each release publishes a distinct revision.
func RecordVersion() string {
	if version.Version != "" {
		return version.Version
	}

	return fallbackVersion
}

// BuildRecord constructs the OASF record. `now` is injected for deterministic
// testing; production callers should pass time.Now().UTC().
func BuildRecord(now time.Time) (*corev1.Record, error) {
	recordVersion := RecordVersion()
	contentBytes := []byte(markdown)

	digest := "sha256:" + ContentSHA256()
	encodedPayload := base64.StdEncoding.EncodeToString(contentBytes)

	doc := map[string]any{
		"name":           RecordName,
		"schema_version": SchemaVersion,
		"version":        recordVersion,
		"description":    "Self-describing usage guide for this AGNTCY Directory (DIR) instance: how an AI agent can publish, search, pull, sign, and verify OASF agent records.",
		"authors":        []string{"AGNTCY Contributors"},
		"created_at":     now.UTC().Format(time.RFC3339),
		// `domains` is Recommended, not Required; emit empty to silence the
		// validator warning without binding to a specific domain.
		"domains": []any{},
		"skills": []map[string]any{
			{
				"name": "natural_language_processing/natural_language_understanding/contextual_comprehension",
				"id":   skillContextualComprehensionID,
			},
		},
		"modules": []map[string]any{
			{
				"name": agentskillsModuleName,
				"id":   agentskillsModuleID,
				"data": map[string]any{
					"skill_file": SkillFileName,
					"skill_manifest": map[string]any{
						"name":        "agntcy-dir",
						"description": "Use this skill to interact with an AGNTCY Directory (DIR) instance.",
						"version":     recordVersion,
						"frontmatter_metadata": map[string]any{
							"author": "AGNTCY Contributors",
						},
					},
				},
				// agentskills.artifact is the OASF descriptor slot for inline
				// module payloads. descriptor.data is bytestring_t, so the
				// markdown is base64-encoded.
				"artifact": map[string]any{
					"media_type": MediaType,
					"size":       len(contentBytes),
					"digest":     digest,
					"data":       encodedPayload,
				},
			},
		},
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshal skill record: %w", err)
	}

	record, err := corev1.UnmarshalRecord(raw)
	if err != nil {
		return nil, fmt.Errorf("decode skill record: %w", err)
	}

	return record, nil
}
