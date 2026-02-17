// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import storev1 "github.com/agntcy/dir/api/store/v1"

type SyncObject interface {
	GetID() string
	GetRemoteDirectoryURL() string
	GetCIDs() []string
	GetStatus() storev1.SyncStatus
}
