// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSearcher struct {
	req  *routingv1.SearchRequest
	hits []*routingv1.SearchResponse
	err  error
}

func (f *fakeSearcher) Search(_ context.Context, req *routingv1.SearchRequest) (<-chan *routingv1.SearchResponse, error) {
	f.req = req
	if f.err != nil {
		return nil, f.err
	}

	ch := make(chan *routingv1.SearchResponse, len(f.hits))
	for _, hit := range f.hits {
		ch <- hit
	}

	close(ch)

	return ch, nil
}

type createdSync struct {
	remoteURL         string
	cids              []string
	remoteRegistryURL string
	repositoryName    string
}

type fakeSyncs struct {
	created []createdSync
	err     error
}

func (f *fakeSyncs) CreateSync(remoteURL string, cids []string, remoteRegistryURL, repositoryName string) (string, error) {
	if f.err != nil {
		return "", f.err
	}

	f.created = append(f.created, createdSync{
		remoteURL:         remoteURL,
		cids:              append([]string(nil), cids...),
		remoteRegistryURL: remoteRegistryURL,
		repositoryName:    repositoryName,
	})

	return "sync-1", nil
}

func searchHit(peerID, cid string, addrs ...string) *routingv1.SearchResponse {
	return &routingv1.SearchResponse{
		RecordRef: &corev1.RecordRef{Cid: cid},
		Peer:      &routingv1.Peer{Id: peerID, Addrs: addrs},
	}
}

func TestTask_Name_Interval_IsEnabled(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true, Interval: 2 * time.Hour}, nil, nil)
	require.NoError(t, err)

	assert.Equal(t, "sync", task.Name())
	assert.Equal(t, 2*time.Hour, task.Interval())
	assert.True(t, task.IsEnabled())
}

func TestTask_IsEnabled_False(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: false}, nil, nil)
	require.NoError(t, err)
	assert.False(t, task.IsEnabled())
}

func TestTask_Interval_Zero_UsesDefault(t *testing.T) {
	t.Parallel()

	task, err := NewTask(Config{Enabled: true}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, DefaultInterval, task.Interval())
}

func TestTask_Run_NoHits_ReturnsNil(t *testing.T) {
	t.Parallel()

	searcher := &fakeSearcher{}
	db := &fakeSyncs{}
	task, err := NewTask(Config{Enabled: true}, searcher, db)
	require.NoError(t, err)

	assert.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
	require.NotNil(t, searcher.req)
	require.Len(t, searcher.req.GetQueries(), 1)
	assert.Equal(t, routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, searcher.req.GetQueries()[0].GetType())
	assert.Equal(t, DefaultDomain, searcher.req.GetQueries()[0].GetValue())
	assert.Equal(t, uint32(DefaultLimit), searcher.req.GetLimit())
	assert.Equal(t, uint32(1), searcher.req.GetMinMatchScore())
}

func TestTask_Run_SearchError_ReturnsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("routing unavailable")
	task, err := NewTask(Config{Enabled: true}, &fakeSearcher{err: wantErr}, &fakeSyncs{})
	require.NoError(t, err)

	err = task.Run(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, err.Error(), "search routing")
}

func TestTask_Run_DryRun_DoesNotCreateSync(t *testing.T) {
	t.Parallel()

	db := &fakeSyncs{}
	task, err := NewTask(Config{Enabled: true, DryRun: true}, &fakeSearcher{
		hits: []*routingv1.SearchResponse{
			searchHit("peer-a", "cid-1", "/dir/example.com:8888"),
		},
	}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
}

func TestTask_Run_CreatesDirSyncPerPeer(t *testing.T) {
	t.Parallel()

	db := &fakeSyncs{}
	task, err := NewTask(Config{Enabled: true, Criteria: Criteria{Domain: "network_security"}, Limit: 25}, &fakeSearcher{
		hits: []*routingv1.SearchResponse{
			searchHit("peer-a", "cid-1", "/dir/a.example:8888", "/oci/ghcr.io/org"),
			searchHit("peer-a", "cid-2", "/dir/a.example:8888"),
			searchHit("peer-b", "cid-3", "/oci/registry.example/dir"),
		},
	}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	require.Len(t, db.created, 2)
	assert.Equal(t, createdSync{
		remoteURL: "a.example:8888",
		cids:      []string{"cid-1", "cid-2"},
	}, db.created[0])
	assert.Equal(t, createdSync{
		remoteRegistryURL: "registry.example",
		repositoryName:    "dir",
		cids:              []string{"cid-3"},
	}, db.created[1])
}

func TestTask_Run_CreateError_Continues(t *testing.T) {
	t.Parallel()

	db := &fakeSyncs{err: errors.New("db down")}
	task, err := NewTask(Config{Enabled: true}, &fakeSearcher{
		hits: []*routingv1.SearchResponse{
			searchHit("peer-a", "cid-1", "/dir/a.example:8888"),
		},
	}, db)
	require.NoError(t, err)

	assert.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
}

func TestTask_Run_SkipsPeerWithoutAddress(t *testing.T) {
	t.Parallel()

	db := &fakeSyncs{}
	task, err := NewTask(Config{Enabled: true}, &fakeSearcher{
		hits: []*routingv1.SearchResponse{
			searchHit("peer-a", "cid-1", "/ip4/1.2.3.4/tcp/4001"),
		},
	}, db)
	require.NoError(t, err)

	require.NoError(t, task.Run(context.Background()))
	assert.Empty(t, db.created)
}

func TestGroupByPeer_PrefersDirAndDedupsCIDs(t *testing.T) {
	t.Parallel()

	peers := groupByPeer([]*routingv1.SearchResponse{
		searchHit("peer-a", "cid-1", "/dir/a.example:8888"),
		searchHit("peer-a", "cid-1", "/dir/a.example:8888"),
		searchHit("peer-a", "cid-2", "/oci/ignored"),
		nil,
		{Peer: &routingv1.Peer{Id: "empty"}},
	})

	require.Len(t, peers, 1)
	assert.Equal(t, "peer-a", peers[0].id)
	assert.Equal(t, "a.example:8888", peers[0].dirAddr)
	assert.Equal(t, "ignored", peers[0].ociRegistry)
	assert.Equal(t, []string{"cid-1", "cid-2"}, peers[0].cids)
}

// The sync worker takes the registry and the repository as separate fields, so
// an advertised /oci/ address has to be split the same way dirctl splits it.
func TestSplitOCIAddr(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		addr       string
		registry   string
		repository string
	}{
		{addr: "ghcr.io/org", registry: "ghcr.io", repository: "org"},
		{addr: "registry.example:5000/dir", registry: "registry.example:5000", repository: "dir"},
		{addr: "ghcr.io", registry: "ghcr.io", repository: ""},
		{addr: "", registry: "", repository: ""},
	} {
		registry, repository := splitOCIAddr(tc.addr)

		assert.Equal(t, tc.registry, registry, tc.addr)
		assert.Equal(t, tc.repository, repository, tc.addr)
	}
}
