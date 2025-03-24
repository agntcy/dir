// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"strings"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/protocol"
	ocidigest "github.com/opencontainers/go-digest"
)

var (
	ProtocolPrefix     = "dir"
	ProtocolRendezvous = ProtocolPrefix + "/connect"
)

type routing struct {
	dstore types.Datastore
	server *p2p.Server
}

func New(ctx context.Context, opts types.APIOptions) (types.RoutingAPI, error) {
	// Create P2P server
	server, err := p2p.New(ctx,
		p2p.WithListenAddress(opts.Config().Routing.ListenAddress),
		p2p.WithBootstrapAddrs(opts.Config().Routing.BootstrapPeers),
		p2p.WithRefreshInterval(1*time.Second), // quick refresh, TODO: make configurable
		p2p.WithRandevous(ProtocolRendezvous),  // enable libp2p auto-discovery
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithCustomDHTOpts(
			dht.Datastore(opts.Datastore()), // custom DHT datastore
			// dht.Validator(&validator{}),
			dht.NamespacedValidator("dir", &validator{}),    // custom namespace validator
			dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)), // custom DHT protocol
			dht.ProviderStore(&peerstore{}),                 // provider store
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p: %w", err)
	}

	return &routing{
		dstore: opts.Datastore(),
		server: server,
	}, nil
}

func (r *routing) Publish(ctx context.Context, ref *coretypes.ObjectRef, agent *coretypes.Agent) error {
	// Keep track of all skill attribute keys.
	// We will record this across the network.
	var skills []string

	// Cache skills
	for _, skill := range agent.GetSkills() {
		skillKey := fmt.Sprintf("/skills/%s", skill.Key())
		agentSkillKey := fmt.Sprintf("%s/%s", skillKey, ref.GetDigest())
		if err := r.dstore.Put(ctx, datastore.NewKey(agentSkillKey), nil); err != nil {
			return fmt.Errorf("failed to put skill key: %w", err)
		}

		skills = append(skills, skillKey)
	}

	// Cache locators
	for _, loc := range agent.GetLocators() {
		agentLocatorKey := fmt.Sprintf("/locators/%s/%s", loc.Key(), ref.GetDigest())
		if err := r.dstore.Put(ctx, datastore.NewKey(agentLocatorKey), nil); err != nil {
			return fmt.Errorf("failed to put locator key: %w", err)
		}
	}

	// Broadcast to the DHT that we are providing this object
	err := r.server.DHT().Provide(ctx, ref.GetCID(), true)
	if err != nil {
		return fmt.Errorf("failed to announce skill to the network: %w", err)
	}

	// TODO: sync records across p2p network via RPC

	return nil
}

func (r *routing) List(ctx context.Context, prefixQuery string) ([]*coretypes.ObjectRef, error) {
	// Validate query
	if !isValidQuery(prefixQuery) {
		return nil, fmt.Errorf("invalid query: %s", prefixQuery)
	}

	// Query local data
	results, err := r.dstore.Query(ctx, query.Query{
		Prefix: prefixQuery,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query datastore: %w", err)
	}

	// Store fetched data into a slice
	var records []*coretypes.ObjectRef

	// Fetch from local
	for entry := range results.Next() {
		digest, err := getAgentDigestFromKey(entry.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get digest from key: %w", err)
		}

		records = append(records, &coretypes.ObjectRef{
			Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			Digest: digest,
		})
	}

	// TODO: if connected, reach out across the network to find content

	return records, nil
}

var supportedQueryTypes = []string{
	"/skills/",
	"/locators/",
}

func isValidQuery(q string) bool {
	// Check if query has at least a query type and a query value
	// e.e. /skills/ is not valid since it does not have a skill query value
	parts := strings.Split(strings.Trim(q, "/"), "/")
	if len(parts) < 2 { //nolint:mnd
		return false
	}

	// Check if query type is supported
	for _, s := range supportedQueryTypes {
		if strings.HasPrefix(q, s) {
			return true
		}
	}

	return false
}

func getAgentDigestFromKey(k string) (string, error) {
	parts := strings.Split(k, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid key: %s", k)
	}

	// Check if last part is a valid digest.
	digest := parts[len(parts)-1]
	if _, err := ocidigest.Parse(digest); err != nil {
		return "", fmt.Errorf("invalid digest: %s", digest)
	}

	return digest, nil
}
