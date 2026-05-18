// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"
)

const bootstrapResultsPerPeer = 2

type bootstrapPeerValidation struct {
	entries []bootstrapPeerEntry
}

type bootstrapPeerEntry struct {
	index   int
	address string
	result  checkResult
	info    *peer.AddrInfo
}

func validateBootstrapPeers(peerAddrs []string) bootstrapPeerValidation {
	entries := make([]bootstrapPeerEntry, 0, len(peerAddrs))
	for i, peerAddr := range peerAddrs {
		result, peerInfo := bootstrapPeerMultiaddr(i, peerAddr)
		entries = append(entries, bootstrapPeerEntry{
			index:   i,
			address: peerAddr,
			result:  result,
			info:    peerInfo,
		})
	}

	return bootstrapPeerValidation{entries: entries}
}

func (v bootstrapPeerValidation) hasPeers() bool {
	return len(v.entries) > 0
}

func (v bootstrapPeerValidation) invalidEntry() (bootstrapPeerEntry, bool) {
	for _, entry := range v.entries {
		if entry.result.Status != statusPass {
			return entry, true
		}
	}

	return bootstrapPeerEntry{}, false
}

func (v bootstrapPeerValidation) peerInfos() []peer.AddrInfo {
	peerInfos := make([]peer.AddrInfo, 0, len(v.entries))
	for _, entry := range v.entries {
		if entry.info != nil {
			peerInfos = append(peerInfos, *entry.info)
		}
	}

	return peerInfos
}

func bootstrapPeerChecks(ctx context.Context, validation bootstrapPeerValidation, timeout time.Duration) []checkResult {
	if !validation.hasPeers() {
		return []checkResult{
			{
				Name:    "bootstrap_peer_multiaddr",
				Status:  statusSkip,
				Message: "No bootstrap peers configured",
			},
			{
				Name:    "bootstrap_peer_dial",
				Status:  statusSkip,
				Message: "No bootstrap peers configured",
			},
		}
	}

	results := make([]checkResult, 0, len(validation.entries)*bootstrapResultsPerPeer)
	for _, entry := range validation.entries {
		results = append(results, entry.result)
		if entry.result.Status != statusPass {
			results = append(results, skipped("bootstrap_peer_dial", "Skipped because bootstrap peer multiaddr validation failed", map[string]string{
				"address": entry.address,
				"index":   fmt.Sprintf("%d", entry.index),
			}))

			continue
		}

		results = append(results, bootstrapPeerDial(ctx, entry.index, entry.address, entry.info, timeout))
	}

	return results
}

func bootstrapPeerMultiaddr(index int, peerAddr string) (checkResult, *peer.AddrInfo) {
	details := map[string]string{
		"address": peerAddr,
		"index":   fmt.Sprintf("%d", index),
	}

	addr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		details["error"] = err.Error()

		return checkResult{
			Name:    "bootstrap_peer_multiaddr",
			Status:  statusFail,
			Message: "Bootstrap peer multiaddr is invalid",
			Details: details,
		}, nil
	}

	peerID, err := addr.ValueForProtocol(multiaddr.P_P2P)
	if err != nil {
		details["error"] = err.Error()

		return checkResult{
			Name:    "bootstrap_peer_multiaddr",
			Status:  statusFail,
			Message: "Bootstrap peer multiaddr is missing /p2p/<peer-id>",
			Details: details,
		}, nil
	}

	details["peer_id"] = peerID

	peerInfo, err := peer.AddrInfoFromString(peerAddr)
	if err != nil {
		details["error"] = err.Error()

		return checkResult{
			Name:    "bootstrap_peer_multiaddr",
			Status:  statusFail,
			Message: "Bootstrap peer multiaddr cannot be converted to peer address info",
			Details: details,
		}, nil
	}

	return checkResult{
		Name:    "bootstrap_peer_multiaddr",
		Status:  statusPass,
		Message: "Bootstrap peer multiaddr is valid",
		Details: details,
	}, peerInfo
}

func bootstrapPeerDial(ctx context.Context, index int, peerAddr string, peerInfo *peer.AddrInfo, timeout time.Duration) checkResult {
	details := map[string]string{
		"address": peerAddr,
		"index":   fmt.Sprintf("%d", index),
		"peer_id": peerInfo.ID.String(),
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	host, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		elapsed := time.Since(start)
		details["error"] = err.Error()

		return checkResult{
			Name:    "bootstrap_peer_dial",
			Status:  statusFail,
			Message: "Failed to create temporary libp2p host",
			Elapsed: elapsed.String(),
			Details: details,
		}
	}

	if err := host.Connect(checkCtx, *peerInfo); err != nil {
		elapsed := time.Since(start)
		details["error"] = err.Error()
		details["connectedness"] = host.Network().Connectedness(peerInfo.ID).String()

		details["host_connected_peer_count"] = fmt.Sprintf("%d", len(host.Network().Peers()))
		if closeErr := host.Close(); closeErr != nil {
			details["close_error"] = closeErr.Error()
		}

		return checkResult{
			Name:    "bootstrap_peer_dial",
			Status:  statusFail,
			Message: "Failed to connect to bootstrap peer",
			Elapsed: elapsed.String(),
			Details: details,
		}
	}

	elapsed := time.Since(start)
	details["connectedness"] = host.Network().Connectedness(peerInfo.ID).String()
	details["host_connected_peer_count"] = fmt.Sprintf("%d", len(host.Network().Peers()))
	protocols, protocolErr := host.Peerstore().GetProtocols(peerInfo.ID)
	addPeerProtocolDetails(details, protocols, protocolErr)

	if closeErr := host.Close(); closeErr != nil {
		details["close_error"] = closeErr.Error()

		return checkResult{
			Name:    "bootstrap_peer_dial",
			Status:  statusWarn,
			Message: "Bootstrap peer accepted libp2p connection, but host cleanup reported an error",
			Elapsed: elapsed.String(),
			Details: details,
		}
	}

	return checkResult{
		Name:    "bootstrap_peer_dial",
		Status:  statusPass,
		Message: "Bootstrap peer accepted libp2p connection and exposed peer metadata",
		Elapsed: elapsed.String(),
		Details: details,
	}
}

func addPeerProtocolDetails(details map[string]string, protocols []protocol.ID, err error) {
	if err != nil {
		details["protocol_error"] = err.Error()

		return
	}

	if len(protocols) == 0 {
		details["protocol_count"] = "0"

		return
	}

	protocolStrings := make([]string, 0, len(protocols))
	for _, proto := range protocols {
		protocolStrings = append(protocolStrings, string(proto))
	}

	details["protocol_count"] = fmt.Sprintf("%d", len(protocolStrings))
	details["protocols"] = strings.Join(protocolStrings, ",")
	details["has_kad_dht_protocol"] = fmt.Sprintf("%t", hasProtocolPrefix(protocolStrings, "/ipfs/kad") || hasProtocolPrefix(protocolStrings, "dir/kad"))
	details["has_dir_rpc_protocol"] = fmt.Sprintf("%t", hasProtocolPrefix(protocolStrings, "/dir/rpc"))
	details["has_gossipsub_protocol"] = fmt.Sprintf("%t", hasProtocolPrefix(protocolStrings, "/meshsub") || hasProtocolPrefix(protocolStrings, "/floodsub"))
}

func hasProtocolPrefix(protocols []string, prefix string) bool {
	for _, proto := range protocols {
		if strings.HasPrefix(proto, prefix) {
			return true
		}
	}

	return false
}

func joinPeerIDs(peers []peer.ID) string {
	ids := make([]string, 0, len(peers))
	for _, p := range peers {
		ids = append(ids, p.String())
	}

	return strings.Join(ids, ",")
}
