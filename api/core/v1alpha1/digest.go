// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"errors"
	"fmt"
	"regexp"
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

func (d *Digest) Encode() string {
	return fmt.Sprintf("%s:%s", DigestType_name[int32(d.GetType())], d.GetValue())
}

func (d *Digest) Decode(str string) error {
	// parse and validate digest
	parts := strings.Split(str, ":")
	if len(parts) != 2 { //nolint:mnd
		return errors.New("digest parts not found")
	}

	// validate digest
	digestType := DigestType(DigestType_value[parts[0]])
	if digestType == DigestType_DIGEST_TYPE_SHA256 && !isValidSHA256(parts[1]) {
		return errors.New("invalid SHA-256 hash")
	}

	// update digest
	*d = Digest{
		Type:  digestType,
		Value: parts[1],
	}

	return nil
}

// isValidSHA256 checks if a string is a valid SHA-256 hash.
func isValidSHA256(s string) bool {
	// SHA-256 hash is a 64-character hexadecimal string
	matched, err := regexp.MatchString("^[a-fA-F0-9]{64}$", s)
	if err != nil {
		return false
	}

	return matched
}
