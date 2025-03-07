// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package syft

import (
	"context"
	"fmt"
	"path"
	"slices"

	"github.com/anchore/syft/syft"
	"github.com/agntcy/dir/cli/builder/plugins/runtime/analyzer"
)

type Syft struct{}

func (s *Syft) SBOM(filePath string, supportedPackages []string) (analyzer.SBOM, error) {
	ctx := context.Background()

	cfg := syft.DefaultGetSourceConfig()
	source, err := syft.GetSource(ctx, filePath, cfg)
	if err != nil {
		return analyzer.SBOM{}, fmt.Errorf("failed to get source: %w", err)
	}

	sbomConfig := syft.DefaultCreateSBOMConfig()
	sbom, err := syft.CreateSBOM(ctx, source, sbomConfig)
	if err != nil {
		return analyzer.SBOM{}, fmt.Errorf("failed to create SBOM: %w", err)
	}

	var packages []analyzer.Package
	p := sbom.Artifacts.Packages.Sorted()
	for _, pkg := range p {
		// Skip packages not in agent framework packages list
		if !slices.Contains[[]string](supportedPackages, pkg.Name) {
			continue
		}

		packages = append(packages, analyzer.Package{
			Name:    pkg.Name,
			Version: pkg.Version,
		})
	}

	return analyzer.SBOM{
		// normalize source name
		Name:     path.Base(sbom.Source.Name),
		Packages: packages,
	}, nil
}
