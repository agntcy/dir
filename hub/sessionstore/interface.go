// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package sessionstore provides session and token storage for the Agent Hub CLI and related applications.
package sessionstore

// SessionStore defines the interface for session storage backends.
type SessionStore interface {
	// GetHubSession retrieves a session by key.
	GetHubSession(string) (*HubSession, error)
	// SaveHubSession saves a session by key.
	SaveHubSession(string, *HubSession) error
	// RemoveSession deletes a session by key.
	RemoveSession(string) error
}
