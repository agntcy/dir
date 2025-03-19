// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/stretchr/testify/assert"
)

func TestNode(t *testing.T) {
	// set log file
	file, err := os.Create("output.log")
	assert.NoError(t, err)
	defer file.Close()
	log.SetOutput(file)

	// set context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// create bootstrap node
	bootNode, _ := startNode(t, ctx, "/ip4/0.0.0.0/tcp/0", nil)
	var bootAddrs []string
	for _, mAddr := range bootNode.Addrs() {
		bootAddrs = append(bootAddrs, mAddr.String()+"/p2p/"+bootNode.ID().String())
	}

	// create participating nodes
	_, aliceCh := startNode(t, ctx, "/ip4/0.0.0.0/tcp/0", bootAddrs)
	_, bobCh := startNode(t, ctx, "/ip4/0.0.0.0/tcp/0", bootAddrs)

	// make sure they exchanged messages
	log.Println("waiting for nodes to connect...")
	log.Println("received from alice:", <-aliceCh)
	log.Println("received from bob:", <-bobCh)
}

func startNode(t *testing.T, ctx context.Context, addr string, bootstrapAddrs []string) (host.Host, <-chan string) {
	hostCh := make(chan host.Host)
	readCh := make(chan string)

	// start node
	go func() {
		err := start(ctx, addr, bootstrapAddrs, hostCh, readCh)
		assert.NoError(t, err)
	}()
	host := <-hostCh

	return host, readCh
}
