// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore/query"
	ocidigest "github.com/opencontainers/go-digest"
)

var (
	// TODO: expose gRPC interfaces over p2p via streams or RPCs.
	ProtocolID         = "dir/v1.0.0"
	ProtocolRendezvous = ProtocolID + "connect"
)

type routing struct {
	ds types.Datastore
}

func New(opts types.APIOptions) (types.RoutingAPI, error) {
	return &routing{
		ds: opts.Datastore(),
	}, nil
}

func (r *routing) Publish(context.Context, *coretypes.ObjectRef, *coretypes.Agent) error {
	panic("unimplemented")
}

func (r *routing) List(ctx context.Context, q string) ([]*coretypes.ObjectRef, error) {
	var refs []*coretypes.ObjectRef //nolint:prealloc

	// Validate query
	if !isValidQuery(q) {
		return nil, fmt.Errorf("invalid query: %s", q)
	}

	res, err := r.ds.Query(ctx, query.Query{
		Prefix: q,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query datastore: %w", err)
	}

	// Convert results to ObjectRefs
	for entry := range res.Next() {
		digest, err := getAgentDigestFromKey(entry.Key)
		if err != nil {
			return nil, fmt.Errorf("failed to get digest from key: %w", err)
		}

		refs = append(refs, &coretypes.ObjectRef{
			Digest: digest,
		})
	}

	return refs, nil
}

var supportedQueryTypes = []string{
	"/skills/",
	"/locators/",
	"/extensions/",
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
	if _, err := ocidigest.Parse(parts[len(parts)-1]); err != nil {
		return "", fmt.Errorf("invalid digest: %s", parts[len(parts)-1])
	}

	return parts[len(parts)-1], nil
}
