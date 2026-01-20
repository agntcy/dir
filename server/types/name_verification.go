// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// NameVerificationObject represents a name verification result.
type NameVerificationObject interface {
	GetRecordCID() string
	GetMethod() string
	GetKeyID() string
	GetStatus() string
	GetError() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}
