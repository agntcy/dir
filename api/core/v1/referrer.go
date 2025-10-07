// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

// ReferrerObject defines an interface for referrer objects that can be
// marshaled and unmarshaled to/from RecordReferrer format.
type ReferrerObject interface {
	// UnmarshalReferrer loads the object from a RecordReferrer.
	UnmarshalReferrer(*RecordReferrer) error

	// MarshalReferrer exports the object into a RecordReferrer.
	MarshalReferrer() (*RecordReferrer, error)

	// ReferrerType returns the type of the referrer.
	// Examples:
	//   - Signature: "agntcy.dir.sign.v1.Signature"
	//   - PublicKey: "agntcy.dir.sign.v1.PublicKey"
	ReferrerType() string
}
