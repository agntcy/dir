// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package rpc

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
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
	MaxPullSize          = 4 * 1024 * 1024 // 4 MB
)

type RPCAPI struct {
	service *Service
}

type PullResponse struct {
	Cid         string
	Annotations map[string]string
	Data        []byte
}

type LookupResponse struct {
	Cid         string
	Annotations map[string]string
}

// NOTE: List-related types removed since List is a local-only operation
// and should not be part of peer-to-peer RPC communication

func (r *RPCAPI) Lookup(ctx context.Context, in *corev1.RecordRef, out *LookupResponse) error {
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
		Cid:         meta.GetCid(),
		Annotations: meta.GetAnnotations(),
	}

	return nil
}

func (r *RPCAPI) Pull(ctx context.Context, in *corev1.RecordRef, out *PullResponse) error {
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

	// pull data
	record, err := r.service.store.Pull(ctx, in)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to pull: %s", st.Message())
	}

	canonicalBytes, err := record.Marshal()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// set output
	*out = PullResponse{
		Cid:         meta.GetCid(),
		Data:        canonicalBytes,
		Annotations: meta.GetAnnotations(),
	}

	return nil
}

// NOTE: List RPC method removed since List is a local-only operation

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	store     types.StoreAPI
}

func New(host host.Host, store types.StoreAPI) (*Service, error) {
	service := &Service{
		rpcServer: rpc.NewServer(host, Protocol),
		host:      host,
		store:     store,
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

func (s *Service) Lookup(ctx context.Context, peer peer.ID, req *corev1.RecordRef) (*corev1.RecordRef, error) {
	logger.Debug("P2p RPC: Executing Lookup request on remote peer", "peer", peer, "req", req)

	var resp LookupResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncLookup, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	return &corev1.RecordRef{
		Cid: resp.Cid,
	}, nil
}

func (s *Service) Pull(ctx context.Context, peer peer.ID, req *corev1.RecordRef) (*corev1.Record, error) {
	logger.Debug("P2p RPC: Executing Pull request on remote peer", "peer", peer, "req", req)

	var resp PullResponse

	err := s.rpcClient.CallContext(ctx, peer, DirService, DirServiceFuncPull, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	record, err := corev1.UnmarshalRecord(resp.Data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record: %v", err)
	}

	return record, nil
}

// NOTE: List RPC client method removed since List is a local-only operation
// Use Search for network-wide record discovery instead
