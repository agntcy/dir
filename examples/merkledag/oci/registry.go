package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

// registry is the global handler registry
var registry = newVersionRegistry()

// versionRegistry implements HandlerRegistry
type versionRegistry struct {
	versions     map[string][]EntityHandler
	mediaTypeMap map[string]map[string]EntityHandler
}

func newVersionRegistry() *versionRegistry {
	return &versionRegistry{
		versions:     make(map[string][]EntityHandler),
		mediaTypeMap: make(map[string]map[string]EntityHandler),
	}
}

func (r *versionRegistry) RegisterVersion(schemaVersion string, handlers []EntityHandler) {
	r.versions[schemaVersion] = handlers

	// Build media type lookup map for this version
	mediaTypeMap := make(map[string]EntityHandler)
	for _, handler := range handlers {
		mediaTypeMap[handler.MediaType()] = handler
	}
	r.mediaTypeMap[schemaVersion] = mediaTypeMap
}

func (r *versionRegistry) GetHandlers(schemaVersion string) ([]EntityHandler, error) {
	handlers, exists := r.versions[schemaVersion]
	if !exists {
		return nil, fmt.Errorf("no handlers registered for schema version %s", schemaVersion)
	}
	return handlers, nil
}

func (r *versionRegistry) getHandlerByMediaType(schemaVersion, mediaType string) (EntityHandler, bool) {
	versionMap, exists := r.mediaTypeMap[schemaVersion]
	if !exists {
		return nil, false
	}
	handler, exists := versionMap[mediaType]
	return handler, exists
}

// RegisterVersion registers handlers for a specific schema version
func RegisterVersion(schemaVersion string, handlers []EntityHandler) {
	registry.RegisterVersion(schemaVersion, handlers)
}

// Push converts an OASF record to an OCI manifest structure and pushes it
func Push(ctx context.Context, repo oras.Target, record *oasfv1.Record) (ocispec.Descriptor, error) {
	// Get handlers for this schema version
	handlers, err := registry.GetHandlers(record.GetSchemaVersion())
	if err != nil {
		return ocispec.Descriptor{}, err
	}

	var configDesc ocispec.Descriptor
	var layers []ocispec.Descriptor

	// Process all entity types using their handlers
	for i, handler := range handlers {
		entities := handler.GetEntities(record)
		for _, entity := range entities {
			// Convert entity to descriptor with embedded data
			desc, err := handler.ToLayer(entity)
			if err != nil {
				return ocispec.Descriptor{}, err
			}

			// Push the descriptor data to repository
			pushedDesc, err := oras.PushBytes(ctx, repo, desc.MediaType, desc.Data)
			if err != nil {
				return ocispec.Descriptor{}, fmt.Errorf("failed to push layer data: %w", err)
			}

			// Preserve annotations from original descriptor
			pushedDesc.Annotations = desc.Annotations

			// First handler (MetadataHandler) produces the config descriptor
			if i == 0 {
				configDesc = pushedDesc
			} else {
				layers = append(layers, pushedDesc)
			}
		}
	}

	// Pack manifest
	return packManifest(ctx, repo, record, configDesc, layers)
}

// Pull retrieves an OCI manifest and reconstructs the OASF record
func Pull(ctx context.Context, repo oras.Target, manifestDesc ocispec.Descriptor) (*oasfv1.Record, error) {
	// Fetch and unmarshal manifest
	manifest, err := fetchManifest(ctx, repo, manifestDesc)
	if err != nil {
		return nil, err
	}

	// Initialize record
	record := &oasfv1.Record{}

	// Process config layer first to get schema version
	if err := processLayer(ctx, repo, manifest.Config, record, ""); err != nil {
		return nil, err
	}

	// Now we have the schema version, validate handlers exist
	schemaVersion := record.GetSchemaVersion()
	if _, err := registry.GetHandlers(schemaVersion); err != nil {
		return nil, fmt.Errorf("unsupported schema version %s: %w", schemaVersion, err)
	}

	// Restore created_at from manifest annotations
	if manifest.Annotations != nil {
		if createdAt, ok := manifest.Annotations["org.opencontainers.image.created"]; ok {
			record.CreatedAt = createdAt
		}
	}

	// Process each layer
	for _, layer := range manifest.Layers {
		if err := processLayer(ctx, repo, layer, record, schemaVersion); err != nil {
			return nil, err
		}
	}

	return record, nil
}

// packManifest creates and pushes the manifest
func packManifest(ctx context.Context, repo oras.Target, record *oasfv1.Record, configDesc ocispec.Descriptor, layers []ocispec.Descriptor) (ocispec.Descriptor, error) {
	manifestAnnotations := map[string]string{
		"org.agntcy.record.schema_version": record.GetSchemaVersion(),
		"org.opencontainers.image.created": record.GetCreatedAt(),
	}

	manifestDesc, err := oras.PackManifest(ctx, repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ConfigDescriptor:    &configDesc,
			ManifestAnnotations: manifestAnnotations,
			Layers:              layers,
		},
	)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to pack manifest: %w", err)
	}

	return manifestDesc, nil
}

// fetchManifest fetches and unmarshals the manifest
func fetchManifest(ctx context.Context, repo oras.Target, manifestDesc ocispec.Descriptor) (*ocispec.Manifest, error) {
	manifestBytes, err := fetchContent(ctx, repo, manifestDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// processLayer processes a single layer based on its media type using handler registry
func processLayer(ctx context.Context, repo oras.Target, layer ocispec.Descriptor, record *oasfv1.Record, schemaVersion string) error {
	// If schema version is not provided (config layer), use empty string to process metadata first
	if schemaVersion == "" {
		schemaVersion = record.GetSchemaVersion()
		if schemaVersion == "" {
			// This is the config layer being processed, try to extract schema version from it
			schemaVersion = extractSchemaVersionFromConfig(ctx, repo, layer)
		}
	}

	// Look up handler by media type
	handler, exists := registry.getHandlerByMediaType(schemaVersion, layer.MediaType)
	if !exists {
		// Skip unknown media types
		return nil
	}

	// Fetch layer data from repository
	layerData, err := fetchContent(ctx, repo, layer)
	if err != nil {
		return fmt.Errorf("failed to fetch layer data: %w", err)
	}

	// Populate descriptor with fetched data
	layer.Data = layerData

	// Convert layer to entity using handler
	entity, err := handler.FromLayer(layer)
	if err != nil {
		return err
	}

	// Add to record
	handler.AppendToRecord(record, entity)

	return nil
}

// extractSchemaVersionFromConfig attempts to extract schema version from config layer
func extractSchemaVersionFromConfig(ctx context.Context, repo oras.Target, layer ocispec.Descriptor) string {
	// Fetch layer data
	layerData, err := fetchContent(ctx, repo, layer)
	if err != nil {
		return ""
	}

	// Try to unmarshal as metadata to get schema version
	var metadata struct {
		SchemaVersion string `json:"schema_version"`
	}
	if err := json.Unmarshal(layerData, &metadata); err != nil {
		return ""
	}

	return metadata.SchemaVersion
}

// fetchContent is a helper to fetch content from a descriptor
func fetchContent(ctx context.Context, repo oras.Target, desc ocispec.Descriptor) ([]byte, error) {
	if len(desc.Data) > 0 {
		return desc.Data, nil
	}

	reader, err := repo.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	return content, nil
}
