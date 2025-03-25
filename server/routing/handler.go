// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package routing

import (
	"context"
	"log"
	"regexp"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-kad-dht/providers"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
)

var _ providers.ProviderStore = &handler{}

type handler struct {
	*providers.ProviderManager
	hostID   string
	notifyCh chan<- *handlerSync
}

type handlerSync struct {
	Ref  *coretypes.ObjectRef
	Peer peer.AddrInfo
}

func (h *handler) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	if err := h.handleAnnounce(ctx, key, prov); err != nil {
		// log this error only
		log.Printf("Failed to resolve agent: %v", err)
	}

	return h.ProviderManager.AddProvider(ctx, key, prov)
}

func (h *handler) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return h.ProviderManager.GetProviders(ctx, key)
}

// handleAnnounce tries to parse the data from provider in order to update the local routing data
// about the content and peer.
func (h *handler) handleAnnounce(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	// validete if the provider is not the same as the host
	if peer.ID(h.hostID) == prov.ID {
		return nil
	}

	// get ref digest from request
	// if this fails, it may mean that it's not DIR-constructed CID
	cast, err := mh.Cast(key)
	if err != nil {
		return nil
	}

	// create CID from multihash
	// NOTE: we can only get the digest here, but not the type
	// NOTE: we have to reach out to the provider anyway to update data
	ref := &coretypes.ObjectRef{}

	err = ref.FromCID(cid.NewCidV1(cid.Raw, cast))
	if err != nil {
		return nil
	}

	// validate if valid sha256 digest
	if !regexp.MustCompile(`^[a-fA-F0-9]{64}$`).MatchString(strings.TrimPrefix(ref.GetDigest(), "sha256:")) {
		// this is not an object of interest
		return nil
	}

	log.Printf("Peer %s received announcement event for object %s from Peer %s", h.hostID, ref.GetDigest(), prov.ID)

	// notify the channel
	h.notifyCh <- &handlerSync{
		Ref:  ref,
		Peer: prov,
	}

	return nil
}
