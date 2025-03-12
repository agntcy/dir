// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package analyzer

import "errors"

var (
	// ErrNoSBOM is returned when no SBOM is found.
	ErrNoSBOM = errors.New("no SBOM found")

	// ErrNoRuntimeVersion is returned when no runtime version is found.
	ErrNoRuntimeVersion = errors.New("no runtime version found")

	// ErrUnsupportedRuntime is returned when the runtime is not supported.
	ErrUnsupportedRuntime = errors.New("unsupported runtime")
)

// Dependency represents a package dependency.
type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SBOM represents the Software Bill of Materials for a package.
type SBOM struct {
	Name     string    `json:"name"`
	Packages []Package `json:"packages,omitempty"`
}

// RuntimeInfo represents the minimal runtime version required.
type RuntimeInfo struct {
	Language LanguageType `json:"language"`
	Version  string       `json:"version,omitempty"`
}

type LanguageType string

const (
	Python LanguageType = "python"
)

// Analyzer is the interface that wraps the methods required to analyze packages.
type Analyzer interface {
	// AnalyzeSBOM analyzes the SBOM of a package and returns
	// its dependencies related to agentic requirements.
	SBOM(path string) (SBOM, error)

	// AnalyzeRuntimeVersion analyzes a package to determine the
	// minimal version of the runtime required.
	RuntimeVersion(path string) (RuntimeInfo, error)
}
