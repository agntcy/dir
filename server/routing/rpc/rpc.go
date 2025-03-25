// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// TODO: proper cleanup and implementation needed!

const (
	Protocol             = protocol.ID("/dir/rpc/1.0.0")
	DirService           = "RPCAPI"
	DirServiceFuncLookup = "Lookup"
	DirServiceFuncPull   = "Pull"
	DirServiceFuncList   = "List"
	MaxPullSize          = 4 * 1024 * 1024 // 4 MB
)

type RPCAPI struct {
	service *Service
}

func (r *RPCAPI) Lookup(ctx context.Context, in *coretypes.ObjectRef, out *coretypes.ObjectRef) error {
	// validate request
	if in == nil || out == nil {
		return fmt.Errorf("invalid request: nil request/response")
	}

	// handle lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	*out = *meta //nolint

	return nil
}

func (r *RPCAPI) Pull(ctx context.Context, in *coretypes.ObjectRef, out *coretypes.Object) error {
	// validate request
	if in == nil || out == nil {
		return fmt.Errorf("invalid request: nil request/response")
	}

	// lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// validate lookup before pull
	if meta.Type != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return fmt.Errorf("can only pull agent object")
	}
	if meta.Size > MaxPullSize {
		return fmt.Errorf("object too large to pull: %d bytes", meta.Size)
	}

	// pull data
	reader, err := r.service.store.Pull(ctx, meta)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}
	defer reader.Close()

	// read result from reader
	data, err := io.ReadAll(io.LimitReader(reader, int64(meta.Size)))
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	// convert to agent
	var agent *coretypes.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	*out = coretypes.Object{
		Ref:   meta,
		Agent: agent,
	}

	return nil
}

func (r *RPCAPI) List(ctx context.Context, inCh <-chan *routetypes.ListRequest, out chan<- *routetypes.ListResponse_Item) error {
	// validate request
	in := <-inCh
	if in == nil || out == nil {
		return fmt.Errorf("invalid request: nil request/response")
	}

	// list
	listCh, err := r.service.route.List(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// forward results
	for item := range listCh {
		out <- item
	}

	return nil
}

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	store     types.StoreAPI
	route     types.RoutingAPI
}

func New(ctx context.Context, host host.Host, store types.StoreAPI, route types.RoutingAPI) (*Service, error) {
	service := &Service{
		rpcServer: rpc.NewServer(host, Protocol),
		host:      host,
		store:     store,
		route:     route,
	}

	// register api
	rpcAPI := RPCAPI{service: service}
	err := service.rpcServer.Register(&rpcAPI)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	// update client
	service.rpcClient = rpc.NewClientWithServer(host, Protocol, service.rpcServer)

	return service, nil
}

func (s *Service) Lookup(ctx context.Context, peer peer.ID, req *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	var resp coretypes.ObjectRef
	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncLookup, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *Service) Pull(ctx context.Context, peer peer.ID, req *coretypes.ObjectRef) (*coretypes.Object, error) {
	var resp coretypes.Object
	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncPull, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// range over the result channel, then read the error after the loop
func (s *Service) List(ctx context.Context, peers []peer.ID, req *routetypes.ListRequest) (<-chan *routetypes.ListResponse_Item, <-chan error) {
	outCh := make(chan *routetypes.ListResponse_Item, 10)
	errCh := make(chan error, 1)
	go func() {
		inCh := make(chan *routetypes.ListRequest, 1)
		inCh <- req
		errs := s.rpcClient.MultiStream(ctx,
			peers,
			DirService,
			DirServiceFuncList,
			inCh,
			outCh,
		)
		errCh <- errors.Join(errs...)
	}()

	return outCh, errCh
}

func (s *Service) getPeers() peer.IDSlice {
	var filtered peer.IDSlice
	for _, pID := range s.host.Peerstore().Peers() {
		if pID != s.host.ID() {
			filtered = append(filtered, pID)
		}
	}

	return filtered
}
