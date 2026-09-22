// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"slices"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2"
)

// Deletion is the one referrer operation that cannot work from the CID alone: a referrer CID
// addresses its blob, not its manifest. These tests assert the referrer is actually gone rather
// than that the call returned no error, because every failure mode in this path reports success.

// referrerStoreFixture returns a local store holding one record to attach referrers to.
func referrerStoreFixture(t *testing.T) (*store, string) {
	t.Helper()

	s, ok := loadLocalStore(t).(*store)
	require.True(t, ok, "local store should be *store")

	ref, err := s.Push(testCtx, corev1.New(&typesv1alpha1.Record{
		Name:          "referrer-test-agent",
		SchemaVersion: "0.7.0",
	}))
	require.NoError(t, err, "push record")

	return s, ref.GetCid()
}

// attachReferrer pushes a scan-report referrer whose payload is unique to payload, so that each
// referrer gets a distinct CID.
func attachReferrer(t *testing.T, s *store, recordCID, payload string) string {
	t.Helper()

	ref, err := s.PushReferrer(testCtx, recordCID, &corev1.RecordReferrer{
		Type:        corev1.ScanReportReferrerType,
		RecordRef:   &corev1.RecordRef{Cid: recordCID},
		Annotations: map[string]string{"payload": payload},
	})
	require.NoError(t, err, "push referrer")
	require.NotEmpty(t, ref.GetCid(), "pushed referrer should have a CID")

	return ref.GetCid()
}

// walkCIDs returns the sorted CIDs of the record's scan-report referrers.
func walkCIDs(t *testing.T, s *store, recordCID string) []string {
	t.Helper()

	cids := []string{}

	err := s.WalkReferrers(testCtx, recordCID, corev1.ScanReportReferrerType,
		func(ref *corev1.RecordReferrer) error {
			cids = append(cids, ref.GetReferrerRef().GetCid())

			return nil
		},
	)
	require.NoError(t, err, "walk referrers")

	slices.Sort(cids)

	return cids
}

func sorted(cids ...string) []string {
	slices.Sort(cids)

	return cids
}

// A referrer is reached through its subject's Referrers API, so it carries no tag of its own. A
// tag would add nothing to discovery while listing every referrer alongside the records in the
// registry's tag list and index.
func TestPushReferrer_DoesNotTagTheReferrer(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	cid := attachReferrer(t, s, recordCID, "untagged")

	_, err := s.repo.Resolve(testCtx, cid)
	require.Error(t, err, "referrer CID should not resolve as a tag")

	_, err = s.repo.Resolve(testCtx, recordCID)
	require.NoError(t, err, "the record itself stays tagged")

	// Discovery and deletion go through the subject, so both still work untagged.
	assert.Equal(t, []string{cid}, walkCIDs(t, s, recordCID))

	deleted, err := s.DeleteReferrer(testCtx, recordCID, cid, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrer")
	assert.Equal(t, []string{cid}, deleted)
	assert.Empty(t, walkCIDs(t, s, recordCID), "referrer should be gone")
}

func TestDeleteReferrer_RemovesTheReferrer(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	cid := attachReferrer(t, s, recordCID, "only")
	require.Equal(t, []string{cid}, walkCIDs(t, s, recordCID), "referrer should be attached")

	deleted, err := s.DeleteReferrer(testCtx, recordCID, cid, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrer")
	assert.Equal(t, []string{cid}, deleted)

	assert.Empty(t, walkCIDs(t, s, recordCID), "referrer should be gone")
}

func TestDeleteReferrer_LeavesTheOthers(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	first := attachReferrer(t, s, recordCID, "first")
	second := attachReferrer(t, s, recordCID, "second")
	third := attachReferrer(t, s, recordCID, "third")

	deleted, err := s.DeleteReferrer(testCtx, recordCID, second, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrer")
	assert.Equal(t, []string{second}, deleted)

	assert.Equal(t, sorted(first, third), walkCIDs(t, s, recordCID))
}

func TestDeleteReferrers_DeletesOnlyTheNamedOnes(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	first := attachReferrer(t, s, recordCID, "first")
	second := attachReferrer(t, s, recordCID, "second")
	third := attachReferrer(t, s, recordCID, "third")

	deleted, err := s.DeleteReferrers(testCtx, recordCID,
		[]string{first, third}, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrers")
	assert.Equal(t, sorted(first, third), sorted(deleted...))

	assert.Equal(t, []string{second}, walkCIDs(t, s, recordCID))
}

// An empty set must mean "delete nothing". Reading it as "delete everything of this type" would
// let a caller that computed no candidates clear the record's referrers.
func TestDeleteReferrers_EmptyDeletesNothing(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	first := attachReferrer(t, s, recordCID, "first")
	second := attachReferrer(t, s, recordCID, "second")

	deleted, err := s.DeleteReferrers(testCtx, recordCID, nil, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrers")
	assert.Empty(t, deleted)

	assert.Equal(t, sorted(first, second), walkCIDs(t, s, recordCID))
}

// Deleting with an empty CID stays the explicit way to clear a type.
func TestDeleteReferrer_EmptyCIDDeletesEveryReferrerOfType(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	attachReferrer(t, s, recordCID, "first")
	attachReferrer(t, s, recordCID, "second")

	deleted, err := s.DeleteReferrer(testCtx, recordCID, "", corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrers")
	assert.Len(t, deleted, 2)

	assert.Empty(t, walkCIDs(t, s, recordCID))
}

// Referrers pushed before the CID annotation existed carry no CID of their own. The CID addresses
// the referrer blob, so it can be recomputed - without that, such a referrer has no identity and
// deletion and pull-by-CID skip it in silence.
func TestWalkReferrers_DerivesCIDWhenAnnotationIsMissing(t *testing.T) {
	s, recordCID := referrerStoreFixture(t)

	recordDesc, err := s.repo.Resolve(testCtx, recordCID)
	require.NoError(t, err, "resolve record")

	payload, err := (&corev1.RecordReferrer{
		Type:      corev1.ScanReportReferrerType,
		RecordRef: &corev1.RecordRef{Cid: recordCID},
	}).Marshal()
	require.NoError(t, err, "marshal referrer")

	blobDesc, err := oras.PushBytes(testCtx, s.repo, DefaultReferrerArtifactMediaType, payload)
	require.NoError(t, err, "push referrer blob")

	// No CID annotation, reproducing a referrer written before that annotation was introduced.
	_, err = oras.PackManifest(testCtx, s.repo, oras.PackManifestVersion1_1,
		ocispec.MediaTypeImageManifest, oras.PackManifestOptions{
			Subject: &recordDesc,
			ManifestAnnotations: map[string]string{
				corev1.ReferrerTypeAnnotationKey: corev1.ScanReportReferrerType,
			},
			Layers: []ocispec.Descriptor{blobDesc},
		})
	require.NoError(t, err, "pack referrer manifest")

	wantCID, err := corev1.ConvertDigestToCID(blobDesc.Digest)
	require.NoError(t, err, "derive CID")

	assert.Equal(t, []string{wantCID}, walkCIDs(t, s, recordCID),
		"walk should report the CID derived from the blob digest")

	deleted, err := s.DeleteReferrer(testCtx, recordCID, wantCID, corev1.ScanReportReferrerType)
	require.NoError(t, err, "delete referrer")
	assert.Equal(t, []string{wantCID}, deleted, "an untagged, unannotated referrer must be deletable")

	assert.Empty(t, walkCIDs(t, s, recordCID))
}
