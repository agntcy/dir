// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e_production

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/tests/e2e-production/testdata"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	ghcrServerAddrEnv  = "DIRECTORY_E2E_GHCR_SERVER_ADDRESS"
	cosignTestPassword = "e2e-test-password"
)

func collectSearchCIDs(result <-chan *searchv1.SearchCIDsResponse, done <-chan struct{}, errCh <-chan error) ([]string, error) {
	var cids []string

	var errs error

	for {
		select {
		case resp, ok := <-result:
			if !ok {
				continue
			}

			cids = append(cids, resp.GetRecordCid())
		case err, ok := <-errCh:
			if !ok {
				continue
			}

			errs = errors.Join(errs, err)
		case <-done:
			return cids, errs
		}
	}
}

var _ = ginkgo.Describe("GHCR external OCI backend", ginkgo.Ordered, func() {
	var (
		ghcrClient *client.Client
		ctx        context.Context
		recordRef  *corev1.RecordRef
		record     *corev1.Record
	)

	ginkgo.BeforeAll(func() {
		serverAddr := os.Getenv(ghcrServerAddrEnv)
		if serverAddr == "" {
			ginkgo.Skip(fmt.Sprintf("Skipping GHCR tests: %s not set", ghcrServerAddrEnv))
		}

		ctx = context.Background()

		cfg, err := client.LoadConfig()
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to load client config from env")

		cfg.ServerAddress = serverAddr

		ghcrClient, err = client.New(ctx, client.WithConfig(cfg))
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create GHCR Directory client")

		record, err = corev1.UnmarshalRecord(testdata.RecordGHCRJSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if ghcrClient != nil {
			ghcrClient.Close()
		}
	})

	ginkgo.Context("Push and Pull", ginkgo.Ordered, func() {
		ginkgo.It("should push a record via the API", func() {
			var err error

			recordRef, err = ghcrClient.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(recordRef).NotTo(gomega.BeNil())
			gomega.Expect(recordRef.GetCid()).NotTo(gomega.BeEmpty())

			fmt.Fprintf(ginkgo.GinkgoWriter, "pushed record with CID: %s\n", recordRef.GetCid())
		})

		ginkgo.It("should pull the same record by CID and verify content matches", func() {
			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			pulledRecord, err := ghcrClient.Pull(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(pulledRecord).NotTo(gomega.BeNil())

			pushedData, err := record.Marshal()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			pulledData, err := pulledRecord.Marshal()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(pulledData).To(gomega.Equal(pushedData), "pulled record canonical data must match pushed record")
		})

		ginkgo.It("should return the same CID on duplicate push", func() {
			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			dupRef, err := ghcrClient.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(dupRef.GetCid()).To(gomega.Equal(recordRef.GetCid()))
		})

		ginkgo.It("should resolve metadata via Lookup", func() {
			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			meta, err := ghcrClient.Lookup(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(meta).NotTo(gomega.BeNil())
		})
	})

	ginkgo.Context("Sign and Verify", ginkgo.Ordered, func() {
		var (
			keyDir     string
			privateKey string
			publicKey  string
		)

		ginkgo.BeforeAll(func() {
			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			var err error

			keyDir, err = os.MkdirTemp("", "ghcr-e2e-sign-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			privateKey = filepath.Join(keyDir, "cosign.key")
			publicKey = filepath.Join(keyDir, "cosign.pub")

			cmd := exec.CommandContext(ctx, "cosign", "generate-key-pair")
			cmd.Dir = keyDir

			cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+cosignTestPassword)

			output, err := cmd.CombinedOutput()
			gomega.Expect(err).NotTo(gomega.HaveOccurred(),
				"cosign generate-key-pair failed: %s", string(output))

			gomega.Expect(privateKey).To(gomega.BeAnExistingFile())
			gomega.Expect(publicKey).To(gomega.BeAnExistingFile())
		})

		ginkgo.AfterAll(func() {
			if keyDir != "" {
				os.RemoveAll(keyDir)
			}
		})

		ginkgo.It("should sign the record with a cosign key pair", func() {
			resp, err := ghcrClient.Sign(ctx, &signv1.SignRequest{
				RecordRef: recordRef,
				Provider: &signv1.SignRequestProvider{
					Request: &signv1.SignRequestProvider_Key{
						Key: &signv1.SignWithKey{
							PrivateKey: privateKey,
							Password:   []byte(cosignTestPassword),
						},
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSignature()).NotTo(gomega.BeNil())
		})

		ginkgo.It("should verify the signature locally", func() {
			resp, err := ghcrClient.Verify(ctx, &signv1.VerifyRequest{
				RecordRef: recordRef,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSuccess()).To(gomega.BeTrue(),
				"signature verification should succeed; error: %s", resp.GetErrorMessage())
		})

		ginkgo.It("should pull the signature referrer", func() {
			sigs, err := ghcrClient.PullSignatures(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(sigs).NotTo(gomega.BeEmpty(), "should have at least one signature")
		})

		ginkgo.It("should pull the public key referrer", func() {
			keys, err := ghcrClient.PullPublicKeys(ctx, recordRef)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(keys).NotTo(gomega.BeEmpty(), "should have at least one public key")
		})
	})

	ginkgo.Context("Discovery via Search", func() {
		ginkgo.It("should find results by name", func() {
			sr, err := ghcrClient.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
				Queries: []*searchv1.RecordQuery{
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
						Value: "burger_seller_agent_ghcr",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cids, err := collectSearchCIDs(sr.ResCh(), sr.DoneCh(), sr.ErrCh())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(cids).NotTo(gomega.BeEmpty(), "should find at least one record by name")
		})

		ginkgo.It("should find results by skill name", func() {
			sr, err := ghcrClient.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
				Queries: []*searchv1.RecordQuery{
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME,
						Value: "natural_language_processing/natural_language_understanding/contextual_comprehension",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cids, err := collectSearchCIDs(sr.ResCh(), sr.DoneCh(), sr.ErrCh())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(cids).NotTo(gomega.BeEmpty(), "should find at least one record by skill name")
		})

		ginkgo.It("should find results by name and schema version", func() {
			sr, err := ghcrClient.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
				Queries: []*searchv1.RecordQuery{
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
						Value: "burger_seller_agent_ghcr",
					},
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION,
						Value: "1.0.0",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cids, err := collectSearchCIDs(sr.ResCh(), sr.DoneCh(), sr.ErrCh())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(cids).NotTo(gomega.BeEmpty(), "should find at least one record by name + schema version")
		})

		ginkgo.It("should not find a non-existent record", func() {
			sr, err := ghcrClient.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{
				Queries: []*searchv1.RecordQuery{
					{
						Type:  searchv1.RecordQueryType_RECORD_QUERY_TYPE_NAME,
						Value: "this_agent_does_not_exist_ghcr_e2e",
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			cids, err := collectSearchCIDs(sr.ResCh(), sr.DoneCh(), sr.ErrCh())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(cids).To(gomega.BeEmpty())
		})
	})

	ginkgo.Context("Delete", func() {
		ginkgo.It("should fail because GHCR does not support OCI delete API", func() {
			err := ghcrClient.Delete(ctx, recordRef)
			gomega.Expect(err).To(gomega.HaveOccurred())
			gomega.Expect(err.Error()).To(gomega.ContainSubstring("does not support OCI delete API"))
		})
	})
})
