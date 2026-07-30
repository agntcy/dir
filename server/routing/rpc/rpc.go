// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:revive
package rpc

import (
	"context"
	"errors"

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

var (
	errReferrerListCap = errors.New("referrer list cap reached")
	errReferrerFound   = errors.New("referrer found")
)

// TODO: proper cleanup and implementation needed!

const (
	Protocol                    = protocol.ID("/dir/rpc/1.0.0")
	DirService                  = "RPCAPI"
	DirServiceFuncLookup        = "Lookup"
	DirServiceFuncPull          = "Pull"
	DirServiceFuncListReferrers = "ListReferrers"
	DirServiceFuncPullReferrer  = "PullReferrer"
	MaxPullSize                 = 4 * 1024 * 1024 // 4 MB
	MaxListReferrers            = 1024
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

// ReferrerDescriptor carries referrer metadata without blob data.
type ReferrerDescriptor struct {
	Cid  string
	Type string
}

type ListReferrersResponse struct {
	Referrers []ReferrerDescriptor
	Truncated bool
}

type PullReferrerRequest struct {
	RecordRef *corev1.RecordRef
	Referrer  *ReferrerDescriptor
}

type PullReferrerResponse struct {
	Referrer *corev1.RecordReferrer
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

func (r *RPCAPI) ListReferrers(ctx context.Context, in *corev1.RecordRef, out *ListReferrersResponse) error {
	if in == nil || out == nil {
		return status.Error(codes.InvalidArgument, "invalid request: nil request/response") //nolint:wrapcheck
	}

	requestPeer, _ := rpc.GetRequestSender(ctx)

	logger.Debug("P2p RPC: Executing ListReferrers request on remote peer",
		"peer", r.service.localPeerID(),
		"request_peer", requestPeer,
		"record_cid", in.GetCid(),
	)

	if err := validateRecordRef(in); err != nil {
		return err //nolint:wrapcheck
	}

	refStore, err := r.service.getReferrerStore()
	if err != nil {
		return err //nolint:wrapcheck
	}

	var (
		referrers []ReferrerDescriptor
		truncated bool
	)

	walkFn := func(referrer *corev1.RecordReferrer) error {
		if len(referrers) >= MaxListReferrers {
			truncated = true

			return errReferrerListCap
		}

		referrers = append(referrers, ReferrerDescriptor{
			Cid:  referrer.GetReferrerRef().GetCid(),
			Type: referrer.GetType(),
		})

		return nil
	}

	if err := refStore.WalkReferrers(ctx, in.GetCid(), "", walkFn); err != nil && !errors.Is(err, errReferrerListCap) {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to list referrers: %s", st.Message())
	}

	if truncated {
		logger.Warn("ListReferrers returned truncated referrer metadata",
			"request_peer", requestPeer,
			"record_cid", in.GetCid(),
			"count", len(referrers),
		)
	}

	*out = ListReferrersResponse{Referrers: referrers, Truncated: truncated}

	return nil
}

func (r *RPCAPI) PullReferrer(ctx context.Context, in *PullReferrerRequest, out *PullReferrerResponse) error {
	if in == nil || out == nil {
		return status.Error(codes.InvalidArgument, "invalid request: nil request/response") //nolint:wrapcheck
	}

	requestPeer, _ := rpc.GetRequestSender(ctx)

	logger.Debug("P2p RPC: Executing PullReferrer request on remote peer",
		"peer", r.service.localPeerID(),
		"request_peer", requestPeer,
		"record_cid", in.RecordRef.GetCid(),
		"referrer_type", in.Referrer.Type,
		"referrer_cid", in.Referrer.Cid,
	)

	if err := validateRecordRef(in.RecordRef); err != nil {
		return err //nolint:wrapcheck
	}

	if in.Referrer.Type == "" {
		return status.Error(codes.InvalidArgument, "referrer type is required") //nolint:wrapcheck
	}

	if in.Referrer.Cid == "" {
		return status.Error(codes.InvalidArgument, "referrer cid is required") //nolint:wrapcheck
	}

	refStore, err := r.service.getReferrerStore()
	if err != nil {
		return err //nolint:wrapcheck
	}

	var matched *corev1.RecordReferrer

	walkFn := func(referrer *corev1.RecordReferrer) error {
		if referrer.GetReferrerRef().GetCid() != in.Referrer.Cid {
			return nil
		}

		matched = referrer

		return errReferrerFound
	}

	err = refStore.WalkReferrers(ctx, in.RecordRef.GetCid(), in.Referrer.Type, walkFn)
	if err != nil && !errors.Is(err, errReferrerFound) {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to pull referrer: %s", st.Message())
	}

	if matched == nil {
		return status.Errorf(codes.NotFound, "referrer %s not found for record %s", in.Referrer.Cid, in.RecordRef.GetCid())
	}

	blob, err := matched.Marshal()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to marshal referrer: %v", err)
	}

	if len(blob) > MaxPullSize {
		logger.Warn("PullReferrer rejected oversize referrer blob",
			"request_peer", requestPeer,
			"record_cid", in.RecordRef.GetCid(),
			"referrer_type", in.Referrer.Type,
			"referrer_cid", in.Referrer.Cid,
			"size", len(blob),
			"max_size", MaxPullSize,
		)

		return status.Errorf(codes.FailedPrecondition, "referrer blob exceeds maximum size of %d bytes", MaxPullSize)
	}

	logger.Info("PullReferrer served referrer",
		"request_peer", requestPeer,
		"record_cid", in.RecordRef.GetCid(),
		"referrer_type", in.Referrer.Type,
		"referrer_cid", in.Referrer.Cid,
		"size", len(blob),
	)

	*out = PullReferrerResponse{Referrer: matched}

	return nil
}

// NOTE: List RPC method removed since List is a local-only operation

type Service struct {
	rpcServer *rpc.Server
	rpcClient *rpc.Client
	host      host.Host
	store     types.StoreAPI
	refStore  types.ReferrerStoreAPI
}

func New(host host.Host, store types.StoreAPI) (*Service, error) {
	var refStore types.ReferrerStoreAPI
	if rs, ok := store.(types.ReferrerStoreAPI); ok {
		refStore = rs
	}

	service := &Service{
		rpcServer: rpc.NewServer(host, Protocol),
		host:      host,
		store:     store,
		refStore:  refStore,
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

func (s *Service) ListReferrers(ctx context.Context, peerID peer.ID, req *corev1.RecordRef) ([]ReferrerDescriptor, error) {
	logger.Debug("P2p RPC: Executing ListReferrers request on remote peer", "peer", peerID, "req", req)

	var resp ListReferrersResponse

	err := s.rpcClient.CallContext(ctx, peerID, DirService, DirServiceFuncListReferrers, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	return resp.Referrers, nil
}

func (s *Service) PullReferrer(
	ctx context.Context,
	peerID peer.ID,
	recordRef *corev1.RecordRef,
	referrer ReferrerDescriptor,
) (*corev1.RecordReferrer, error) {
	logger.Debug("P2p RPC: Executing PullReferrer request on remote peer",
		"peer", peerID,
		"record_ref", recordRef,
		"referrer_type", referrer.Type,
		"referrer_cid", referrer.Cid,
	)

	req := &PullReferrerRequest{
		RecordRef: recordRef,
		Referrer: &ReferrerDescriptor{
			Type: referrer.Type,
			Cid:  referrer.Cid,
		},
	}

	var resp PullReferrerResponse

	err := s.rpcClient.CallContext(ctx, peerID, DirService, DirServiceFuncPullReferrer, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to call remote peer: %v", err)
	}

	return resp.Referrer, nil
}

func (s *Service) getReferrerStore() (types.ReferrerStoreAPI, error) {
	if s.refStore == nil {
		return nil, status.Error(codes.Unimplemented, "referrer storage is not supported by the current store implementation") //nolint:wrapcheck
	}

	return s.refStore, nil
}

func (s *Service) localPeerID() peer.ID {
	if s.host == nil {
		return ""
	}

	return s.host.ID()
}

func validateRecordRef(recordRef *corev1.RecordRef) error {
	if recordRef == nil || recordRef.GetCid() == "" {
		return status.Error(codes.InvalidArgument, "record cid is required") //nolint:wrapcheck
	}

	return nil
}

// NOTE: List RPC client method removed since List is a local-only operation
// Use Search for network-wide record discovery instead
