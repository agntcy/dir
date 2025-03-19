// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"path"
	"testing"

	"github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	ctx := context.Background()

	// create bootstrap node
	bootstrapAddr := "/ip4/0.0.0.0/tcp/9000"
	bootNode, err := NewNode(ctx, bootstrapAddr, nil, datastore.NewMapDatastore())
	assert.NoError(t, err)
	defer bootNode.Close()
	bootstrapAddrs := []string{
		path.Join(bootstrapAddr, "ipfs", bootNode.host.ID().String()),
	}

	// create participating nodes
	alice, err := NewNode(ctx, "/ip4/0.0.0.0/tcp/9001", bootstrapAddrs, datastore.NewMapDatastore())
	assert.NoError(t, err)
	defer alice.Close()

	bob, err := NewNode(ctx, "/ip4/0.0.0.0/tcp/9002", bootstrapAddrs, datastore.NewMapDatastore())
	assert.NoError(t, err)
	defer bob.Close()

	// TODO: add routing flow tests
}
