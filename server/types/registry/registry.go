// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package registry provides shared types for OCI registry backends.
package registry

import (
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

	// RegistryTypeOCI represents a generic OCI registry type.
	// Must support OCI 1.1 Distribution/Image Spec.
	RegistryTypeOCI RegistryType = "oci"

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
	case RegistryTypeOCI:
		return true
	default:
		return false
	}
}
