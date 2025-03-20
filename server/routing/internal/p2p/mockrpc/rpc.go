// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package mockrpc

import (
	"context"
	"fmt"
	"log"
	"time"

	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const (
	EchoService         = "EchoRPCAPI"
	EchoServiceFuncEcho = "Echo"
)

type EchoRPCAPI struct {
	service *Service
}

type Envelope struct {
	Message string
}

func (r *EchoRPCAPI) Echo(_ context.Context, in Envelope, out *Envelope) error {
	*out = r.service.ReceiveEcho(in)

	return nil
}

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	protocol  protocol.ID
	listenCh  chan<- string
	ignored   peerMap
}

func Start(ctx context.Context, host host.Host, protocol protocol.ID, listenCh chan<- string, ignored []peer.AddrInfo) error {
	service := &Service{
		host:     host,
		protocol: protocol,
		listenCh: listenCh,
		ignored:  newPeerMap(append(peer.AddrInfosToIDs(ignored), host.ID())),
	}

	err := service.SetupRPC()
	if err != nil {
		return err
	}

	// send dummy message
	go service.StartMessaging(ctx)

	return nil
}

func (s *Service) StartMessaging(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Echo(ctx, "Message: Hello from "+s.host.ID().String())
		}
	}
}

func (s *Service) SetupRPC() error {
	echoRPCAPI := EchoRPCAPI{service: s}

	s.rpcServer = rpc.NewServer(s.host, s.protocol)

	err := s.rpcServer.Register(&echoRPCAPI)
	if err != nil {
		return err //nolint:wrapcheck
	}

	s.rpcClient = rpc.NewClientWithServer(s.host, s.protocol, s.rpcServer)

	return nil
}

func (s *Service) Echo(ctx context.Context, message string) {
	peers := filterPeers(s.host.Peerstore().Peers(), s.ignored)
	replies := make([]*Envelope, len(peers))

	// Send message to all peers
	errs := s.rpcClient.MultiCall(
		newCtxsN(ctx, len(peers)),
		peers,
		EchoService,
		EchoServiceFuncEcho,
		Envelope{Message: message},
		copyEnvelopesToIfaces(replies),
	)

	// Check responses from peers
	for i, err := range errs {
		if err != nil {
			log.Printf("Peer %s returned error: %-v\n", peers[i].String(), err)
		} else {
			log.Printf("Peer %s echoed: %s\n", peers[i].String(), replies[i].Message)
		}
	}
}

func (s *Service) ReceiveEcho(e Envelope) Envelope {
	msg := fmt.Sprintf("Peer %s echoing: %s", s.host.ID(), e.Message)
	s.listenCh <- msg

	return Envelope{Message: msg}
}

func newCtxsN(ctx context.Context, n int) []context.Context {
	ctxs := make([]context.Context, 0, n)
	for range n {
		ctxs = append(ctxs, ctx)
	}

	return ctxs
}

func copyEnvelopesToIfaces(in []*Envelope) []interface{} {
	ifaces := make([]interface{}, len(in))

	for i := range in {
		in[i] = &Envelope{}
		ifaces[i] = in[i]
	}

	return ifaces
}

type peerMap map[peer.ID]struct{}

func newPeerMap(peers peer.IDSlice) peerMap {
	peerMap := peerMap{}
	for _, peer := range peers {
		peerMap[peer] = struct{}{}
	}

	return peerMap
}

func filterPeers(peers peer.IDSlice, ignored peerMap) peer.IDSlice {
	var filtered peer.IDSlice

	for _, p := range peers {
		if _, exists := ignored[p]; !exists {
			filtered = append(filtered, p)
		}
	}

	return filtered
}
