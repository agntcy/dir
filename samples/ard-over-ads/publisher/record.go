// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package publisher

import (
	"encoding/base64"

	oasftypesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	adscorev1 "github.com/agntcy/dir/api/core/v1"
	ocidigest "github.com/opencontainers/go-digest"
	"google.golang.org/protobuf/types/known/structpb"
)

// AgentSkill is a sample agent skill for testing purposes.
var AgentSkill = `This is a sample agent skill for testing purposes.`

// AgentRecord is a sample OASF record for testing purposes.
var AgentRecord = adscorev1.New(&oasftypesv1.Record{
	Name:          "test-ad",
	Version:       "testing.v1.0.0",
	SchemaVersion: "1.0.0",
	Description:   "This is a test ad for the ADS system.",
	CreatedAt:     "2024-06-01T12:00:00Z",
	Authors:       []string{"Alice"},
	// Skills from: https://schema.oasf.outshift.com/1.0.0/skills/image_segmentation
	Skills: []*oasftypesv1.Skill{
		{Name: "images_computer_vision/image_segmentation"},
	},
	Modules: []*oasftypesv1.Module{
		// Module from: https://schema.oasf.outshift.com/1.0.0/modules/agentskills
		{
			Name: "core/language_model/agentskills",
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"skill_file": structpb.NewStringValue("SKILL.md"),
					"skill_manifest": structpb.NewStructValue(&structpb.Struct{
						Fields: map[string]*structpb.Value{
							"name":        structpb.NewStringValue("sample"),
							"description": structpb.NewStringValue("This is a sample skill manifest for testing purposes."),
							"version":     structpb.NewStringValue("1.0.0"),
						},
					}),
				},
			},
			Artifact: &oasftypesv1.Descriptor{
				MediaType: catalogv1.ProtocolAgentSkillsMdMediaType,
				Size:      uint64(len(AgentSkill)),
				Digest:    ocidigest.FromString(AgentSkill).String(),
				Data:      []byte(base64.StdEncoding.EncodeToString([]byte(AgentSkill))),
			},
		},
	},
})
