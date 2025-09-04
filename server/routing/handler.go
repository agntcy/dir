// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
)

var (
	_             providers.ProviderStore = &handler{}
	handlerLogger                         = logging.Logger("routing/handler")
)

type handler struct {
	*providers.ProviderManager
	hostID   string
	notifyCh chan<- *handlerSync
}

type handlerSync struct {
	Ref  *corev1.RecordRef
	Peer peer.AddrInfo
}

func (h *handler) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	if err := h.handleAnnounce(ctx, key, prov); err != nil {
		// log this error only
		handlerLogger.Error("Failed to handle announce", "error", err)
	}

	if err := h.ProviderManager.AddProvider(ctx, key, prov); err != nil {
		return fmt.Errorf("failed to add provider: %w", err)
	}

	return nil
}

func (h *handler) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	providers, err := h.ProviderManager.GetProviders(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	return providers, nil
}

// handleAnnounce tries to parse the data from provider in order to update the local routing data
// about the content and peer.
// nolint:unparam
func (h *handler) handleAnnounce(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	keyStr := string(key)
	handlerLogger.Debug("Received announcement event", "key", keyStr, "provider", prov)

	// validate if the provider is not the same as the host
	if peer.ID(h.hostID) == prov.ID {
		handlerLogger.Info("Ignoring announcement event from self", "provider", prov)

		return nil
	}

	// All announcements are now treated as CID provider announcements
	// Labels are discovered via pull-based mechanism when content is fetched
	return h.handleCIDProviderAnnouncement(ctx, key, prov)
}

// handleCIDProviderAnnouncement handles CID provider announcements (existing logic).
func (h *handler) handleCIDProviderAnnouncement(_ context.Context, key []byte, prov peer.AddrInfo) error {
	// get ref cid from request
	// if this fails, it may mean that it's not DIR-constructed CID
	cast, err := mh.Cast(key)
	if err != nil {
		handlerLogger.Error("Failed to cast key to multihash", "error", err)

		return nil
	}

	// create CID from multihash
	ref := &corev1.RecordRef{
		Cid: cid.NewCidV1(1, cast).String(),
	}

	// Validate that we have a non-empty CID
	if ref.GetCid() == "" {
		handlerLogger.Info("Ignoring announcement event for empty CID")

		return nil
	}

	handlerLogger.Info("CID provider announcement event", "ref", ref, "provider", prov, "host", h.hostID)

	// notify the channel
	h.notifyCh <- &handlerSync{
		Ref:  ref,
		Peer: prov,
	}

	return nil
}
