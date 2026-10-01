// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/server/datastore"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProtocolDirName(t *testing.T) {
	// Both shapes the constant has actually held, plus a deeper one. Anything
	// that assumes a trailing version segment breaks on "dir", which is what v1
	// shipped.
	tests := []struct {
		prefix string
		want   string
	}{
		{prefix: "dir", want: "dir"},
		{prefix: "/dir/2", want: "dir-2"},
		{prefix: "/dir/2/", want: "dir-2"},
		{prefix: "/agntcy/dir/3", want: "agntcy-dir-3"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			assert.Equal(t, tt.want, protocolDirName(tt.prefix))
		})
	}
}

func TestDatastoreVersionDirIsBelowTheConfiguredPath(t *testing.T) {
	got := datastoreVersionDir("/etc/routing/datastore")

	assert.Equal(t, filepath.Join("/etc/routing/datastore", protocolDirName(ProtocolPrefix)), got)
	assert.NotEqual(t, "/etc/routing/datastore", got, "the versioned path must not be the configured path itself")
}

// TestVersionedDatastoreLeavesTheParentOpenable is the property the whole
// upgrade rests on. Badger takes an exclusive flock on its directory, and
// because flock binds to the open file description rather than the process, a
// second open of the same directory fails even from inside this one. Putting v2
// a level down is what lets the migration read v1 while v2 is live.
func TestVersionedDatastoreLeavesTheParentOpenable(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()

	// Seed the parent the way v1 did: /records/<CID>, nil value.
	v1, err := datastore.New(datastore.WithFsProvider(base))
	require.NoError(t, err)
	require.NoError(t, v1.Put(ctx, ds.NewKey("/records/bafyseeded"), nil))
	require.NoError(t, v1.Close())

	// v2 opens its own subdirectory and holds it for the rest of the test.
	v2, err := datastore.New(datastore.WithFsProvider(datastoreVersionDir(base)))
	require.NoError(t, err)

	t.Cleanup(func() { _ = v2.Close() })

	// The migration opens the parent while v2 is live, and finds v1's keys.
	migration, err := datastore.New(datastore.WithFsProvider(base))
	require.NoError(t, err, "the parent must stay openable while v2 holds its subdirectory")

	results, err := migration.Query(ctx, query.Query{Prefix: "/records/", KeysOnly: true})
	require.NoError(t, err)

	entries, err := results.Rest()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "/records/bafyseeded", entries[0].Key)

	require.NoError(t, migration.Close())

	// v2 is unaffected by the migration opening and closing beneath it.
	require.NoError(t, v2.Put(ctx, ds.NewKey("/after"), []byte("ok")))
}

// TestSameDirectoryCannotBeOpenedTwice records why the subdirectory is
// required rather than merely tidy: without it, the migration cannot open the
// v1 datastore at all.
func TestSameDirectoryCannotBeOpenedTwice(t *testing.T) {
	base := t.TempDir()

	held, err := datastore.New(datastore.WithFsProvider(base))
	require.NoError(t, err)

	t.Cleanup(func() { _ = held.Close() })

	_, err = datastore.New(datastore.WithFsProvider(base))
	require.Error(t, err, "Badger must refuse a second open of a held directory")
}
