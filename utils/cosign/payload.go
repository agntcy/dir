// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

type Payload struct {
	Critical Critical `json:"critical"`
}

type Critical struct {
	Identity Identity `json:"identity"`
	Image    Image    `json:"image"`
}

type Identity struct {
	DockerReference string `json:"docker-reference"`
}

type Image struct {
	DockerManifestDigest string `json:"docker-manifest-digest"`
}

func GeneratePayload(registry, repo, digest string) *Payload {
	return &Payload{
		Critical: Critical{
			Identity: Identity{
				DockerReference: registry + "/" + repo,
			},
			Image: Image{
				DockerManifestDigest: digest,
			},
		},
	}
}
