// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	searchv1 "github.com/agntcy/dir/api/search/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Search record summaries", func() {
	ginkgo.It("pages distinct indexed records without OCI content", func(ctx ginkgo.SpecContext) {
		gomega.Expect(t.db.Create(&gormdb.Record{
			RecordCID: "summary-alpha", Name: "alpha", Version: "1.0.0", OASFCreatedAt: "2026-10-01T00:00:00Z",
			Skills:  []gormdb.Skill{{Name: "review-one"}, {Name: "review-two"}},
			Domains: []gormdb.Domain{{Name: "finance"}},
			Modules: []gormdb.Module{{Name: "integration/mcp"}},
		}).Error).To(gomega.Succeed())
		gomega.Expect(t.db.Create(&gormdb.Record{
			RecordCID: "summary-bravo", Name: "bravo", Version: "1.0.0",
			Skills: []gormdb.Skill{{Name: "review-one"}},
		}).Error).To(gomega.Succeed())
		gomega.Expect(t.db.Create(&gormdb.RecordUsageMetrics{
			RecordCID: "summary-bravo", PullCount: 2,
		}).Error).To(gomega.Succeed())

		client := searchv1.NewSearchServiceClient(t.conn)
		query := []*searchv1.RecordQuery{{
			Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME, Value: "review-*",
		}}

		first, err := client.ListRecordSummaries(ctx, &searchv1.ListRecordSummariesRequest{
			Queries: query, Limit: new(uint32(1)), SortMode: searchv1.SortMode_SORT_MODE_POPULARITY,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(first.GetRecords()).To(gomega.HaveLen(1))
		gomega.Expect(first.GetRecords()[0].GetCid()).To(gomega.Equal("summary-bravo"))
		gomega.Expect(first.GetHasMore()).To(gomega.BeTrue())
		gomega.Expect(first.GetTotalCount()).To(gomega.Equal(uint32(2)))

		second, err := client.ListRecordSummaries(ctx, &searchv1.ListRecordSummariesRequest{
			Queries: query, Limit: new(uint32(1)), Offset: new(uint32(1)),
			SortMode: searchv1.SortMode_SORT_MODE_POPULARITY,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(second.GetRecords()).To(gomega.HaveLen(1))
		gomega.Expect(second.GetRecords()[0].GetCid()).To(gomega.Equal("summary-alpha"))
		gomega.Expect(second.GetRecords()[0].GetCreatedAt()).To(gomega.Equal("2026-10-01T00:00:00Z"))
		gomega.Expect(second.GetRecords()[0].GetSkills()).To(gomega.Equal([]string{"review-one", "review-two"}))
		gomega.Expect(second.GetRecords()[0].GetDomains()).To(gomega.Equal([]string{"finance"}))
		gomega.Expect(second.GetRecords()[0].GetModules()).To(gomega.Equal([]string{"integration/mcp"}))
		gomega.Expect(second.GetHasMore()).To(gomega.BeFalse())
		gomega.Expect(second.GetTotalCount()).To(gomega.Equal(uint32(2)))
	})
})
