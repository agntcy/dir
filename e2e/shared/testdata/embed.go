// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package testdata

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed record_v031.json
var ExpectedRecordV031JSON []byte

//go:embed record_v070.json
var ExpectedRecordV070JSON []byte

//go:embed record_v070_sync_v4.json
var ExpectedRecordV070SyncV4JSON []byte

//go:embed record_v070_sync_v5.json
var ExpectedRecordV070SyncV5JSON []byte
