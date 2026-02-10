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

//go:embed record_warnings_080.json
var ExpectedRecordWarningsV080JSON []byte

//go:embed record_100.json
var ExpectedRecordV100JSON []byte

//go:embed expected_cid_list.json
var ExpectedCIDs []byte
