// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// NewDomainVerification creates a new Verification with DomainVerification info.
func NewDomainVerification(dv *DomainVerification) *Verification {
	return &Verification{
		Info: &Verification_Domain{
			Domain: dv,
		},
	}
}
