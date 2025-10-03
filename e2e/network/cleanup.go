// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
)

// Package-level variables for tracking CIDs across all network tests.
var (
	deployTestCIDs       []string
	syncTestCIDs         []string
	remoteSearchTestCIDs []string
	gossipsubTestCIDs    []string
)

// This ensures clean state between different test files (Describe blocks).
func CleanupNetworkRecords(cids []string, testName string) {
	if len(cids) == 0 {
		ginkgo.GinkgoWriter.Printf("No CIDs to clean up for %s", testName)

		return
	}

	cleanupCLI := utils.NewCLI()

	ginkgo.GinkgoWriter.Printf("Cleaning up %d test records from %s", len(cids), testName)

	for _, cid := range cids {
		if cid == "" {
			continue // Skip empty CIDs
		}

		// Clean up from each peer to ensure complete isolation
		for _, peerAddr := range utils.PeerAddrs {
			ginkgo.GinkgoWriter.Printf("  Cleaning CID %s from peer %s", cid, peerAddr)

			// Try to unpublish from routing (may fail if not published, which is okay)
			_, err := cleanupCLI.Routing().Unpublish(cid).OnServer(peerAddr).Execute()
			if err != nil {
				ginkgo.GinkgoWriter.Printf("    Unpublish warning: %v (may not have been published)", err)
			}

			// Try to delete from storage (may fail if not stored, which is okay)
			_, err = cleanupCLI.Delete(cid).OnServer(peerAddr).Execute()
			if err != nil {
				ginkgo.GinkgoWriter.Printf("    Delete warning: %v (may not have been stored)", err)
			}
		}
	}

	ginkgo.GinkgoWriter.Printf("Cleanup completed for %s - all peers should be clean", testName)
}

// RegisterCIDForCleanup adds a CID to the appropriate test file's tracking array.
func RegisterCIDForCleanup(cid, testFile string) {
	switch testFile {
	case "deploy":
		deployTestCIDs = append(deployTestCIDs, cid)
	case "sync":
		syncTestCIDs = append(syncTestCIDs, cid)
	case "search":
		remoteSearchTestCIDs = append(remoteSearchTestCIDs, cid)
	case "gossipsub":
		gossipsubTestCIDs = append(gossipsubTestCIDs, cid)
	default:
		ginkgo.GinkgoWriter.Printf("Warning: Unknown test file %s for CID %s", testFile, cid)
	}
}

// CleanupAllNetworkTests removes all CIDs from all test files (used by AfterSuite).
func CleanupAllNetworkTests() {
	allCIDs := []string{}
	allCIDs = append(allCIDs, deployTestCIDs...)
	allCIDs = append(allCIDs, syncTestCIDs...)
	allCIDs = append(allCIDs, remoteSearchTestCIDs...)
	allCIDs = append(allCIDs, gossipsubTestCIDs...)

	CleanupNetworkRecords(allCIDs, "all network tests")
}
