// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// SignatureVerificationObject represents one signature verification result.
//
//nolint:interfacebloat
type SignatureVerificationObject interface {
	GetRecordCID() string
	GetSignatureDigest() string
	GetStatus() string
	GetErrorMessage() string
	GetSignerType() string
	GetSignerIssuer() string
	GetSignerSubject() string
	GetSignerPublicKey() string
	GetSignerAlgorithm() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}
