// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sessionstore

type SessionStore interface {
	GetHubSession(string) (*HubSession, error)
	SaveHubSession(string, *HubSession) error
	RemoveSession(string) error
}
