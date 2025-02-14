// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"encoding/hex"
	"fmt"
	"strings"
)

func init() {
	// Override allowed names for digest types
	DigestType_name = map[int32]string{
		0: "unspecified",
		1: "sha256",
	}
	DigestType_value = map[string]int32{
		"unspecified": 0,
		"sha256":      1,
	}
}

func (d *Digest) ToString() string {
	return fmt.Sprintf("%s:%x", DigestType_name[int32(d.GetType())], d.GetValue())
}

func (d *Digest) FromString(str string) error {
	// parse and validate digest
	parts := strings.Split(str, ":")
	if len(parts) != 2 {
		return fmt.Errorf("digest parts not found")
	}

	// extract signature
	digestValue, err := hex.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("failed to decode digest: %w", err)
	}

	// update digest
	*d = Digest{
		Type:  DigestType(DigestType_value[parts[0]]),
		Value: digestValue,
	}

	return nil
}
