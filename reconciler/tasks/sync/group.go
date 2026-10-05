// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"slices"
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
)

const (
	dirPrefix = "/dir/"
	ociPrefix = "/oci/"
)

// peerGroup is the CIDs announced by one peer, plus the first /dir/ and /oci/
// addresses seen for that peer. Matches dirctl sync create --stdin.
type peerGroup struct {
	id      string
	dirAddr string
	ociAddr string
	cids    []string
}

func groupByPeer(results []*routingv1.SearchResponse) []*peerGroup {
	byID := make(map[string]*peerGroup)
	order := make([]string, 0)

	for _, result := range results {
		if result == nil || result.GetPeer() == nil || result.GetRecordRef() == nil {
			continue
		}

		cid := result.GetRecordRef().GetCid()
		if cid == "" {
			continue
		}

		peer := result.GetPeer()
		id := peer.GetId()

		group, ok := byID[id]
		if !ok {
			group = &peerGroup{id: id}
			byID[id] = group
			order = append(order, id)
		}

		group.cids = appendUnique(group.cids, cid)

		for _, addr := range peer.GetAddrs() {
			switch {
			case strings.HasPrefix(addr, dirPrefix) && group.dirAddr == "":
				group.dirAddr = strings.TrimPrefix(addr, dirPrefix)
			case strings.HasPrefix(addr, ociPrefix) && group.ociAddr == "":
				group.ociAddr = strings.TrimPrefix(addr, ociPrefix)
			}
		}
	}

	out := make([]*peerGroup, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}

	return out
}

func appendUnique(cids []string, cid string) []string {
	if slices.Contains(cids, cid) {
		return cids
	}

	return append(cids, cid)
}
