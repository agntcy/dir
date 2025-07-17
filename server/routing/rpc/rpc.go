// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routetypes "github.com/agntcy/dir/api/routing/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("rpc")

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

type ListRequest struct {
	Peer   string
	Labels []string
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
	logger.Debug("P2p RPC: Executing Lookup request on remote peer", "peer", r.service.host.ID())

	// validate request
	if in == nil || out == nil {
		return status.Error(codes.InvalidArgument, "invalid request: nil request/response") //nolint:wrapcheck
	}

	// handle lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to lookup: %s", st.Message())
	}

	// write result
	*out = LookupResponse{
		Digest:      meta.GetDigest(),
		Type:        meta.GetType(),
		Size:        meta.GetSize(),
		Annotations: meta.GetAnnotations(),
	}

	return nil
}

func (r *RPCAPI) Pull(ctx context.Context, in *coretypes.ObjectRef, out *PullResponse) error {
	logger.Debug("P2p RPC: Executing Pull request on remote peer", "peer", r.service.host.ID())

	// validate request
	if in == nil || out == nil {
		return status.Error(codes.InvalidArgument, "invalid request: nil request/response") //nolint:wrapcheck
	}

	// lookup
	meta, err := r.service.store.Lookup(ctx, in)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to lookup: %s", st.Message())
	}

	// validate lookup before pull
	if meta.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return status.Errorf(codes.Internal, "can only pull agent object")
	}

	if meta.GetSize() > MaxPullSize {
		return status.Errorf(codes.Internal, "object too large to pull: %d bytes", meta.GetSize())
	}

	// pull data
	reader, err := r.service.store.Pull(ctx, meta)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to pull: %s", st.Message())
	}
	defer reader.Close()

	// read result from reader
	data, err := io.ReadAll(io.LimitReader(reader, MaxPullSize))
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read data: %v", err)
	}

	// set output
	*out = PullResponse{
		Digest:      meta.GetDigest(),
		Type:        meta.GetType(),
		Size:        meta.GetSize(),
		Data:        data,
		Annotations: meta.GetAnnotations(),
	}

	return nil
}

func (r *RPCAPI) List(ctx context.Context, inCh <-chan *ListRequest, outCh chan<- *ListResponse) error {
	defer close(outCh)

	for in := range inCh {
		logger.Debug("P2p RPC: Executing List request on remote peer", "peer", r.service.host.ID())

		// local list
		listCh, err := r.service.route.List(ctx, &routetypes.ListRequest{
			LegacyListRequest: &routetypes.LegacyListRequest{
				Labels: in.Labels,
			},
		})
		if err != nil {
			st := status.Convert(err)

			return status.Errorf(st.Code(), "failed to list: %s", st.Message())
		}

		// resolve response before forwarding
		for item := range listCh {
			result := &ListResponse{
				Labels:      item.GetLabels(),
				LabelCounts: item.GetLabelCounts(),
				Peer:        r.service.host.ID().String(), // remote peer where local list was called
			}

			if record := item.GetRecord(); record != nil {
				result.Annotations = record.GetAnnotations()
				result.Size = record.GetSize()
				result.Digest = record.GetDigest()
				result.Type = record.GetType()
			}

			// forward data
			outCh <- result
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

func New(host host.Host, store types.StoreAPI, route types.RoutingAPI) (*Service, error) {
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
	logger.Debug("P2p RPC: Executing Lookup request on remote peer", "peer", peer, "req", req)

	var resp LookupResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncLookup, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	return &coretypes.ObjectRef{
		Digest:      resp.Digest,
		Type:        resp.Type,
		Size:        resp.Size,
		Annotations: resp.Annotations,
	}, nil
}

func (s *Service) Pull(ctx context.Context, peer peer.ID, req *coretypes.ObjectRef) (*coretypes.Object, error) {
	logger.Debug("P2p RPC: Executing Pull request on remote peer", "peer", peer, "req", req)

	var resp PullResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncPull, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	// convert to agent
	// TODO
	var agent *objectsv1.Agent
	if err := json.Unmarshal(resp.Data, &agent); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal agent data: %v", err)
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
//
//nolint:mnd
func (s *Service) List(ctx context.Context, peers []peer.ID, req *routetypes.ListRequest) (<-chan *routetypes.LegacyListResponse_Item, error) {
	logger.Debug("P2p RPC: Executing List request on remote peers", "peers", peers, "req", req)

	// reserve reasonable buffer size for output results
	respCh := make(chan *routetypes.LegacyListResponse_Item, 10000)

	// run processing in the background
	outCh := make(chan *ListResponse, 10000) // used as intermediary forwarding channel
	go func() {
		// run logic in the background
		// prepare inputs for each call
		inCh := make(chan *ListRequest, len(peers)+1)
		for _, peer := range peers {
			inCh <- &ListRequest{
					Peer:   peer.String(),
					Labels: req.GetLegacyListRequest().GetLabels(),
			}
		}

		close(inCh)

		// run async
		errs := s.rpcClient.MultiStream(ctx,
			peers,
			DirService,
			DirServiceFuncList,
			inCh,
			outCh,
		)

		// log error
		if err := errors.Join(errs...); err != nil {
			logger.Error("Failed to process all List RPC requests", "error", err)

			return
		}

		logger.Info("Successfully processed all List RPC requests", "peers", peers)
	}()

	// forward results from one goroutine to the output channel
	go func() {
		// close resp channel once done so the subscribers can finish
		defer close(respCh)

		// remove duplicate outputs to avoid redundant entries
		// this can happen when multiple peers are connected to the same peer that holds the object
		seenPeerAgents := make(map[string]struct{})

		// forward data to response channel
		for out := range outCh {
			uniqueKey := out.Peer + out.Digest

			// check if we have already seen this peer
			if _, ok := seenPeerAgents[uniqueKey]; ok {
				continue
			}

			seenPeerAgents[uniqueKey] = struct{}{}
			respCh <- &routetypes.LegacyListResponse_Item{
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
