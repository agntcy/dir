// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
)

// Package-level variables for tracking CIDs across all network tests.
var (
	deployTestCIDs         []string
	syncTestCIDs           []string
	remoteSearchTestCIDs   []string
	gossipsubTestCIDs      []string
	nameResolutionTestCIDs []string
)

// This ensures clean state between different test files (Describe blocks).
func CleanupNetworkRecords(cids []string, testName string, peers []*utils.CLI) {
	if len(cids) == 0 {
		ginkgo.GinkgoWriter.Printf("No CIDs to clean up for %s", testName)

		return
	}

	ginkgo.GinkgoWriter.Printf("Cleaning up %d test records from %s", len(cids), testName)

	for _, cid := range cids {
		if cid == "" {
			continue // Skip empty CIDs
		}

		// Clean up from each peer to ensure complete isolation
		for peerID, peer := range peers {
			ginkgo.GinkgoWriter.Printf("  Cleaning CID %s from peer %d", cid, peerID+1)

			// Try to unpublish from routing (may fail if not published, which is okay)
			_, _ = peer.Routing().Unpublish(cid).SuppressStderr().Execute()

			// Try to delete from storage (may fail if not stored, which is okay)
			_, _ = peer.Delete(cid).SuppressStderr().Execute()
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
	case "name_resolution":
		nameResolutionTestCIDs = append(nameResolutionTestCIDs, cid)
	default:
		ginkgo.GinkgoWriter.Printf("Warning: Unknown test file %s for CID %s", testFile, cid)
	}
}

// CleanupAllNetworkTests removes all CIDs from all test files (used by AfterSuite).
func CleanupAllNetworkTests(peers []*utils.CLI) {
	allCIDs := []string{}
	allCIDs = append(allCIDs, deployTestCIDs...)
	allCIDs = append(allCIDs, syncTestCIDs...)
	allCIDs = append(allCIDs, remoteSearchTestCIDs...)
	allCIDs = append(allCIDs, gossipsubTestCIDs...)
	allCIDs = append(allCIDs, nameResolutionTestCIDs...)

	CleanupNetworkRecords(allCIDs, "all network tests", peers)
}
