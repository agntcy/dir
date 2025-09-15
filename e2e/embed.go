// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed testdata/record_v031.json
var expectedRecordV031JSON []byte

//go:embed testdata/record_v070.json
var expectedRecordV070JSON []byte

//go:embed testdata/record_v070_sync.json
var expectedRecordV070SyncJSON []byte
