// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

// SignatureVerificationObject represents one signer verification result.
//
//nolint:interfacebloat
type SignatureVerificationObject interface {
	GetRecordCID() string
	GetSignerKey() string
	GetStatus() string
	GetErrorMessage() string
	GetSignerType() string
	GetSignerIssuer() string
	GetSignerSubject() string
	GetSignerCertificateIssuer() string
	GetSignerPublicKey() string
	GetSignerAlgorithm() string
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}
