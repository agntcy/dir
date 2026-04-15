// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package testdata

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed record_070.json
var ExpectedRecordV070JSON []byte

//go:embed record_080_v4.json
var ExpectedRecordV080V4JSON []byte

//go:embed record_080_v5.json
var ExpectedRecordV080V5JSON []byte

//go:embed record_070_sync_v4.json
var ExpectedRecordV070SyncV4JSON []byte

//go:embed record_070_sync_v5.json
var ExpectedRecordV070SyncV5JSON []byte

//go:embed record_100.json
var ExpectedRecordV100JSON []byte

//go:embed record_070_name_resolution.json
var ExpectedRecordV070NameResolutionJSON []byte

// A2AAgentCard is a sample A2A AgentCard JSON following the Agent-to-Agent protocol
// (https://a2a-protocol.org/latest/specification/#441-agentcard).
// Structure inspired by https://github.com/a2aproject/a2a-samples (Apache-2.0); content is original.
//
//go:embed a2a-agent-card.json
var A2AAgentCard []byte

// SkillMarkdown is a sample SKILL.md following the Agent Skills format
// (https://agentskills.io/specification). The on-disk directory at
// code-review/SKILL.md can be used directly with `dirctl import --type=agent-skill`.
// Structure inspired by https://github.com/anthropics/skills (Apache-2.0); content is original.
//
//go:embed code-review/SKILL.md
var SkillMarkdown []byte
