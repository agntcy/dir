// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package routing

import (
	"context"
	"fmt"
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

type notification struct {
	Ref  *coretypes.ObjectRef
	Peer peer.AddrInfo
}

type handler struct {
	*providers.ProviderManager
	hostID   string
	notifyCh chan<- *notification
}

func (h *handler) AddProvider(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	if err := h.resolveAgent(ctx, key, prov); err != nil {
		// log this error only
		log.Printf("Failed to resolve agent: %v", err)
	}

	return h.ProviderManager.AddProvider(ctx, key, prov)
}

func (h *handler) GetProviders(ctx context.Context, key []byte) ([]peer.AddrInfo, error) {
	return h.ProviderManager.GetProviders(ctx, key)
}

// resolveAgent tries to reach out to the provider in order to update the local routing data
// about the content and peer.
func (h *handler) resolveAgent(ctx context.Context, key []byte, prov peer.AddrInfo) error {
	// get ref digest from request
	// if this fails, it may mean that it's not DIR-constructed CID
	cast, err := mh.Cast(key)
	if err != nil {
		return err
	}

	// create CID from multihash
	// NOTE: we can only get the digest here, but not the type
	// NOTE: we have to reach out to the provider anyway to update data
	ref := &coretypes.ObjectRef{}
	ref.FromCID(cid.NewCidV1(cid.Raw, cast))

	// validate if valid sha256 digest
	if !regexp.MustCompile(`^[a-fA-F0-9]{64}$`).MatchString(strings.TrimPrefix(ref.Digest, "sha256:")) {
		return fmt.Errorf("not a digest CID")
	}

	// validete ref digest
	if peer.ID(h.hostID) == prov.ID {
		return fmt.Errorf("self announcement")
	}

	log.Printf("Peer %s: Received announcement event %s from Peer %s", h.hostID, ref, prov.ID)

	// notify the channel
	h.notifyCh <- &notification{
		Ref:  ref,
		Peer: prov,
	}

	return nil
}
