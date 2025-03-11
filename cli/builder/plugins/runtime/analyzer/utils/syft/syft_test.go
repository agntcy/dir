// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package syft

import (
	"testing"

	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
)

func TestSyft(t *testing.T) {
	t.Run("SBOM", func(t *testing.T) {
		// Test the SBOM method
		s := Syft{}

		sbom, err := s.SBOM("testdata", []string{"crewai"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedSBOM := analyzer.SBOM{
			Name: "testdata",
			Packages: []analyzer.Package{
				{Name: "crewai", Version: "0.83.0"},
			},
		}

		if sbom.Name != expectedSBOM.Name || len(sbom.Packages) != len(expectedSBOM.Packages) {
			t.Errorf("unexpected SBOM: got %+v, want %+v", sbom, expectedSBOM)
		}

		if sbom.Packages[0].Name != expectedSBOM.Packages[0].Name || sbom.Packages[0].Version != expectedSBOM.Packages[0].Version {
			t.Errorf("unexpected package: got %+v, want %+v", sbom.Packages[0], expectedSBOM.Packages[0])
		}
	})
}
