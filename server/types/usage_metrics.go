// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

type UsageMetricsObject interface {
	GetRecordCID() string
	GetPullCount() uint64
	GetProviderCount() uint32
	GetLookupCount() uint64
	GetExportCount() uint64
	GetViewCount() uint64
	GetLastUsedAt() *time.Time
}
