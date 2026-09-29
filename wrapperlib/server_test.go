// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// mustProtoJSON marshals msg with protojson, failing the test on error. It is
// used throughout the test files to build request bodies for handle-based
// exported functions without hand-writing brittle JSON strings.
func mustProtoJSON(t *testing.T, msg proto.Message) string {
	t.Helper()

	b, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("failed to marshal %T to protojson: %v", msg, err)
	}

	return string(b)
}

// assertErrorNonEmpty parses resp as a JSON object with an "error" field and
// fails the test if that field is empty.
func assertErrorNonEmpty(t *testing.T, resp string) {
	t.Helper()

	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		t.Fatalf("response is not valid JSON: %v (response: %s)", err, resp)
	}

	if out.Error == "" {
		t.Fatalf("expected a non-empty error field, got response: %s", resp)
	}
}

// recordRefEnvelopeJSON builds the plain-JSON envelope `{"record_ref": <protojson RecordRef>}`
// expected by PullSignatures/PullPublicKeys (see recordRefRequest in verify.go),
// which is not itself a proto message so it cannot be built via mustProtoJSON.
func recordRefEnvelopeJSON(t *testing.T, cid string) string {
	t.Helper()

	refJSON := mustProtoJSON(t, &corev1.RecordRef{Cid: cid})

	return fmt.Sprintf(`{"record_ref":%s}`, refJSON)
}

// mustProtoUnmarshal parses raw (protojson-encoded, as produced by
// protoToRaw/marshalProto) into msg, failing the test on error. Plain
// encoding/json must not be used for these payloads: protojson emits
// camelCase field names by default (e.g. "referrerRef"), which do not match
// the snake_case `json:"..."` struct tags encoding/json would look for.
func mustProtoUnmarshal(t *testing.T, raw json.RawMessage, msg proto.Message) {
	t.Helper()

	if err := protojson.Unmarshal(raw, msg); err != nil {
		t.Fatalf("failed to unmarshal %T from protojson: %v (raw: %s)", msg, err, raw)
	}
}

// assertNoError parses resp as a JSON object with an "error" field and fails
// the test if that field is non-empty.
func assertNoError(t *testing.T, resp string) {
	t.Helper()

	var out struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		t.Fatalf("response is not valid JSON: %v (response: %s)", err, resp)
	}

	if out.Error != "" {
		t.Fatalf("unexpected error in response: %s", out.Error)
	}
}

// --- shared canned data ---

func mustStruct(m map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}

	return s
}

var cannedRecordData = mustStruct(map[string]any{
	"name":           "test-record",
	"schema_version": "v0.7.0",
})

const cannedPublicKeyPEM = "-----BEGIN PUBLIC KEY-----\nMOCKKEYDATA\n-----END PUBLIC KEY-----"

// --- mock storev1.StoreServiceServer ---

type mockStoreServer struct {
	storev1.UnimplementedStoreServiceServer

	pushSeq atomic.Int64
}

func (s *mockStoreServer) Push(stream storev1.StoreService_PushServer) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err //nolint:wrapcheck
		}

		n := s.pushSeq.Add(1)
		if err := stream.Send(&corev1.RecordRef{Cid: fmt.Sprintf("baecid-push-%03d", n)}); err != nil {
			return err //nolint:wrapcheck
		}
	}
}

func (s *mockStoreServer) Pull(stream storev1.StoreService_PullServer) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err //nolint:wrapcheck
		}

		if err := stream.Send(&corev1.Record{Data: cannedRecordData}); err != nil {
			return err //nolint:wrapcheck
		}
	}
}

func (s *mockStoreServer) Lookup(stream storev1.StoreService_LookupServer) error {
	for {
		ref, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return err //nolint:wrapcheck
		}

		meta := &corev1.RecordMeta{
			Cid:           ref.GetCid(),
			SchemaVersion: "v0.7.0",
			CreatedAt:     "2024-01-01T00:00:00Z",
		}
		if err := stream.Send(meta); err != nil {
			return err //nolint:wrapcheck
		}
	}
}

func (s *mockStoreServer) Delete(stream storev1.StoreService_DeleteServer) error {
	for {
		_, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(&emptypb.Empty{}) //nolint:wrapcheck
		}

		if err != nil {
			return err //nolint:wrapcheck
		}
	}
}

func (s *mockStoreServer) PushReferrer(stream storev1.StoreService_PushReferrerServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err //nolint:wrapcheck
	}

	return stream.Send(&storev1.PushReferrerResponse{ //nolint:wrapcheck
		Success:     true,
		ReferrerRef: &corev1.ReferrerRef{Cid: req.GetRecordRef().GetCid()},
	})
}

func (s *mockStoreServer) PullReferrer(stream storev1.StoreService_PullReferrerServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err //nolint:wrapcheck
	}

	sig := &signv1.Signature{
		SignedAt:    "2024-01-01T00:00:00Z",
		Algorithm:   "ecdsa",
		Signature:   "c2ln",
		ContentType: "application/vnd.dev.cosign.simplesigning.v1+json",
	}

	sigReferrer, err := sig.MarshalReferrer()
	if err != nil {
		return err //nolint:wrapcheck
	}

	sigReferrer.RecordRef = req.GetRecordRef()

	pk := &signv1.PublicKey{Key: cannedPublicKeyPEM}

	pkReferrer, err := pk.MarshalReferrer()
	if err != nil {
		return err //nolint:wrapcheck
	}

	pkReferrer.RecordRef = req.GetRecordRef()

	var referrers []*corev1.RecordReferrer

	switch req.GetReferrerType() {
	case corev1.SignatureReferrerType:
		referrers = []*corev1.RecordReferrer{sigReferrer}
	case corev1.PublicKeyReferrerType:
		referrers = []*corev1.RecordReferrer{pkReferrer}
	default:
		referrers = []*corev1.RecordReferrer{sigReferrer, pkReferrer}
	}

	for _, r := range referrers {
		if err := stream.Send(&storev1.PullReferrerResponse{Referrer: r}); err != nil {
			return err //nolint:wrapcheck
		}
	}

	return nil
}

func (s *mockStoreServer) DeleteReferrer(stream storev1.StoreService_DeleteReferrerServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err //nolint:wrapcheck
	}

	return stream.Send(&storev1.DeleteReferrerResponse{ //nolint:wrapcheck
		ReferrerRefs: []*corev1.ReferrerRef{req.GetReferrerRef()},
	})
}

// --- mock signv1.SignServiceServer ---

type mockSignServer struct {
	signv1.UnimplementedSignServiceServer
}

func (s *mockSignServer) Sign(_ context.Context, _ *signv1.SignRequest) (*signv1.SignResponse, error) {
	return &signv1.SignResponse{Signature: &signv1.Signature{Algorithm: "ecdsa"}}, nil
}

func (s *mockSignServer) Verify(_ context.Context, _ *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	return &signv1.VerifyResponse{Success: true}, nil
}

// --- test harness ---

// startTestServer starts a real gRPC server on 127.0.0.1:0 with mock
// implementations of every service the wrapper touches, and returns its
// address plus a cleanup function that stops the server.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	storev1.RegisterStoreServiceServer(grpcServer, &mockStoreServer{})
	signv1.RegisterSignServiceServer(grpcServer, &mockSignServer{})

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	cleanup := func() {
		grpcServer.Stop()
		_ = lis.Close()
	}

	return lis.Addr().String(), cleanup
}

// newTestClientHandle calls the wrapper's own NewClient C export against
// addr with an insecure auth mode, registers a t.Cleanup to close the
// resulting handle, and returns the handle for use by other exported
// functions in the test.
func newTestClientHandle(t *testing.T, addr string) int64 {
	t.Helper()

	cfgJSON := fmt.Sprintf(`{"server_address":%q,"auth_mode":"insecure"}`, addr)

	resp := bridgeNoHandle(NewClient, cfgJSON)

	var out struct {
		Handle int64  `json:"handle"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal([]byte(resp), &out); err != nil {
		t.Fatalf("failed to parse NewClient response: %v", err)
	}

	if out.Error != "" {
		t.Fatalf("NewClient failed: %s", out.Error)
	}

	t.Cleanup(func() {
		bridgeHandleOnly(CloseClient, out.Handle)
	})

	return out.Handle
}
