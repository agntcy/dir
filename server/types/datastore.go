// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/ipfs/go-datastore"
)

// Key provides a unified path-based interface for all objects.
// All objects (and their properties) in the system are identified by a key.
// Note: This key schema MUST be used for all object, including storage, routing,
// and any other internal data to allow interoperability between different services.
//
// TODO: define key logic for transformation and querying between services
// TODO: expand interface if needed with Dir-specific data
//
// Ref: https://ipld.io/
// Ref: https://github.com/ipfs/go-datastore/blob/d16c26966647697a53ec76d918ec9bb41e32a754/key.go#L32
type Key interface {
	String() string
	Namespace() string
	Path() []string
	Validate() error
}

// Datastore is a local key-value store with path-like query syntax.
// Used as a local cache for information such as
// peers, contents, cache, and storage metadata.
//
// Backends: Badger, BoltDB, LevelDB, Mem, Map, etc.
// Providers: Filesystem (local/remote), OCI (remote sync), S3 (remote sync).
//
// NOTE: This is an internal interface to serve internal and external APIs.
//
// Ref: https://github.com/ipfs/go-datastore
type Datastore interface {
	datastore.Batching
}
