// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package v1 is a Go-only compatibility shim; it has no wire format and no
// service. The naming.v1 service was removed in favour of identity.v1, but the
// pinned github.com/agntcy/dir-mcp (embedded by `dirctl mcp serve`) still
// compiles against these result types through Client.GetVerificationInfo and
// Client.GetVerificationInfoByName. They are filled from the record's
// ownership claim.
//
// Remove this package together with `dirctl mcp serve`, or once dir-mcp reads
// identity.v1 directly.
package v1

import "google.golang.org/protobuf/types/known/timestamppb"

// GetVerificationInfoResponse is the result of a name ownership lookup.
type GetVerificationInfoResponse struct {
	Verified     bool
	Verification *Verification
	ErrorMessage string
}

func (r *GetVerificationInfoResponse) GetVerified() bool {
	return r != nil && r.Verified
}

func (r *GetVerificationInfoResponse) GetVerification() *Verification {
	if r == nil {
		return nil
	}

	return r.Verification
}

func (r *GetVerificationInfoResponse) GetErrorMessage() string {
	if r == nil {
		return ""
	}

	return r.ErrorMessage
}

// Verification holds the details of a verified name.
type Verification struct {
	Domain *DomainVerification
}

func (v *Verification) GetDomain() *DomainVerification {
	if v == nil {
		return nil
	}

	return v.Domain
}

// DomainVerification describes the verified owner of a record's name.
type DomainVerification struct {
	// Domain is the verified owner subject.
	Domain string
	// Method is how the owner was verified.
	Method string
	// KeyId is the matched key, which an ownership claim does not carry.
	KeyId string
	// VerifiedAt is when the owner was last verified.
	VerifiedAt *timestamppb.Timestamp
}

func (d *DomainVerification) GetDomain() string {
	if d == nil {
		return ""
	}

	return d.Domain
}

func (d *DomainVerification) GetMethod() string {
	if d == nil {
		return ""
	}

	return d.Method
}

func (d *DomainVerification) GetKeyId() string {
	if d == nil {
		return ""
	}

	return d.KeyId
}

func (d *DomainVerification) GetVerifiedAt() *timestamppb.Timestamp {
	if d == nil {
		return nil
	}

	return d.VerifiedAt
}
