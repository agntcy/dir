// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

func TestDropPrivateAddrs(t *testing.T) {
	keep := []string{
		"/ip4/1.1.1.1/tcp/8999",
		"/ip6/2001:4860:4860::8888/tcp/8999",
		"/dns4/dir.example.com/tcp/8999",
		"/" + DirProtocol + "/dir.example.com:8888",
		"/" + OciProtocol + "/" + EncodeAppAddr("ghcr.io/org/repo"),
	}
	drop := []string{
		"/ip4/10.0.0.1/tcp/8999",
		"/ip4/172.16.0.1/tcp/8999",
		"/ip4/192.168.1.1/tcp/8999",
		"/ip4/127.0.0.1/tcp/8999",
		"/ip4/0.0.0.0/tcp/8999",
		"/ip6/::1/tcp/8999",
		"/ip6/fe80::1/tcp/8999",
	}

	var in []ma.Multiaddr
	for _, s := range append(append([]string{}, keep...), drop...) {
		in = append(in, ma.StringCast(s))
	}

	out := dropPrivateAddrs(in)
	if len(out) != len(keep) {
		t.Fatalf("got %d addrs, want %d: %v", len(out), len(keep), out)
	}

	for i, a := range out {
		if a.String() != keep[i] {
			t.Errorf("addr %d: got %s, want %s", i, a, keep[i])
		}
	}
}
