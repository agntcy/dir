// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/protocol"
	mh "github.com/multiformats/go-multihash"
	ocidigest "github.com/opencontainers/go-digest"
)

var (
	// TODO: expose gRPC interfaces over p2p via streams or RPCs.
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
		// p2p.WithRefreshInterval(1*time.Minute),
		p2p.WithRandevous(ProtocolRendezvous),
		p2p.WithIdentityKeyPath(opts.Config().Routing.KeyPath),
		p2p.WithCustomDHTOpts(
			dht.Datastore(opts.Datastore()),
			// dht.Validator(&validator{}),
			dht.NamespacedValidator("dir", &validator{}),
			dht.ProtocolPrefix(protocol.ID(ProtocolPrefix)),
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
	// Validate
	if ref.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return fmt.Errorf("invalid object type: %s", ref.GetType())
	}

	// Keep track of all skills attribute keys.
	// We will publish this to the DHT.
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

	// Announce skills to the DHT
	if err := r.announceSkills(ctx, skills); err != nil {
		return fmt.Errorf("failed to announce skills to the network: %w", err)
	}

	return nil
}

func (r *routing) List(ctx context.Context, prefixQuery string) ([]*coretypes.ObjectRef, error) {
	// Validate query
	if !isValidQuery(prefixQuery) {
		return nil, fmt.Errorf("invalid query: %s", prefixQuery)
	}

	// Query local data
	results, err := r.dstore.Query(ctx, query.Query{Prefix: prefixQuery})
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

	// Resolve query!
	// Find providers and try fetching from those
	attrKey := path.Clean("/dir/" + prefixQuery)
	resCh := r.server.DHT().FindProvidersAsync(ctx, keyToCid(attrKey), 10)
	for res := range resCh {
		fmt.Printf("Found a peer %s serving the key %s\n", res.ID.String(), attrKey)
	}

	return records, nil
}

// Announce to the network that we can provide
// the data for the given skills.
//
// We need to announce that we can serve all available skills, ie
//
//	skill/X, skill/X/Y, skill/X/Y/Z
//
// Otherwise, we cannot lookup if a node has: skill/text, or skill/text/rag
//
// We dont publish data about locators, we filter that directly from the node
// once we discover it via DHT and connect to it via RPC.
func (r *routing) announceSkills(ctx context.Context, skills []string) error {
	// Create a map of skills we need to publish
	skillMap := map[string]*struct{}{}
	for _, skill := range skills {
		for _, subSkill := range splitSkillPath(skill) {
			skillMap[subSkill] = nil
		}
	}

	// Announce skills
	for skill := range skillMap {
		skill = path.Clean("/dir/" + skill)
		err := r.server.DHT().Provide(ctx, keyToCid(skill), true)
		if err != nil {
			return fmt.Errorf("failed to announce skill to the network: %w", err)
		}
	}

	return nil
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

func keyToCid(key string) cid.Cid {
	hash, _ := mh.Sum([]byte(key), mh.SHA2_256, -1)
	return cid.NewCidV1(cid.Raw, hash)
}

// splits the skill into sub-skills, for example:
// /skills/text/rag => /skills/text, /skills/text/rag
func splitSkillPath(skillPath string) []string {
	// Split the skillPath by the delimiter '/'
	parts := strings.Split(path.Clean(skillPath), "/")

	subSkills := []string{}
	for i := 2; i < len(parts); i++ {
		// Construct sub-skill paths incrementally
		subSkill := strings.Join(parts[:i+1], "/")
		subSkills = append(subSkills, subSkill)
	}

	// always return sorted
	slices.Sort(subSkills)

	return subSkills
}
