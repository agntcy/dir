// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/ipfs/go-datastore"
)

// Datastore is a local key-value store with path-like query syntax.
// Used as a local cache for information such as
// peers, contents, cache, and storage metadata.
//
// Backends: Badger, BoltDB, LevelDB, Mem, Map, etc.
// Providers: Filesystem (local/remote), OCI (remote sync), S3 (remote sync).
//
// NOTE: This is an interface to serve internal and external APIs.
//
// Ref: https://github.com/ipfs/go-datastore
type Datastore interface {
	datastore.Batching
}
