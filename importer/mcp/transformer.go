// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/config"
	"github.com/agntcy/dir/importer/enricher"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	// DefaultSchemaVersion is the default version of the OASF schema.
	DefaultOASFVersion = "0.8.0"
)

// Transformer implements the pipeline.Transformer interface for MCP records.
type Transformer struct {
	host *enricher.MCPHostClient
}

// NewTransformer creates a new MCP transformer.
// If cfg.Enrich is true, it initializes an enricher client using cfg.EnricherConfig.
func NewTransformer(ctx context.Context, cfg config.Config) (*Transformer, error) {
	var host *enricher.MCPHostClient

	if cfg.Enrich {
		// Create enricher configuration
		enricherCfg := enricher.Config{
			ConfigFile:            cfg.EnricherConfigFile,
			SkillsPromptTemplate:  cfg.EnricherSkillsPromptTemplate,
			DomainsPromptTemplate: cfg.EnricherDomainsPromptTemplate,
		}

		var err error

		host, err = enricher.NewMCPHost(ctx, enricherCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
		}
	}

	return &Transformer{
		host: host,
	}, nil
}

// Transform converts an MCP server response to OASF format.
func (t *Transformer) Transform(ctx context.Context, source interface{}) (*corev1.Record, error) {
	// Convert interface{} to ServerResponse
	response, ok := ServerResponseFromInterface(source)
	if !ok {
		return nil, fmt.Errorf("invalid source type: expected mcpapiv0.ServerResponse, got %T", source)
	}

	// Convert to OASF format
	record, err := t.convertToOASF(ctx, response)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server %s:%s to OASF: %w",
			response.Server.Name, response.Server.Version, err)
	}

	// Attach MCP source for debugging push failures
	// Store in a way that won't interfere with the record
	if record.GetData() != nil && record.Data.Fields != nil {
		if mcpBytes, err := json.Marshal(response.Server); err == nil {
			// Store as a JSON string for later retrieval
			record.Data.Fields["__mcp_debug_source"] = structpb.NewStringValue(string(mcpBytes))
		}
	}

	return record, nil
}

// convertToOASF converts an MCP server response to OASF format.
//
//nolint:unparam
func (t *Transformer) convertToOASF(ctx context.Context, response mcpapiv0.ServerResponse) (*corev1.Record, error) {
	server := response.Server

	// Convert the MCP ServerJSON to a structpb.Struct
	serverBytes, err := json.Marshal(server)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal server to JSON: %w", err)
	}

	var serverMap map[string]interface{}
	if err := json.Unmarshal(serverBytes, &serverMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal server JSON to map: %w", err)
	}

	serverStruct, err := structpb.NewStruct(serverMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert server map to structpb.Struct: %w", err)
	}

	mcpData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"server": structpb.NewStructValue(serverStruct),
		},
	}

	// Translate MCP struct to OASF record struct
	recordStruct, err := translator.MCPToRecord(mcpData)
	if err != nil {
		// Print MCP source on translation failure
		if mcpBytes, jsonErr := json.MarshalIndent(server, "", "  "); jsonErr == nil {
			fmt.Fprintf(os.Stderr, "\n========================================\n")
			fmt.Fprintf(os.Stderr, "TRANSLATION FAILED for: %s@%s\n", server.Name, server.Version)
			fmt.Fprintf(os.Stderr, "========================================\n")
			fmt.Fprintf(os.Stderr, "MCP Source:\n%s\n", string(mcpBytes))
			fmt.Fprintf(os.Stderr, "========================================\n\n")
			os.Stderr.Sync()
		}

		return nil, fmt.Errorf("failed to convert MCP data to OASF record: %w", err)
	}

	// Enrich the record with proper OASF skills and domains if enrichment is enabled
	if t.host != nil {
		if err := t.enrichRecord(ctx, recordStruct); err != nil {
			return nil, err
		}
	}

	return &corev1.Record{
		Data: recordStruct,
	}, nil
}

// enrichRecord handles the enrichment of a record with skills and domains.
func (t *Transformer) enrichRecord(ctx context.Context, recordStruct *structpb.Struct) error {
	// Convert structpb.Struct to typesv1alpha1.Record for enrichment
	oasfRecord, err := structToOASFRecord(recordStruct)
	if err != nil {
		return fmt.Errorf("failed to convert struct to OASF record for enrichment: %w", err)
	}

	// Clear default skills and domains before enrichment - let the LLM select appropriate ones
	oasfRecord.Skills = nil
	oasfRecord.Domains = nil

	// Context with timeout for enrichment operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute) //nolint:mnd
	defer cancel()

	// Enrich with skills
	enrichedRecord, err := t.host.EnrichWithSkills(ctxWithTimeout, oasfRecord)
	if err != nil {
		return fmt.Errorf("failed to enrich record with skills: %w", err)
	}

	// Enrich with domains (using the already skill-enriched record)
	enrichedRecord, err = t.host.EnrichWithDomains(ctxWithTimeout, enrichedRecord)
	if err != nil {
		return fmt.Errorf("failed to enrich record with domains: %w", err)
	}

	// Update both skills and domains fields, preserve everything else from the original record
	if err := updateSkillsInStruct(recordStruct, enrichedRecord.GetSkills()); err != nil {
		return fmt.Errorf("failed to update skills in record: %w", err)
	}

	if err := updateDomainsInStruct(recordStruct, enrichedRecord.GetDomains()); err != nil {
		return fmt.Errorf("failed to update domains in record: %w", err)
	}

	return nil
}

// structToOASFRecord converts a structpb.Struct to typesv1alpha1.Record for enrichment.
func structToOASFRecord(s *structpb.Struct) (*typesv1alpha1.Record, error) {
	// Marshal struct to JSON
	jsonBytes, err := protojson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	// Unmarshal JSON into typesv1alpha1.Record
	var record typesv1alpha1.Record
	if err := protojson.Unmarshal(jsonBytes, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to OASF record: %w", err)
	}

	return &record, nil
}

// enrichedItem represents any enriched field (skill or domain) with name and id.
type enrichedItem interface {
	GetName() string
	GetId() uint32
}

// updateFieldsInStruct is a generic helper that updates a field in a structpb.Struct with enriched items.
// This preserves all other fields including schema_version, name, version, etc.
func updateFieldsInStruct[T enrichedItem](recordStruct *structpb.Struct, fieldName string, enrichedItems []T) error {
	if recordStruct.Fields == nil {
		return errors.New("record struct has no fields")
	}

	// Convert enriched items to structpb.ListValue
	itemsList := &structpb.ListValue{
		Values: make([]*structpb.Value, 0, len(enrichedItems)),
	}

	for _, item := range enrichedItems {
		itemStruct := &structpb.Struct{
			Fields: make(map[string]*structpb.Value),
		}

		// Add name field (required)
		if item.GetName() != "" {
			itemStruct.Fields["name"] = structpb.NewStringValue(item.GetName())
		}

		// Add id field if present
		if item.GetId() != 0 {
			itemStruct.Fields["id"] = structpb.NewNumberValue(float64(item.GetId()))
		}

		itemsList.Values = append(itemsList.Values, structpb.NewStructValue(itemStruct))
	}

	// Update the field in the record
	recordStruct.Fields[fieldName] = structpb.NewListValue(itemsList)

	return nil
}

// updateSkillsInStruct updates the skills field in a structpb.Struct with enriched skills.
// This preserves all other fields including schema_version, name, version, etc.
func updateSkillsInStruct(recordStruct *structpb.Struct, enrichedSkills []*typesv1alpha1.Skill) error {
	return updateFieldsInStruct(recordStruct, "skills", enrichedSkills)
}

// updateDomainsInStruct updates the domains field in a structpb.Struct with enriched domains.
// This preserves all other fields including schema_version, name, version, etc.
func updateDomainsInStruct(recordStruct *structpb.Struct, enrichedDomains []*typesv1alpha1.Domain) error {
	return updateFieldsInStruct(recordStruct, "domains", enrichedDomains)
}
