// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"encoding/json"
	"fmt"
)

type Payload struct {
	Critical Critical `json:"critical"`
}

type Critical struct {
	Image Image `json:"image"`
}

type Image struct {
	DockerManifestDigest string `json:"docker-manifest-digest"`
}

func GeneratePayload(digest string) ([]byte, error) {
	payload := &Payload{
		Critical: Critical{
			Image: Image{
				DockerManifestDigest: digest,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return payloadBytes, nil
}
