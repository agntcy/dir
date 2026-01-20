// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

// WorkItem represents a verification task to be processed by workers.
type WorkItem struct {
	RecordCID       string
	Name            string // The record name (with protocol prefix)
	PublicKeyDigest string // Digest of the public key referrer for direct retrieval
}
