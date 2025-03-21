// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/client"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
)

var _ = ginkgo.Describe("client end-to-end tests", func() {
	var err error
	ctx := context.Background()

	// Create a new client
	c, err := client.New(client.WithEnvConfig())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create agent object
	agent := &coretypes.Agent{
		Name:    "test-agent",
		Version: "v1",
		Skills: []*coretypes.Skill{
			{
				CategoryName: Ptr("test-category"),
				ClassName:    Ptr("test-class"),
			},
		},
		Locators: []*coretypes.Locator{
			{
				Type: "source-code",
				Url:  "url1",
			},
		},
	}

	// Marshal the Agent struct back to bytes.
	agentData, err := json.Marshal(agent)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create ref
	ref := &coretypes.ObjectRef{
		Digest:      digest.FromBytes(agentData).String(),
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Size:        uint64(len(agentData)),
		Annotations: agent.GetAnnotations(),
	}

	ginkgo.Context("agent push and pull", func() {
		ginkgo.It("should push an agent to store", func() {
			ref, err = c.Push(ctx, ref, bytes.NewReader(agentData))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate valid digest
			_, err = digest.Parse(ref.GetDigest())
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should pull an agent from store", func() {
			// Pull the agent object
			reader, err := c.Pull(ctx, ref)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Reader to bytes
			pulledAgentData, err := io.ReadAll(reader)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare pushed and pulled agent
			equal, err := compareJSONAgents(agentData, pulledAgentData)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})
	})

	ginkgo.Context("routing publish and list", func() {
		ginkgo.It("should publish an agent", func() {
			err = c.Publish(ctx, &coretypes.ObjectRef{
				Digest: ref.GetDigest(),
				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.It("should list published agent by skill", func() {
			refs, err := c.List(ctx, "/skills/test-category/test-class")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(refs).To(gomega.HaveLen(1))
		})
	})
})
