// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"encoding/json"
	"fmt"
)

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

// Marshal marshals the RecordReferrer similar to how Record.Marshal marshals Record.
func (r *RecordReferrer) Marshal() ([]byte, error) {
	if r == nil {
		return nil, nil
	}

	jsonBytes, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal RecordReferrer: %w", err)
	}

	var normalized any
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, fmt.Errorf("failed to normalize JSON for canonical ordering: %w", err)
	}

	canonicalBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized JSON with sorted keys: %w", err)
	}

	return canonicalBytes, nil
}
