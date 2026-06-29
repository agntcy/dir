// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"slices"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Usage metrics and popularity sort", ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		refA *corev1.RecordRef // v070 record — will receive more pulls
		refB *corev1.RecordRef // v080 record — will receive fewer pulls
	)

	ginkgo.BeforeAll(func(ctx context.Context) {
		recordA, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		refA, err = testEnv.Client.Push(ctx, recordA)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Small delay so B gets a strictly later created_at in the DB,
		// making the SORT_MODE_RECENCY assertion deterministic.
		time.Sleep(50 * time.Millisecond)

		recordB, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV080V4JSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		refB, err = testEnv.Client.Push(ctx, recordB)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func(ctx context.Context) {
		if refA != nil {
			_ = testEnv.Client.Delete(ctx, refA)
		}

		if refB != nil {
			_ = testEnv.Client.Delete(ctx, refB)
		}
	})

	ginkgo.It("should accumulate pull counts: A×3, B×1", func(ctx context.Context) {
		for range 3 {
			_, err := testEnv.Client.Pull(ctx, refA)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}

		_, err := testEnv.Client.Pull(ctx, refB)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.It("should accumulate lookup counts: A×2", func(ctx context.Context) {
		for range 2 {
			_, err := testEnv.Client.Lookup(ctx, refA)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}
	})

	ginkgo.It("SORT_MODE_POPULARITY should rank A before B", func(ctx context.Context) {
		// Both v070 and v080 share skill ID 10201.
		cids, err := searchCIDs(ctx, &searchv1.SearchCIDsRequest{
			Queries: []*searchv1.RecordQuery{
				{
					Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
					Value: "10201",
				},
			},
			SortMode: searchv1.SortMode_SORT_MODE_POPULARITY,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cids).To(gomega.ContainElements(refA.GetCid(), refB.GetCid()))

		idxA := slices.Index(cids, refA.GetCid())
		idxB := slices.Index(cids, refB.GetCid())
		gomega.Expect(idxA).To(gomega.BeNumerically("<", idxB),
			"A (pull_count=3, lookup_count=2) should rank above B (pull_count=1, lookup_count=0)")
	})

	ginkgo.It("SORT_MODE_RECENCY should rank B before A", func(ctx context.Context) {
		cids, err := searchCIDs(ctx, &searchv1.SearchCIDsRequest{
			Queries: []*searchv1.RecordQuery{
				{
					Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID,
					Value: "10201",
				},
			},
			SortMode: searchv1.SortMode_SORT_MODE_RECENCY,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cids).To(gomega.ContainElements(refA.GetCid(), refB.GetCid()))

		idxA := slices.Index(cids, refA.GetCid())
		idxB := slices.Index(cids, refB.GetCid())
		gomega.Expect(idxB).To(gomega.BeNumerically("<", idxA),
			"B (pushed after A) should appear first under recency sort")
	})
})

// searchCIDs collects all CIDs from a SearchCIDs streaming response.
func searchCIDs(ctx context.Context, req *searchv1.SearchCIDsRequest) ([]string, error) {
	result, err := testEnv.Client.SearchCIDs(ctx, req)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return collectCIDs(result), nil
}

// collectCIDs drains a SearchCIDs StreamResult into an ordered slice of CID strings.
func collectCIDs(result streaming.StreamResult[searchv1.SearchCIDsResponse]) []string {
	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			if resp != nil {
				cids = append(cids, resp.GetRecordCid())
			}
		case <-result.ErrCh():
			// Ignore per-item errors; the caller checks the final slice.
		case <-result.DoneCh():
			return cids
		}
	}
}
