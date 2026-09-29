// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"sync"

	"github.com/agntcy/dir/client"
)

// clientRegistry holds live *client.Client instances keyed by an opaque
// int64 handle that is returned to C callers from NewClient. The C ABI has
// no notion of a Go pointer, so every handle-based exported function takes
// this handle as its first argument and looks the client up here.
var (
	clientRegistry   = map[int64]*client.Client{}
	clientRegistryMu sync.Mutex
	nextHandle       int64
)

// registerClient stores c under a freshly allocated handle and returns it.
func registerClient(c *client.Client) int64 {
	clientRegistryMu.Lock()
	defer clientRegistryMu.Unlock()

	nextHandle++
	handle := nextHandle
	clientRegistry[handle] = c

	return handle
}

// lookupClient returns the client registered under handle, if any.
func lookupClient(handle int64) (*client.Client, bool) {
	clientRegistryMu.Lock()
	defer clientRegistryMu.Unlock()

	c, ok := clientRegistry[handle]

	return c, ok
}

// unregisterClient removes handle from the registry. It is a no-op if the
// handle is not present.
func unregisterClient(handle int64) {
	clientRegistryMu.Lock()
	defer clientRegistryMu.Unlock()

	delete(clientRegistry, handle)
}
