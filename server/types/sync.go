// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"

type SyncObject interface {
	GetID() string
	GetRemoteDirectoryURL() string
	GetStatus() storev1alpha2.SyncStatus
}
