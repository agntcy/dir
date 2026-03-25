// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package shared

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// ExtractNameVersion extracts "name@version" from a record.
func ExtractNameVersion(record *corev1.Record) (string, error) {
	if record == nil || record.GetData() == nil {
		return "", errors.New("record or record data is nil")
	}

	fields := record.GetData().GetFields()
	if fields == nil {
		return "", errors.New("record data fields are nil")
	}

	// Extract name
	nameVal, ok := fields["name"]
	if !ok {
		return "", errors.New("record missing 'name' field")
	}

	name := nameVal.GetStringValue()
	if name == "" {
		return "", errors.New("record 'name' field is empty")
	}

	// Extract version
	versionVal, ok := fields["version"]
	if !ok {
		return "", errors.New("record missing 'version' field")
	}

	version := versionVal.GetStringValue()
	if version == "" {
		return "", errors.New("record 'version' field is empty")
	}

	return fmt.Sprintf("%s@%s", name, version), nil
}
