// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package registry provides shared types for OCI registry backends.
package registry

import (
	"strings"

	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("server/types/registry")

// RegistryType defines the type of OCI registry backend.
// Only explicitly tested registries are fully supported.
type RegistryType string

const (
	// RegistryTypeZot represents a Zot registry.
	RegistryTypeZot RegistryType = "zot"

	// RegistryTypeGHCR represents GitHub Container Registry.
	RegistryTypeGHCR RegistryType = "ghcr"

	// RegistryTypeDockerHub represents Docker Hub.
	RegistryTypeDockerHub RegistryType = "dockerhub"

	// RegistryTypeUnknown represents an unknown registry type.
	RegistryTypeUnknown RegistryType = "unknown"

	// DefaultRegistryType is the default registry type for backward compatibility.
	DefaultRegistryType = RegistryTypeZot
)

// IsSupported returns true if the registry type is explicitly supported and tested.
// Logs a warning if an experimental registry type (ghcr, dockerhub) is used.
func IsSupported(r RegistryType) bool {
	switch r {
	case RegistryTypeZot:
		return true
	case RegistryTypeGHCR, RegistryTypeDockerHub:
		logger.Warn("Registry type support is experimental and not fully tested. "+
			"The default deployment configuration (Zot registry) is not appropriate for this registry type. "+
			"Do not use in production.",
			"registry_type", string(r))

		return true
	case RegistryTypeUnknown:
		return false
	default:
		return false
	}
}

// DetectRegistryType attempts to detect the registry type from a URL.
func DetectRegistryType(url string) RegistryType {
	url = strings.ToLower(url)

	switch {
	case strings.Contains(url, "ghcr.io"):
		return RegistryTypeGHCR
	case strings.Contains(url, "docker.io") || strings.Contains(url, "registry.hub.docker.com"):
		return RegistryTypeDockerHub
	case strings.Contains(url, "zot") || strings.Contains(url, "localhost"):
		return RegistryTypeZot
	default:
		return RegistryTypeUnknown
	}
}
