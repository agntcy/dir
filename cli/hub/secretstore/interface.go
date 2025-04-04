// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package secretstore

type SecretStore interface {
	GetHubSecret(string) (*HubSecret, error)
	SaveHubSecret(string, *HubSecret) error
	RemoveHubSecret(string) error
}
