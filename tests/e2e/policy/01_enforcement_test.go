// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// settleTimeout is how long a record may take to reach its verdict and the
	// server to act on it. The testenv's tasks run once an hour, so a record is
	// indexed and evaluated only because the reconciler heard of its push: after
	// the 3s window, an indexer run and a policy run, then the server noticing
	// the policy. The timeout leaves room for a slow runner.
	settleTimeout = 2 * time.Minute

	// rejectedWindow is how long a rejected record is watched after it has had
	// its verdict, to see that it is not served later.
	rejectedWindow = 5 * time.Second

	pollInterval = time.Second

	// described is a description the testenv's policy passes.
	described = "describes what it does"

	// withheldMessage is what a caller is told about a record the node does
	// not serve under its policy, and nothing more.
	withheldMessage = "is not available under this node's content policy"

	// policyID is the policy the testenv enforces. A caller must never be told it.
	policyID = "has-description"
)

// newRecord returns a record no other test has pushed, so its CID is its own.
// The testenv's policy passes a record only if it has a description.
func newRecord(name, description string) *corev1.Record {
	ginkgo.GinkgoHelper()

	var fields map[string]any
	gomega.Expect(json.Unmarshal(testdata.ExpectedRecordV100JSON, &fields)).To(gomega.Succeed())

	fields["name"] = fmt.Sprintf("%s-%d", name, time.Now().UnixNano())
	fields["description"] = description

	data, err := json.Marshal(fields)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	record, err := corev1.UnmarshalRecord(data)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return record
}

func push(ctx context.Context, record *corev1.Record) *corev1.RecordRef {
	ginkgo.GinkgoHelper()

	ref, err := testEnv.Client.Push(ctx, record)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return ref
}

func pullCode(ctx context.Context, ref *corev1.RecordRef) codes.Code {
	_, err := testEnv.Client.Pull(ctx, ref)

	return status.Code(err)
}

// searchByName returns the CIDs of the records with this name.
func searchByName(ctx context.Context, name string) []string {
	ginkgo.GinkgoHelper()

	result, err := testEnv.Client.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
		Queries: []*searchv1.RecordQuery{{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME, Value: name}},
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return drain(result)
}

func drain(result streaming.StreamResult[searchv1.SearchCIDsResponse]) []string {
	var cids []string

	for {
		select {
		case resp := <-result.ResCh():
			if resp != nil {
				cids = append(cids, resp.GetRecordCid())
			}
		case <-result.ErrCh():
			// Per-item errors leave the CID out, which the caller's assertion sees.
		case <-result.DoneCh():
			return cids
		}
	}
}

// expectWithheld asserts that a read was refused as a record the node does not
// serve under its policy, and that the refusal says nothing more than that.
func expectWithheld(err error) {
	ginkgo.GinkgoHelper()

	gomega.Expect(err).To(gomega.HaveOccurred())
	gomega.Expect(status.Code(err)).To(gomega.Equal(codes.PermissionDenied))
	gomega.Expect(err.Error()).To(gomega.ContainSubstring(withheldMessage))
	gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring(policyID), "the refusal must not name the policy")
}

var _ = ginkgo.Describe("Content policy enforcement", ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		compliant, rejected       *corev1.Record
		compliantRef, rejectedRef *corev1.RecordRef
	)

	ginkgo.BeforeAll(func(ctx context.Context) {
		// A policy is enforced once the reconciler has registered it and its
		// verdicts cover the records the node holds; until then the node serves
		// as it did. Push a record the policy rejects and wait for it to be
		// refused: from then on every read follows the policy.
		probeRef := push(ctx, newRecord("policy-probe", ""))

		gomega.Eventually(func() codes.Code { return pullCode(ctx, probeRef) }).
			WithTimeout(settleTimeout).
			WithPolling(pollInterval).
			Should(gomega.Equal(codes.PermissionDenied), "a record the policy rejects was never refused: the policy is not enforced, or it passed the record")
	})

	ginkgo.It("withholds a record until it has been evaluated", func(ctx context.Context) {
		compliant = newRecord("policy-compliant", described)
		rejected = newRecord("policy-rejected", "")

		compliantRef = push(ctx, compliant)
		rejectedRef = push(ctx, rejected)

		// Just pushed: neither has a verdict yet, so neither is served, though
		// the policy would pass the first. The reconciler waits a few seconds
		// after a push before it indexes, so this holds however fast the host.
		_, err := testEnv.Client.Pull(ctx, compliantRef)
		expectWithheld(err)

		_, err = testEnv.Client.Lookup(ctx, compliantRef)
		expectWithheld(err)
	})

	ginkgo.It("serves a record once the policy passes it", func(ctx context.Context) {
		gomega.Expect(compliantRef).NotTo(gomega.BeNil(), "the record must have been pushed")

		gomega.Eventually(func() codes.Code { return pullCode(ctx, compliantRef) }).
			WithTimeout(settleTimeout).
			WithPolling(pollInterval).
			Should(gomega.Equal(codes.OK), "a record the policy passes must be served once it is evaluated")

		pulled, err := testEnv.Client.Pull(ctx, compliantRef)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(pulled.GetData().AsMap()["name"]).To(gomega.Equal(compliant.GetData().AsMap()["name"]))

		meta, err := testEnv.Client.Lookup(ctx, compliantRef)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(meta.GetCid()).To(gomega.Equal(compliantRef.GetCid()))

		name, ok := compliant.GetData().AsMap()["name"].(string)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(searchByName(ctx, name)).To(gomega.ContainElement(compliantRef.GetCid()))
	})

	ginkgo.It("keeps a record the policy rejects out of reach", func(ctx context.Context) {
		gomega.Expect(rejectedRef).NotTo(gomega.BeNil(), "the record must have been pushed")

		// The rejected record was pushed with the compliant one and evaluated in
		// the same run, so by now it has had its verdict too. It stays withheld.
		gomega.Consistently(func() codes.Code { return pullCode(ctx, rejectedRef) }).
			WithTimeout(rejectedWindow).
			WithPolling(pollInterval).
			Should(gomega.Equal(codes.PermissionDenied))

		_, err := testEnv.Client.Pull(ctx, rejectedRef)
		expectWithheld(err)

		_, err = testEnv.Client.Lookup(ctx, rejectedRef)
		expectWithheld(err)

		name, ok := rejected.GetData().AsMap()["name"].(string)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(searchByName(ctx, name)).NotTo(gomega.ContainElement(rejectedRef.GetCid()))
	})

	ginkgo.It("does not say whether it holds a record it withholds", func(ctx context.Context) {
		// A record that was never pushed gets the answer a withheld one gets:
		// an enforcing node checks its policy before looking in the store, so
		// the refusal does not reveal which CIDs it holds.
		neverPushed := newRecord("policy-never-pushed", described)

		data, err := neverPushed.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		_, err = testEnv.Client.Pull(ctx, &corev1.RecordRef{Cid: utils.CalculateCIDFromData(data)})
		expectWithheld(err)
	})
})
