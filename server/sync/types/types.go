// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

// WorkItem represents a sync task to be processed by workers.
type WorkItem struct {
	SyncID             string
	RemoteDirectoryURL string
}
