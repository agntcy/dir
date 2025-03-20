// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type APIRegistrer func(host.Host) error

type options struct {
	Key             crypto.PrivKey
	ListenAddress   string
	BootstrapPeers  []peer.AddrInfo
	RefreshInterval time.Duration
	Randevous       string
	APIRegistrer    APIRegistrer
}

type Option func(*options) error

func WithRandevous(randevous string) Option {
	return func(opts *options) error {
		opts.Randevous = randevous

		return nil
	}
}

func WithIdentityKey(key crypto.PrivKey) Option {
	return func(opts *options) error {
		opts.Key = key

		return nil
	}
}

func WithListenAddress(addr string) Option {
	return func(opts *options) error {
		opts.ListenAddress = addr

		return nil
	}
}

func WithBootstrapAddrs(addrs []string) Option {
	return func(opts *options) error {
		peerInfos := make([]peer.AddrInfo, len(addrs))

		for i, addr := range addrs {
			peerinfo, err := peer.AddrInfoFromString(addr)
			if err != nil {
				return fmt.Errorf("invalid bootstrap addr: %w", err)
			}

			peerInfos[i] = *peerinfo
		}

		opts.BootstrapPeers = peerInfos

		return nil
	}
}

func WithBootstrapPeers(peers []peer.AddrInfo) Option {
	return func(opts *options) error {
		opts.BootstrapPeers = peers

		return nil
	}
}

func WithRefreshInterval(period time.Duration) Option {
	return func(opts *options) error {
		opts.RefreshInterval = period

		return nil
	}
}

// API can only be registreded for non-bootstrap nodes.
func WithAPIRegistrer(reg APIRegistrer) Option {
	return func(opts *options) error {
		opts.APIRegistrer = reg

		return nil
	}
}

func withRandomIdentity() Option {
	return func(opts *options) error {
		if opts.Key != nil {
			return nil
		}

		// Generate random key
		generatedKey, _, err := crypto.GenerateKeyPairWithReader(
			crypto.Ed25519, // Select your key type. Ed25519 are nice short
			-1,             // Select key length when possible (i.e. RSA).
			rand.Reader,    // Always generate a random ID
		)
		if err != nil {
			return fmt.Errorf("failed to create identity key: %w", err)
		}

		// set key
		opts.Key = generatedKey

		return nil
	}
}
