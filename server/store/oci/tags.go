// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
)

var tagLogger = logging.Logger("store/oci/tags")

// TagStrategy defines different tagging strategies for enhanced discovery
type TagStrategy struct {
	// EnableNameTags controls whether to generate name-based tags
	EnableNameTags bool
	// EnableCapabilityTags controls whether to generate capability-based tags
	EnableCapabilityTags bool
	// EnableInfrastructureTags controls whether to generate infrastructure-based tags
	EnableInfrastructureTags bool
	// EnableTeamTags controls whether to generate team-based tags
	EnableTeamTags bool
	// EnableContentAddressable controls whether to include the CID tag (always recommended)
	EnableContentAddressable bool
	// MaxTagsPerRecord limits the total number of tags to prevent tag explosion
	MaxTagsPerRecord int
}

// DefaultTagStrategy provides a balanced tagging approach
var DefaultTagStrategy = TagStrategy{
	EnableNameTags:           true,
	EnableCapabilityTags:     true,
	EnableInfrastructureTags: true,
	EnableTeamTags:           true,
	EnableContentAddressable: true,
	MaxTagsPerRecord:         20,
}

// normalizeTagForOCI ensures tag compliance with OCI registry requirements
// Tags must be valid ASCII and follow specific patterns
func normalizeTagForOCI(tag string) string {
	if tag == "" {
		return ""
	}

	// Convert to lowercase (OCI registries typically prefer lowercase)
	normalized := strings.ToLower(tag)

	// Replace invalid characters with safe alternatives
	// OCI tag regex: [a-zA-Z0-9_][a-zA-Z0-9._-]*
	var result strings.Builder
	for i, char := range normalized {
		switch {
		case (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'):
			result.WriteRune(char)
		case char == '_' || char == '.' || char == '-':
			// Valid characters for non-first position
			if i == 0 {
				result.WriteRune('_') // Replace invalid first char
			} else {
				result.WriteRune(char)
			}
		case char == ' ':
			result.WriteRune('-') // Replace spaces with hyphens
		case char == '/' || char == '\\':
			result.WriteRune('.') // Replace path separators with dots
		default:
			// Replace other invalid characters with underscore
			result.WriteRune('_')
		}
	}

	tag = result.String()

	// Ensure first character is valid ([a-zA-Z0-9_])
	if len(tag) > 0 && !(tag[0] >= 'a' && tag[0] <= 'z') && !(tag[0] >= '0' && tag[0] <= '9') && tag[0] != '_' {
		tag = "_" + tag
	}

	// Limit length (OCI doesn't specify but reasonable limit)
	if len(tag) > 128 {
		tag = tag[:128]
	}

	// Remove trailing separators
	tag = strings.TrimRight(tag, ".-_")

	return tag
}

// generateTagsFromMetadata is the core tag generation logic that works with metadata
// This eliminates duplication between generateDiscoveryTags and reconstructTagsFromRecord
func generateTagsFromMetadata(metadata map[string]string, cid string, strategy TagStrategy) []string {
	var tags []string

	// 1. Content-addressable tag (always first for backward compatibility)
	if strategy.EnableContentAddressable && cid != "" {
		tags = append(tags, normalizeTagForOCI(cid))
	}

	// 2. Name-based tags for browsability
	if strategy.EnableNameTags {
		if name := metadata[MetadataKeyName]; name != "" {
			// Basic name tag
			if tag := normalizeTagForOCI(name); tag != "" {
				tags = append(tags, tag)
			}

			// Name with version for specific releases
			if version := metadata[MetadataKeyVersion]; version != "" {
				nameVersionTag := normalizeTagForOCI(name + ":" + version)
				if nameVersionTag != "" {
					tags = append(tags, nameVersionTag)
				}
			}

			// "Latest" convention for latest version
			latestTag := normalizeTagForOCI(name + ":latest")
			if latestTag != "" {
				tags = append(tags, latestTag)
			}
		}
	}

	// 3. Capability-based tags for functional discovery
	if strategy.EnableCapabilityTags {
		if skills := metadata[MetadataKeySkills]; skills != "" {
			skillList := parseCommaSeparated(skills)
			for _, skill := range skillList {
				if skill != "" {
					skillTag := normalizeTagForOCI("skill." + skill)
					if skillTag != "" {
						tags = append(tags, skillTag)
					}
				}
			}
		}

		// Extension-based capability tags
		if extensions := metadata[MetadataKeyExtensionNames]; extensions != "" {
			extList := parseCommaSeparated(extensions)
			for _, ext := range extList {
				if ext != "" {
					extTag := normalizeTagForOCI("ext." + ext)
					if extTag != "" {
						tags = append(tags, extTag)
					}
				}
			}
		}
	}

	// 4. Infrastructure-based tags for deployment discovery
	if strategy.EnableInfrastructureTags {
		if locatorTypes := metadata[MetadataKeyLocatorTypes]; locatorTypes != "" {
			locatorList := parseCommaSeparated(locatorTypes)
			for _, locator := range locatorList {
				if locator != "" {
					deployTag := normalizeTagForOCI("deploy." + locator)
					if deployTag != "" {
						tags = append(tags, deployTag)
					}
				}
			}
		}
	}

	// 5. Team-based tags from custom annotations
	if strategy.EnableTeamTags {
		// Look for team-related annotations
		if team := metadata[MetadataKeyTeam]; team != "" {
			teamTag := normalizeTagForOCI("team." + team)
			if teamTag != "" {
				tags = append(tags, teamTag)
			}
		}

		// Look for organization annotations
		if org := metadata[MetadataKeyOrganization]; org != "" {
			orgTag := normalizeTagForOCI("org." + org)
			if orgTag != "" {
				tags = append(tags, orgTag)
			}
		}

		// Look for project annotations
		if project := metadata[MetadataKeyProject]; project != "" {
			projectTag := normalizeTagForOCI("project." + project)
			if projectTag != "" {
				tags = append(tags, projectTag)
			}
		}
	}

	// Apply deduplication and limits
	tags = removeDuplicateTags(tags)
	if strategy.MaxTagsPerRecord > 0 && len(tags) > strategy.MaxTagsPerRecord {
		tags = tags[:strategy.MaxTagsPerRecord]
	}

	return tags
}

// extractMetadataFromRecord converts record data to the metadata format used by generateTagsFromMetadata
func extractMetadataFromRecord(record *corev1.Record) map[string]string {
	metadata := make(map[string]string)

	// Use adapter pattern for version-agnostic access
	adapter := adapters.NewRecordAdapter(record)
	recordData := adapter.GetRecordData()

	if recordData == nil {
		return metadata
	}

	// Extract core metadata
	if name := recordData.GetName(); name != "" {
		metadata[MetadataKeyName] = name
	}
	if version := recordData.GetVersion(); version != "" {
		metadata[MetadataKeyVersion] = version
	}

	// Extract capability metadata
	if skills := recordData.GetSkills(); len(skills) > 0 {
		skillNames := make([]string, len(skills))
		for i, skill := range skills {
			skillNames[i] = skill.GetName()
		}
		metadata[MetadataKeySkills] = strings.Join(skillNames, ",")
	}

	if extensions := recordData.GetExtensions(); len(extensions) > 0 {
		extensionNames := make([]string, len(extensions))
		for i, extension := range extensions {
			extensionNames[i] = extension.GetName()
		}
		metadata[MetadataKeyExtensionNames] = strings.Join(extensionNames, ",")
	}

	// Extract infrastructure metadata
	if locators := recordData.GetLocators(); len(locators) > 0 {
		locatorTypes := make([]string, len(locators))
		for i, locator := range locators {
			locatorTypes[i] = locator.GetType()
		}
		metadata[MetadataKeyLocatorTypes] = strings.Join(locatorTypes, ",")
	}

	// Extract team metadata from custom annotations
	if annotations := recordData.GetAnnotations(); len(annotations) > 0 {
		for key, value := range annotations {
			switch key {
			case "team":
				metadata[MetadataKeyTeam] = value
			case "organization":
				metadata[MetadataKeyOrganization] = value
			case "project":
				metadata[MetadataKeyProject] = value
			}
		}
	}

	return metadata
}

// generateDiscoveryTags creates multiple tags for enhanced agent discovery
// Now uses the shared helper to eliminate duplication
func generateDiscoveryTags(record *corev1.Record, strategy TagStrategy) []string {
	if record == nil {
		return []string{}
	}

	// Extract metadata from record
	metadata := extractMetadataFromRecord(record)

	// Use shared helper
	return generateTagsFromMetadata(metadata, record.GetCid(), strategy)
}

// reconstructTagsFromRecord rebuilds discovery tags from stored metadata
// Moved to tags.go for better organization and uses shared helper
func reconstructTagsFromRecord(metadata map[string]string, cid string) []string {
	// Use the same DefaultTagStrategy that was used during Push
	// and the same shared helper to ensure perfect synchronization
	return generateTagsFromMetadata(metadata, cid, DefaultTagStrategy)
}

// removeDuplicateTags removes duplicate tags while preserving order
func removeDuplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, tag := range tags {
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	return result
}

// pushManifestWithTags pushes a manifest with multiple discovery tags
func (s *store) pushManifestWithTags(ctx context.Context, manifestDesc ocispec.Descriptor, tags []string) error {
	var tagErrors []string

	for _, tag := range tags {
		if tag == "" {
			continue
		}

		if _, err := oras.Tag(ctx, s.repo, manifestDesc.Digest.String(), tag); err != nil {
			// Log error but continue with other tags - don't fail entire push for tag issues
			tagLogger.Warn("Failed to create discovery tag", "tag", tag, "error", err)
			tagErrors = append(tagErrors, fmt.Sprintf("%s: %v", tag, err))
		} else {
			tagLogger.Debug("Successfully created discovery tag", "tag", tag)
		}
	}

	// If we have tag errors but at least one tag succeeded, log warnings but don't fail
	if len(tagErrors) > 0 {
		if len(tagErrors) == len(tags) {
			// All tags failed - this is a serious issue
			return status.Errorf(codes.Internal, "failed to create any discovery tags: %v", strings.Join(tagErrors, "; "))
		} else {
			// Some tags failed - log but continue
			tagLogger.Warn("Some discovery tags failed", "errors", strings.Join(tagErrors, "; "))
		}
	}

	return nil
}
