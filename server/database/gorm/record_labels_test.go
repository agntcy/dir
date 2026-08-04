// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestGetRecordLabelsMatchesTheRecord guards the seam between the two ways a
// record's labels reach the DHT. Publish advertises what
// types.GetLabelsFromRecord derives from the record; every reprovide afterwards
// advertises what this method reads back from the index. If they drift, a
// record is announced once and then silently goes dark when its provider
// records expire.
func TestGetRecordLabelsMatchesTheRecord(t *testing.T) {
	gdb, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	require.NoError(t, err)

	db := &DB{gormDB: gdb}
	require.NoError(t, db.migrate())

	record := corev1.New(&typesv1alpha1.Record{
		Name:          "label-parity",
		SchemaVersion: "0.7.0",
		Skills: []*typesv1alpha1.Skill{
			{Name: "AI/ML"},
			{Name: "AI/NLP"},
		},
		Domains: []*typesv1alpha1.Domain{
			{Name: "healthcare"},
		},
		Modules: []*typesv1alpha1.Module{
			{Name: "runtime/python"},
		},
		Locators: []*typesv1alpha1.Locator{
			{Type: "docker-image", Url: "https://example.test/image"},
		},
	})

	adapter, err := record.Decode()
	require.NoError(t, err)
	require.NoError(t, db.AddRecord(adapter))

	fromIndex, err := db.GetRecordLabels([]string{record.GetCid()})
	require.NoError(t, err)

	assert.ElementsMatch(t,
		types.GetLabelsFromRecord(adapter),
		fromIndex[record.GetCid()],
		"labels read back from the index must match the ones Publish advertises")
}
