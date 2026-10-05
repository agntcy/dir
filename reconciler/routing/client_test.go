// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufSize = 1024 * 1024

	// waitFor bounds the waits on a stream that should end on its own, so a
	// test fails rather than hangs.
	waitFor = 5 * time.Second
)

// stubServer answers from the values it is given. The tests run it behind a
// real gRPC server so they exercise the actual stream: EOF, status errors and
// cancellation behave as they do in production, which a hand-written
// RoutingServiceClient could only approximate.
type stubServer struct {
	routingv1.UnimplementedRoutingServiceServer

	count    uint32
	countErr error

	// hits are sent in order before searchErr, if any, ends the stream.
	hits      []*routingv1.SearchResponse
	searchErr error

	// endless keeps the stream sending until its context ends, so a test can
	// cancel mid-flight.
	endless bool

	// gotSearch receives the request the server was called with.
	gotSearch chan *routingv1.SearchRequest
}

func (s *stubServer) GetProviderCount(_ context.Context, _ *routingv1.GetProviderCountRequest) (*routingv1.GetProviderCountResponse, error) {
	if s.countErr != nil {
		return nil, s.countErr
	}

	return &routingv1.GetProviderCountResponse{Count: s.count}, nil
}

func (s *stubServer) Search(req *routingv1.SearchRequest, srv routingv1.RoutingService_SearchServer) error {
	if s.gotSearch != nil {
		s.gotSearch <- req
	}

	for _, response := range s.hits {
		if err := srv.Send(response); err != nil {
			return fmt.Errorf("send hit: %w", err)
		}
	}

	for s.endless {
		select {
		case <-srv.Context().Done():
			return nil
		default:
		}

		if err := srv.Send(hit("endless", "peer-endless")); err != nil {
			return fmt.Errorf("send endless hit: %w", err)
		}
	}

	return s.searchErr
}

func hit(cid, peerID string) *routingv1.SearchResponse {
	return &routingv1.SearchResponse{
		RecordRef: &corev1.RecordRef{Cid: cid},
		Peer:      &routingv1.Peer{Id: peerID},
	}
}

// newTestClient serves srv over bufconn and returns a Client dialed to it
// through New, so the constructor is covered too.
func newTestClient(t *testing.T, srv routingv1.RoutingServiceServer) *Client {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	routingv1.RegisterRoutingServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	// grpc.NewClient resolves the target through DNS unless told otherwise, so
	// the bufconn dialer is only reached via the passthrough scheme.
	client, err := New("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()

		grpcServer.Stop()

		_ = lis.Close()
	})

	return client
}

// collect drains ch until it closes, failing the test if that takes too long.
func collect(t *testing.T, ch <-chan *routingv1.SearchResponse) []*routingv1.SearchResponse {
	t.Helper()

	var got []*routingv1.SearchResponse

	deadline := time.After(waitFor)

	for {
		select {
		case resp, open := <-ch:
			if !open {
				return got
			}

			got = append(got, resp)
		case <-deadline:
			t.Fatal("search channel did not close")
		}
	}
}

func cids(responses []*routingv1.SearchResponse) []string {
	out := make([]string, 0, len(responses))
	for _, resp := range responses {
		out = append(out, resp.GetRecordRef().GetCid())
	}

	return out
}

// --- New / Close ---

func TestNew_WithoutDialOptionsFallsBackToInsecure(t *testing.T) {
	t.Parallel()

	// grpc.NewClient does not connect, so this only asserts the constructor
	// accepts a bare address and produces a usable, closable client.
	client, err := New("localhost:1")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NoError(t, client.Close())
}

func TestNew_UnparseableAddressReturnsError(t *testing.T) {
	t.Parallel()

	// A control character makes the target unparseable as a URL, which is one
	// of the few things grpc.NewClient rejects up front rather than on the
	// first RPC.
	_, err := New("host\t:8888")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to apiserver")
}

func TestClose_IsANoOpForAClientBuiltFromAnExistingConnection(t *testing.T) {
	t.Parallel()

	// The caller owns the connection in this case, so Close must not touch it.
	client := NewFromClient(nil)
	assert.NoError(t, client.Close())
	assert.NoError(t, client.Close())
}

// --- GetProviderCount ---

func TestGetProviderCount_ReturnsTheCount(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{count: 7})

	count, err := client.GetProviderCount(t.Context(), "cid-1")
	require.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestGetProviderCount_WrapsTheRPCErrorWithTheCID(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{
		countErr: status.Error(codes.Unavailable, "routing down"),
	})

	count, err := client.GetProviderCount(t.Context(), "cid-1")
	require.Error(t, err)
	assert.Zero(t, count)
	assert.Contains(t, err.Error(), "cid-1")
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

// --- Search ---

func TestSearch_StreamsEveryResultThenClosesTheChannel(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{
		hits: []*routingv1.SearchResponse{
			hit("cid-1", "peer-a"),
			hit("cid-2", "peer-a"),
			hit("cid-3", "peer-b"),
		},
	})

	ch, err := client.Search(t.Context(), &routingv1.SearchRequest{})
	require.NoError(t, err)

	assert.Equal(t, []string{"cid-1", "cid-2", "cid-3"}, cids(collect(t, ch)))
}

func TestSearch_NoResultsClosesTheChannelImmediately(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{})

	ch, err := client.Search(t.Context(), &routingv1.SearchRequest{})
	require.NoError(t, err)

	assert.Empty(t, collect(t, ch))
}

func TestSearch_SendsTheRequestUnchanged(t *testing.T) {
	t.Parallel()

	limit := uint32(25)
	minScore := uint32(1)
	server := &stubServer{gotSearch: make(chan *routingv1.SearchRequest, 1)}
	client := newTestClient(t, server)

	ch, err := client.Search(t.Context(), &routingv1.SearchRequest{
		Queries: []*routingv1.RecordQuery{{
			Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN,
			Value: "network_security",
		}},
		Limit:         &limit,
		MinMatchScore: &minScore,
	})
	require.NoError(t, err)
	collect(t, ch)

	var got *routingv1.SearchRequest

	select {
	case got = <-server.gotSearch:
	case <-time.After(waitFor):
		t.Fatal("server was not called")
	}

	require.Len(t, got.GetQueries(), 1)
	assert.Equal(t, routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, got.GetQueries()[0].GetType())
	assert.Equal(t, "network_security", got.GetQueries()[0].GetValue())
	assert.Equal(t, uint32(25), got.GetLimit())
	assert.Equal(t, uint32(1), got.GetMinMatchScore())
}

// A stream that fails part way through is reported by closing the channel,
// which is what a complete stream does too. The consumer therefore cannot tell
// a truncated result set from a whole one and will act on partial results. The
// error is only logged. Change this test with the fix.
func TestSearch_MidStreamErrorIsIndistinguishableFromCompletion(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{
		hits:      []*routingv1.SearchResponse{hit("cid-1", "peer-a")},
		searchErr: status.Error(codes.Internal, "datastore iteration failed"),
	})

	ch, err := client.Search(t.Context(), &routingv1.SearchRequest{})
	require.NoError(t, err, "the error arrives on the stream, not at call time")

	assert.Equal(t, []string{"cid-1"}, cids(collect(t, ch)),
		"the hit sent before the failure is delivered, then the channel closes silently")
}

func TestSearch_CancellingTheContextEndsTheStream(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{endless: true})

	ctx, cancel := context.WithCancel(t.Context())

	ch, err := client.Search(ctx, &routingv1.SearchRequest{})
	require.NoError(t, err)

	// Take one result to be sure the stream is flowing, then cancel.
	select {
	case <-ch:
	case <-time.After(waitFor):
		cancel()
		t.Fatal("no results before cancel")
	}

	cancel()

	// The channel closes once the goroutine notices, so a consumer that stops
	// reading does not strand it.
	collect(t, ch)
}

func TestSearch_OnAClosedConnectionFailsAtCallTime(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, &stubServer{})
	require.NoError(t, client.Close())

	_, err := client.Search(t.Context(), &routingv1.SearchRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create search stream")
}
