// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"path/filepath"
	"strings"
)

// datastoreVersionDir returns the directory this protocol version keeps its
// routing datastore in, a subdirectory of the configured one.
//
// Badger takes an exclusive lock on its directory and the two versions disagree
// about what the keys in it mean, so a v1 and a v2 node must never be pointed at
// the same one. Versioning the path is what lets an upgrade happen in place: v2
// starts in an empty directory of its own while the v1 files stay readable
// beside it, which is what the publish-state migration needs.
//
// The name is derived from the protocol prefix rather than configured, so a
// future protocol bump cannot forget to bring the storage layout with it. The
// whole prefix is used, not just its last segment: v1's prefix was "dir", with
// no version segment at all, so anything that assumes a trailing version breaks
// on values this constant has actually held.
func datastoreVersionDir(base string) string {
	return filepath.Join(base, protocolDirName(ProtocolPrefix))
}

// protocolDirName flattens a protocol prefix into a single path segment:
// "/dir/2" becomes "dir-2".
func protocolDirName(prefix string) string {
	return strings.ReplaceAll(strings.Trim(prefix, "/"), "/", "-")
}
