// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	DirProtocol     = "dir"
	DirProtocolCode = 65535
)

// Add dir protocol to the host.
//
//nolint:mnd
func init() {
	err := ma.AddProtocol(ma.Protocol{
		Name:  DirProtocol,
		Code:  DirProtocolCode,
		VCode: ma.CodeToVarint(DirProtocolCode),
		Size:  ma.LengthPrefixedVarSize,
		Transcoder: ma.NewTranscoderFromFunctions(
			// String to bytes encoder
			func(s string) ([]byte, error) {
				return []byte(s), nil
			},
			// Bytes to string decoder
			func(b []byte) (string, error) {
				return string(b), nil
			},
			// Validator (optional)
			nil,
		),
	})
	if err != nil {
		panic(fmt.Errorf("failed to add dir protocol: %w", err))
	}
}

// newHost creates a new host libp2p host.
func newHost(listenAddr, dirAPIAddr string, key crypto.PrivKey) (host.Host, error) {
	// Select connection manager
	// connMgr, err := connmgr.NewConnManager(
	// 	100, //nolint:mnd
	// 	400, //nolint:mnd
	// 	connmgr.WithGracePeriod(time.Minute),
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create p2p host connection manager: %w", err)
	// }
	// Create host
	host, err := libp2p.New(
		// Add directory API address to the host address factory
		libp2p.AddrsFactory(
			func(addrs []ma.Multiaddr) []ma.Multiaddr {
				// Only add the dir address if dirAPIAddr is not empty
				if dirAPIAddr != "" {
					dirAddr := ma.StringCast("/dir/" + dirAPIAddr)

					return append(addrs, dirAddr)
				}

				return addrs
			},
		),
		// Use the keypair we generated
		libp2p.Identity(key),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(listenAddr),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// support any other default multiplexer
		libp2p.DefaultMuxers,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		// libp2p.ConnectionManager(connMgr),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// If you want to help other peers to figure out if they are behind
		// NATs, you can launch the server-side of AutoNAT too (AutoRelay
		// already runs the client)
		//
		// This service is highly rate-limited and should not cause any
		// performance issues.
		libp2p.EnableNATService(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host: %w", err)
	}

	return host, nil
}
