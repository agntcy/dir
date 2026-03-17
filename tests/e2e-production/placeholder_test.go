// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e_production

import (
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("E2E production suite", func() {
	ginkgo.It("is configured and runnable", func() {
		gomega.Expect(true).To(gomega.BeTrue())
	})
})
