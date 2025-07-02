// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type SyncObject interface {
	GetID() string
	GetRemoteDirectoryURL() string
	GetStatus() string
}
