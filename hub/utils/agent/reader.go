// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	corev1alpha1 "github.com/agntcy/dirhub/backport/api/core/v1alpha1"
	ocidigest "github.com/opencontainers/go-digest"
)

func GetReader(fpath string, fromStdin bool) (io.ReadCloser, error) {
	if fpath == "" && !fromStdin {
		return nil, errors.New("if no path defined --stdin flag must be set")
	}

	if fpath != "" {
		file, err := os.Open(fpath)
		if err != nil {
			return nil, fmt.Errorf("could not open file %s: %w", fpath, err)
		}

		return file, nil
	}

	return os.Stdin, nil
}

// Marshal marshals the Record using canonical JSON serialization.
// This ensures deterministic, cross-language compatible byte representation.
// The output represents the pure Record data and is used for both CID calculation and storage.
func MarshalCanonical(agent *corev1alpha1.Agent) ([]byte, error) {
	jsonBytes, err := json.Marshal(agent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Record: %w", err)
	}

	var normalized interface{}
	if err := json.Unmarshal(jsonBytes, &normalized); err != nil {
		return nil, fmt.Errorf("failed to normalize JSON for canonical ordering: %w", err)
	}

	canonicalBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal normalized JSON with sorted keys: %w", err)
	}

	return canonicalBytes, nil
}

func GetDigest(agent *corev1alpha1.Agent) (ocidigest.Digest, error) {
	canonicalBytes, err := MarshalCanonical(agent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Record canonically: %w", err)
	}

	hash := ocidigest.SHA256.FromBytes(canonicalBytes)
	return hash, nil
}
