// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"golang.org/x/crypto/ssh"
)

type APIRegistrer func(host.Host) error

type options struct {
	Key                 crypto.PrivKey
	ListenAddress       string
	DirectoryAPIAddress string
	BootstrapPeers      []peer.AddrInfo
	RefreshInterval     time.Duration
	Randevous           string
	APIRegistrer        APIRegistrer
	ProviderStore       providers.ProviderStore
	DHTCustomOpts       func(host.Host) ([]dht.Option, error)
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

func WithIdentityKeyPath(keyPath string) Option {
	return func(opts *options) error {
		// If path is not set, skip
		if keyPath == "" {
			return nil
		}

		// Read data
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return fmt.Errorf("failed to read key: %w", err)
		}

		// Parse the private key
		key, err := ssh.ParseRawPrivateKey(keyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}

		// Try to convert to ED25519 private key
		ed25519Key, ok := key.(ed25519.PrivateKey)
		if !ok {
			return errors.New("key is not an ED25519 private key")
		}

		// Generate random key
		generatedKey, err := crypto.UnmarshalEd25519PrivateKey(ed25519Key)
		if err != nil {
			return fmt.Errorf("failed to unmarshal identity key: %w", err)
		}

		// set key
		opts.Key = generatedKey

		return nil
	}
}

func WithListenAddress(addr string) Option {
	return func(opts *options) error {
		opts.ListenAddress = addr

		return nil
	}
}

func WithDirectoryAPIAddress(addr string) Option {
	return func(opts *options) error {
		opts.DirectoryAPIAddress = addr

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

// WithCustomDHTOpts sets custom config for DHT.
// NOTE: this is app-specific, be careful when using!
func WithCustomDHTOpts(dhtOptFactory func(host.Host) ([]dht.Option, error)) Option {
	return func(opts *options) error {
		opts.DHTCustomOpts = dhtOptFactory

		return nil
	}
}

func withRandomIdentity() Option {
	return func(opts *options) error {
		// Do not generate random identity if we already have the key
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
