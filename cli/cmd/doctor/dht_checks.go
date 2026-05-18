// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"fmt"
	"strings"
	"time"

	serverrouting "github.com/agntcy/dir/server/routing"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

func dhtBootstrap(ctx context.Context, validation bootstrapPeerValidation, timeout time.Duration) checkResult {
	if !validation.hasPeers() {
		return checkResult{
			Name:    "dht_bootstrap_reachability",
			Status:  statusSkip,
			Message: "No bootstrap peers configured",
		}
	}

	if entry, ok := validation.invalidEntry(); ok {
		return skipped("dht_bootstrap_reachability", "Skipped because bootstrap peer multiaddr validation failed", map[string]string{
			"address": entry.address,
			"index":   fmt.Sprintf("%d", entry.index),
		})
	}

	peerInfos := validation.peerInfos()

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()

	libp2pHost, err := libp2p.New(libp2p.NoListenAddrs)
	if err != nil {
		return failedDHTResult("Failed to create temporary libp2p host for DHT bootstrap", err, time.Since(start), nil)
	}

	connectErrors := connectDHTBootstrapPeers(checkCtx, libp2pHost, peerInfos)

	kdht, err := dht.New(checkCtx, libp2pHost,
		dht.Mode(dht.ModeClient),
		dht.ProtocolPrefix(protocol.ID(serverrouting.ProtocolPrefix)),
		dht.BootstrapPeers(peerInfos...),
	)
	if err != nil {
		details := map[string]string{
			"error": err.Error(),
		}
		if closeErr := libp2pHost.Close(); closeErr != nil {
			details["host_close_error"] = closeErr.Error()
		}

		return failedDHTResult("Failed to create temporary DHT client", err, time.Since(start), details)
	}

	if result, ok := bootstrapDHT(checkCtx, kdht, libp2pHost, start); ok {
		return result
	}

	if result, ok := refreshDHTRoutingTable(checkCtx, kdht, libp2pHost, start); ok {
		return result
	}

	return dhtReachabilityResult(checkCtx, kdht, libp2pHost, peerInfos, connectErrors, start)
}

func connectDHTBootstrapPeers(ctx context.Context, libp2pHost host.Host, peerInfos []peer.AddrInfo) []string {
	connectErrors := make([]string, 0)

	for _, peerInfo := range peerInfos {
		if err := libp2pHost.Connect(ctx, peerInfo); err != nil {
			connectErrors = append(connectErrors, fmt.Sprintf("%s: %s", peerInfo.ID, err.Error()))
		}
	}

	return connectErrors
}

func bootstrapDHT(ctx context.Context, kdht *dht.IpfsDHT, libp2pHost host.Host, start time.Time) (checkResult, bool) {
	if err := kdht.Bootstrap(ctx); err != nil {
		details := map[string]string{
			"error": err.Error(),
		}
		addDHTCloseDetails(details, kdht, libp2pHost)

		return failedDHTResult("Failed to bootstrap temporary DHT client", err, time.Since(start), details), true
	}

	return checkResult{}, false
}

func refreshDHTRoutingTable(ctx context.Context, kdht *dht.IpfsDHT, libp2pHost host.Host, start time.Time) (checkResult, bool) {
	refreshDone := kdht.RefreshRoutingTable()
	select {
	case <-refreshDone:
		return checkResult{}, false
	case <-ctx.Done():
		details := map[string]string{
			"error": ctx.Err().Error(),
		}
		addDHTCloseDetails(details, kdht, libp2pHost)

		return failedDHTResult("Timed out while refreshing DHT routing table", ctx.Err(), time.Since(start), details), true
	}
}

func dhtReachabilityResult(ctx context.Context, kdht *dht.IpfsDHT, libp2pHost host.Host, peerInfos []peer.AddrInfo, connectErrors []string, start time.Time) checkResult {
	foundBootstrapPeer, foundBootstrapPeerAddrCount := findFirstDHTBootstrapPeer(ctx, kdht, peerInfos)
	elapsed := time.Since(start)
	routingTable := kdht.RoutingTable()
	routingPeers := routingTable.ListPeers()
	hostConnectedPeerCount := len(libp2pHost.Network().Peers())
	routingTableSize := routingTable.Size()

	details := map[string]string{
		"bootstrap_peer_count":           fmt.Sprintf("%d", len(peerInfos)),
		"find_bootstrap_peer":            fmt.Sprintf("%t", foundBootstrapPeer),
		"find_bootstrap_peer_addr_count": fmt.Sprintf("%d", foundBootstrapPeerAddrCount),
		"host_connected_peer_count":      fmt.Sprintf("%d", hostConnectedPeerCount),
		"protocol_prefix":                serverrouting.ProtocolPrefix,
		"routing_table_size":             fmt.Sprintf("%d", routingTableSize),
		"routing_table_peers":            joinPeerIDs(routingPeers),
	}
	if len(connectErrors) > 0 {
		details["bootstrap_connect_errors"] = strings.Join(connectErrors, "; ")
	}

	if routingTableSize == 0 && !foundBootstrapPeer {
		addDHTCloseDetails(details, kdht, libp2pHost)

		return checkResult{
			Name:    "dht_bootstrap_reachability",
			Status:  statusWarn,
			Message: fmt.Sprintf("Temporary DHT client connected to %d host peer(s), but did not reach the bootstrap peer via DHT", hostConnectedPeerCount),
			Elapsed: elapsed.String(),
			Details: details,
		}
	}

	if addDHTCloseDetails(details, kdht, libp2pHost) {
		return checkResult{
			Name:    "dht_bootstrap_reachability",
			Status:  statusWarn,
			Message: "Temporary DHT client bootstrapped, but cleanup reported an error",
			Elapsed: elapsed.String(),
			Details: details,
		}
	}

	return checkResult{
		Name:    "dht_bootstrap_reachability",
		Status:  statusPass,
		Message: fmt.Sprintf("Temporary DHT client reached bootstrap peer and populated %d DHT routing table peer(s)", routingTableSize),
		Elapsed: elapsed.String(),
		Details: details,
	}
}

func findFirstDHTBootstrapPeer(ctx context.Context, kdht *dht.IpfsDHT, peerInfos []peer.AddrInfo) (bool, int) {
	if len(peerInfos) == 0 {
		return false, 0
	}

	found, err := kdht.FindPeer(ctx, peerInfos[0].ID)
	if err != nil {
		return false, 0
	}

	return true, len(found.Addrs)
}

func failedDHTResult(message string, err error, elapsed time.Duration, details map[string]string) checkResult {
	if details == nil {
		details = map[string]string{
			"error": err.Error(),
		}
	}

	return checkResult{
		Name:    "dht_bootstrap_reachability",
		Status:  statusFail,
		Message: message,
		Elapsed: elapsed.String(),
		Details: details,
	}
}

type closer interface {
	Close() error
}

func addDHTCloseDetails(details map[string]string, kdht closer, host closer) bool {
	hadError := false

	if err := kdht.Close(); err != nil {
		details["dht_close_error"] = err.Error()
		hadError = true
	}

	if err := host.Close(); err != nil {
		details["host_close_error"] = err.Error()
		hadError = true
	}

	return hadError
}
