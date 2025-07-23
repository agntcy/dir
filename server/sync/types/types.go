// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "github.com/agntcy/dir/utils/htpasswd"

// WorkItem represents a sync task to be processed by workers.
type WorkItem struct {
	Type               WorkItemType
	SyncID             string
	RemoteDirectoryURL string
	Credentials        *htpasswd.Credentials
}

// WorkItemType represents the type of sync task.
type WorkItemType string

const (
	WorkItemTypeSyncCreate WorkItemType = "sync-create"
	WorkItemTypeSyncDelete WorkItemType = "sync-delete"
)
