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

//go:embed record_110.json
var ExpectedRecordV110JSON []byte

//go:embed record_070_name_resolution.json
var ExpectedRecordV070NameResolutionJSON []byte

// SkillMarkdown is a sample SKILL.md following the Agent Skills format
// (https://agentskills.io/specification).
// Structure inspired by https://github.com/anthropics/skills (Apache-2.0); content is original.
//
//go:embed code-review/SKILL.md
var SkillMarkdown []byte

// SkillRecordJSON is an OASF record containing a core/language_model/agentskills
// module, generated from code-review/SKILL.md via the oasf-sdk translator.
//
//go:embed skill_record.json
var SkillRecordJSON []byte

// DirectoryRecordJSON is an OASF record for the agntcy Directory service.
//
//go:embed directory-record.json
var DirectoryRecordJSON []byte
