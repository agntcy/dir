// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

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

type PullResponse struct {
	Digest      string
	Type        string
	Size        uint64
	Annotations map[string]string
	Data        []byte
}

type LookupResponse struct {
	Digest      string
	Type        string
	Size        uint64
	Annotations map[string]string
}

type ListResponse struct {
	Labels      []string
	LabelCounts map[string]uint64
	Peer        string
	Digest      string
	Type        string
	Size        uint64
	Annotations map[string]string
}

func (r *RPCAPI) Lookup(ctx context.Context, in *coretypes.ObjectRef, out *LookupResponse) error {
	// validate request
	if in == nil || out == nil {
		return errors.New("invalid request: nil request/response")
	}

	// handle lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// write result
	*out = LookupResponse{
		Digest:      meta.GetDigest(),
		Type:        meta.GetType(),
		Size:        meta.GetSize(),
		Annotations: meta.Annotations,
	}

	return nil
}

func (r *RPCAPI) Pull(ctx context.Context, in *coretypes.ObjectRef, out *PullResponse) error {
	// validate request
	if in == nil || out == nil {
		return errors.New("invalid request: nil request/response")
	}

	// lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// validate lookup before pull
	if meta.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return errors.New("can only pull agent object")
	}

	if meta.GetSize() > MaxPullSize {
		return fmt.Errorf("object too large to pull: %d bytes", meta.GetSize())
	}

	// pull data
	reader, err := r.service.store.Pull(ctx, meta)
	if err != nil {
		return fmt.Errorf("failed to pull: %w", err)
	}
	defer reader.Close()

	// read result from reader
	data, err := io.ReadAll(io.LimitReader(reader, MaxPullSize))
	if err != nil {
		return fmt.Errorf("failed to read: %w", err)
	}

	// set output
	*out = PullResponse{
		Digest:      meta.GetDigest(),
		Type:        meta.GetType(),
		Size:        meta.GetSize(),
		Data:        data,
		Annotations: meta.Annotations,
	}

	return nil
}

func (r *RPCAPI) List(ctx context.Context, inCh <-chan *routetypes.ListRequest, out chan<- *ListResponse) error {
	// validate request
	in := <-inCh
	if in == nil || out == nil {
		return errors.New("invalid request: nil request/response")
	}

	// list
	listCh, err := r.service.route.List(ctx, in)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// forward results
	for item := range listCh {
		out <- &ListResponse{
			Labels:      item.Labels,
			LabelCounts: item.LabelCounts,
			Peer:        item.Peer.Id,
			Digest:      item.Record.GetDigest(),
			Type:        item.Record.GetType(),
			Size:        item.Record.GetSize(),
			Annotations: item.Record.Annotations,
		}
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
	var resp LookupResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncLookup, req, &resp)
	if err != nil {
		return nil, err
	}

	return &coretypes.ObjectRef{
		Digest:      resp.Digest,
		Type:        resp.Type,
		Size:        resp.Size,
		Annotations: resp.Annotations,
	}, nil
}

func (s *Service) Pull(ctx context.Context, peer peer.ID, req *coretypes.ObjectRef) (*coretypes.Object, error) {
	var resp PullResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncPull, req, &resp)
	if err != nil {
		return nil, err
	}

	// convert to agent
	var agent *coretypes.Agent
	if err := json.Unmarshal(resp.Data, &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	return &coretypes.Object{
		Ref: &coretypes.ObjectRef{
			Digest:      resp.Digest,
			Type:        resp.Type,
			Size:        resp.Size,
			Annotations: resp.Annotations,
		},
		Agent: agent,
	}, nil
}

// range over the result channel, then read the error after the loop.
// this is done in best effort mode.
func (s *Service) List(ctx context.Context, peers []peer.ID, req *routetypes.ListRequest) (<-chan *routetypes.ListResponse_Item, error) {
	respCh := make(chan *routetypes.ListResponse_Item, 10)

	go func() {
		inCh := make(chan *routetypes.ListRequest, 1)
		outCh := make(chan *ListResponse, 10)

		// close on done
		defer close(respCh)
		defer close(outCh)

		// run listing
		inCh <- req
		errs := s.rpcClient.MultiStream(ctx,
			filterPeers(s.host.ID(), peers),
			DirService,
			DirServiceFuncList,
			inCh,
			outCh,
		)

		// log error
		if err := errors.Join(errs...); err != nil {
			log.Printf("Failed to list data on remote: %v", err)
		}

		// try forward data left in the channel as only some requests may have failed
		for out := range outCh {
			respCh <- &routetypes.ListResponse_Item{
				Labels:      out.Labels,
				LabelCounts: out.LabelCounts,
				Peer: &routetypes.Peer{
					Id: out.Peer,
				},
				Record: &coretypes.ObjectRef{
					Digest:      out.Digest,
					Type:        out.Type,
					Size:        out.Size,
					Annotations: out.Annotations,
				},
			}
		}
	}()

	return respCh, nil
}

func filterPeers(self peer.ID, peers []peer.ID) peer.IDSlice {
	var filtered peer.IDSlice

	for _, pID := range peers {
		if pID != self {
			filtered = append(filtered, pID)
		}
	}

	return filtered
}
