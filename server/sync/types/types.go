// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

// WorkItem represents a sync task to be processed by workers.
type WorkItem struct {
	Type               WorkItemType
	SyncID             string
	RemoteDirectoryURL string
	CIDs               []string
}

// WorkItemType represents the type of sync task.
type WorkItemType string

const (
	WorkItemTypeSyncCreate WorkItemType = "sync-create"
	WorkItemTypeSyncDelete WorkItemType = "sync-delete"
)
