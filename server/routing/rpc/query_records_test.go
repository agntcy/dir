// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agntcy/dir/server/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeQueryDB implements only the two methods QueryRecords needs; the rest of
// DatabaseAPI is inherited as nil and would panic if ever touched, which is the
// point.
type fakeQueryDB struct {
	types.DatabaseAPI

	// cidsByFilter is consulted with the SkillNames/DomainNames/ModuleNames/
	// LocatorTypes the handler derived, so tests can assert on translation.
	respond func(filters *types.RecordFilters) ([]string, error)

	labels    map[string][]types.Label
	labelsErr error

	appliedFilters []*types.RecordFilters
}

func (f *fakeQueryDB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	filters := &types.RecordFilters{}
	for _, opt := range opts {
		opt(filters)
	}

	f.appliedFilters = append(f.appliedFilters, filters)

	return f.respond(filters)
}

func (f *fakeQueryDB) GetRecordLabels(cids []string) (map[string][]types.Label, error) {
	if f.labelsErr != nil {
		return nil, f.labelsErr
	}

	result := make(map[string][]types.Label, len(cids))

	for _, cid := range cids {
		if labels, ok := f.labels[cid]; ok {
			result[cid] = labels
		}
	}

	return result, nil
}

func newTestHost(t *testing.T) host.Host {
	t.Helper()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)

	t.Cleanup(func() { _ = h.Close() })

	return h
}

// newConnectedServices returns a client service and the peer ID of a server
// service backed by db, already dialled.
func newConnectedServices(t *testing.T, db types.DatabaseAPI) (*Service, peer.ID) {
	t.Helper()

	serverHost := newTestHost(t)
	clientHost := newTestHost(t)

	_, err := New(serverHost, nil, db)
	require.NoError(t, err)

	clientService, err := New(clientHost, nil, nil)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	require.NoError(t, clientHost.Connect(ctx, peer.AddrInfo{
		ID:    serverHost.ID(),
		Addrs: serverHost.Addrs(),
	}))

	return clientService, serverHost.ID()
}

// collect runs a query against the peer and returns its matches.
func collect(t *testing.T, client *Service, peerID peer.ID, req QueryRecordsRequest) ([]RecordMatch, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	return client.QueryRecords(ctx, peerID, &req)
}

// The method has to be accepted by gorpc's reflection-based registration and
// survive the msgpack round trip; only a real call across two hosts proves it.
func TestQueryRecordsReturnsMatchesAcrossPeers(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return []string{"cid-a", "cid-b"}, nil
		},
		labels: map[string][]types.Label{
			"cid-a": {"/skills/AI/ML", "/domains/finance"},
			"cid-b": {"/skills/AI/NLP"},
		},
	}

	client, serverID := newConnectedServices(t, db)

	matches, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{{Type: "skills", Value: "AI"}},
	})
	require.NoError(t, err)

	require.Len(t, matches, 2)
	assert.Equal(t, "cid-a", matches[0].Cid)
	assert.Equal(t, []string{"/skills/AI/ML", "/domains/finance"}, matches[0].Labels)
	assert.Equal(t, "cid-b", matches[1].Cid)
	assert.Equal(t, []string{"/skills/AI/NLP"}, matches[1].Labels)
}

// A record matching several queries must be returned once.
func TestQueryRecordsUnionsQueriesWithoutDuplicates(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(filters *types.RecordFilters) ([]string, error) {
			if len(filters.SkillNames) > 0 {
				return []string{"shared", "skill-only"}, nil
			}

			return []string{"shared", "domain-only"}, nil
		},
		labels: map[string][]types.Label{
			"shared":      {"/skills/AI", "/domains/finance"},
			"skill-only":  {"/skills/AI"},
			"domain-only": {"/domains/finance"},
		},
	}

	client, serverID := newConnectedServices(t, db)

	matches, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{
			{Type: "skills", Value: "AI"},
			{Type: "domains", Value: "finance"},
		},
	})
	require.NoError(t, err)

	cids := make([]string, len(matches))
	for i, match := range matches {
		cids[i] = match.Cid
	}

	assert.Equal(t, []string{"shared", "skill-only", "domain-only"}, cids)
}

func TestQueryRecordsCapsResultsAtLimit(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return []string{"a", "b", "c", "d"}, nil
		},
		labels: map[string][]types.Label{
			"a": {"/skills/AI"},
			"b": {"/skills/AI"},
			"c": {"/skills/AI"},
			"d": {"/skills/AI"},
		},
	}

	client, serverID := newConnectedServices(t, db)

	matches, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{{Type: "skills", Value: "AI"}},
		Limit:   2,
	})
	require.NoError(t, err)

	assert.Len(t, matches, 2)
	require.NotEmpty(t, db.appliedFilters)
	assert.Equal(t, 2, db.appliedFilters[0].Limit, "the limit must reach the database, not just the result loop")
}

// A caller needs to know its view of a peer is partial.
func TestQueryRecordsReportsTruncation(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return []string{"a", "b", "c"}, nil
		},
		labels: map[string][]types.Label{
			"a": {"/skills/AI"},
			"b": {"/skills/AI"},
			"c": {"/skills/AI"},
		},
	}

	client, serverID := newConnectedServices(t, db)

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()

	var resp QueryRecordsResponse

	require.NoError(t, client.rpcClient.CallContext(ctx, serverID, DirService, DirServiceFuncQueryRecords,
		&QueryRecordsRequest{Queries: []RecordQuery{{Type: "skills", Value: "AI"}}, Limit: 2},
		&resp,
	))

	assert.True(t, resp.Truncated)
	assert.Len(t, resp.Records, 2)
}

// Records whose labels disappeared between the two queries carry nothing to
// score against, so they are dropped rather than streamed empty.
func TestQueryRecordsSkipsRecordsWithoutLabels(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return []string{"present", "vanished"}, nil
		},
		labels: map[string][]types.Label{
			"present": {"/skills/AI"},
		},
	}

	client, serverID := newConnectedServices(t, db)

	matches, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{{Type: "skills", Value: "AI"}},
	})
	require.NoError(t, err)

	require.Len(t, matches, 1)
	assert.Equal(t, "present", matches[0].Cid)
}

func TestQueryRecordsRejectsEmptyQueryList(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			t.Error("the database must not be consulted for an empty query list")

			return nil, nil
		},
	}

	client, serverID := newConnectedServices(t, db)

	_, err := collect(t, client, serverID, QueryRecordsRequest{})
	require.Error(t, err)
}

func TestQueryRecordsFailsWithoutDatabase(t *testing.T) {
	t.Parallel()

	client, serverID := newConnectedServices(t, nil)

	_, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{{Type: "skills", Value: "AI"}},
	})
	require.Error(t, err)
}

func TestQueryRecordsSurfacesDatabaseFailure(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return nil, errors.New("database is down")
		},
	}

	client, serverID := newConnectedServices(t, db)

	_, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{{Type: "skills", Value: "AI"}},
	})
	require.Error(t, err)
}

// Unusable queries are skipped, not fatal, so one bad entry does not sink an
// otherwise valid multi-query search.
func TestQueryRecordsSkipsUnusableQueries(t *testing.T) {
	t.Parallel()

	db := &fakeQueryDB{
		respond: func(*types.RecordFilters) ([]string, error) {
			return []string{"cid-a"}, nil
		},
		labels: map[string][]types.Label{"cid-a": {"/skills/AI"}},
	}

	client, serverID := newConnectedServices(t, db)

	matches, err := collect(t, client, serverID, QueryRecordsRequest{
		Queries: []RecordQuery{
			{Type: "nonsense", Value: "AI"},
			{Type: "skills", Value: "  "},
			{Type: "skills", Value: "AI"},
		},
	})
	require.NoError(t, err)

	require.Len(t, matches, 1)
	assert.Len(t, db.appliedFilters, 1, "only the usable query should reach the database")
}

func TestQueryRecordsRejectsNilRequest(t *testing.T) {
	t.Parallel()

	client, serverID := newConnectedServices(t, nil)

	_, err := client.QueryRecords(t.Context(), serverID, nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestQueryFiltersExpandsHierarchicalNamespaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		query    RecordQuery
		expected func(*types.RecordFilters) []string
	}{
		{
			name:     "skills match the value or any descendant",
			query:    RecordQuery{Type: "skills", Value: "AI"},
			expected: func(f *types.RecordFilters) []string { return f.SkillNames },
		},
		{
			name:     "domains match the value or any descendant",
			query:    RecordQuery{Type: "domains", Value: "AI"},
			expected: func(f *types.RecordFilters) []string { return f.DomainNames },
		},
		{
			name:     "modules match the value or any descendant",
			query:    RecordQuery{Type: "modules", Value: "AI"},
			expected: func(f *types.RecordFilters) []string { return f.ModuleNames },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			options, err := queryFilters(tt.query, 10)
			require.NoError(t, err)

			filters := &types.RecordFilters{}
			for _, option := range options {
				option(filters)
			}

			assert.Equal(t, []string{"AI", "AI/*"}, tt.expected(filters))
			assert.Equal(t, 10, filters.Limit)
		})
	}
}

// Locators are flat, so a descendant pattern would be meaningless.
func TestQueryFiltersMatchesLocatorsExactly(t *testing.T) {
	t.Parallel()

	options, err := queryFilters(RecordQuery{Type: "locators", Value: "docker-image"}, 10)
	require.NoError(t, err)

	filters := &types.RecordFilters{}
	for _, option := range options {
		option(filters)
	}

	assert.Equal(t, []string{"docker-image"}, filters.LocatorTypes)
	assert.Empty(t, filters.SkillNames)
}

func TestQueryFiltersRejectsBadQueries(t *testing.T) {
	t.Parallel()

	for _, query := range []RecordQuery{
		{Type: "skills", Value: ""},
		{Type: "skills", Value: "   "},
		{Type: "", Value: "AI"},
		{Type: "unknown", Value: "AI"},
	} {
		_, err := queryFilters(query, 10)
		require.Error(t, err, "expected %+v to be rejected", query)
	}
}

func TestGetDatabaseReportsUnimplementedWhenAbsent(t *testing.T) {
	t.Parallel()

	_, err := (&Service{}).getDatabase()
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}
