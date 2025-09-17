// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

// Network test constants for peer addresses.
const (
	// Local peer addresses used in network deployment tests.
	Peer1Addr = "0.0.0.0:8890"
	Peer2Addr = "0.0.0.0:8891"
	Peer3Addr = "0.0.0.0:8892"

	// Internal Kubernetes service address for peer1.
	Peer1InternalAddr = "agntcy-dir-apiserver.peer1.svc.cluster.local:8888"

	// Test directory prefixes for temporary files.
	NetworkTestDirPrefix = "network-test"
	SignTestDirPrefix    = "sign-test"
)

// PeerAddrs contains all peer addresses for iteration in tests.
var PeerAddrs = []string{Peer1Addr, Peer2Addr, Peer3Addr}
