// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

// WorkItem represents a publication task that needs to be processed by a worker.
type WorkItem struct {
	// PublicationID is the unique identifier of the publication to process
	PublicationID string
}
