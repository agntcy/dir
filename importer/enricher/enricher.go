// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	enricherconfig "github.com/agntcy/dir/importer/enricher/config"
	"github.com/agntcy/dir/importer/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mark3labs/mcphost/sdk"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

var logger = logging.Logger("importer/enricher")

const (
	DebugMode                  = false
	DefaultConfidenceThreshold = 0.5
)

// enricherClient is the subset of enricher used by the transformer.
type enricherClient interface {
	EnrichWithSkills(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error)
	EnrichWithDomains(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error)
	EnrichWithSkillsV1(ctx context.Context, record *typesv1.Record) (*typesv1.Record, error)
	EnrichWithDomainsV1(ctx context.Context, record *typesv1.Record) (*typesv1.Record, error)
}

// hostRunner is the minimal interface for running prompts.
type hostRunner interface {
	Prompt(ctx context.Context, prompt string) (string, error)
	PromptWithCallbacks(ctx context.Context, prompt string, onToolCall func(name, args string), onToolResult func(name, args, result string, isError bool), onChunk func(chunk string)) (string, error)
	ClearSession()
}

type MCPHostClient struct {
	host                  hostRunner
	skillsPromptTemplate  string
	domainsPromptTemplate string
	rateLimiter           *rate.Limiter
}

// EnrichedField represents a single enriched field (skill or domain) with metadata.
type EnrichedField struct {
	Name       string  `json:"name"`
	ID         uint32  `json:"id"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// EnrichmentResponse represents the structured JSON response from the LLM.
// It can contain either skills or domains depending on the enrichment type.
type EnrichmentResponse struct {
	Skills  []EnrichedField `json:"skills,omitempty"`
	Domains []EnrichedField `json:"domains,omitempty"`
}

type Enricher struct {
	host enricherClient
}

func New(ctx context.Context, config enricherconfig.Config) (*Enricher, error) {
	host, err := NewMCPHost(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
	}

	return &Enricher{
		host: host,
	}, nil
}

func NewMCPHost(ctx context.Context, config enricherconfig.Config) (*MCPHostClient, error) {
	// Apply environment variables from config file to current process
	// Note: mcphost doesn't pass env vars from mcphost.json to spawned processes,
	// so we set them in the current process environment where they'll be inherited by child processes.
	if err := applyEnvVarsFromConfig(config.ConfigFile); err != nil {
		logger.Debug("Failed to apply env vars from config", "error", err)
		// Don't fail if env vars can't be applied - they might not be needed
	}

	// Initialize MCP Host
	host, err := sdk.New(ctx, &sdk.Options{
		ConfigFile: config.ConfigFile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
	}

	return newMCPHostFromRunner(ctx, host, config)
}

// NewMCPHostWithRunner creates an MCPHostClient with a custom host runner (e.g. for tests).
// It skips config file and SDK initialization; use for injecting a mock runner.
func NewMCPHostWithRunner(runner hostRunner, config enricherconfig.Config) (*MCPHostClient, error) {
	return newMCPHostFromRunner(context.Background(), runner, config)
}

func newMCPHostFromRunner(ctx context.Context, runner hostRunner, config enricherconfig.Config) (*MCPHostClient, error) {
	if DebugMode {
		if host, ok := runner.(*sdk.MCPHost); ok {
			runGetSchemaToolsPrompt(ctx, host)
		}
	}

	// Convert requests per minute to rate.Limit (requests per second)
	rateLimit := rate.Limit(float64(config.RequestsPerMinute) / 60.0) //nolint:mnd
	rateLimiter := rate.NewLimiter(rateLimit, 1)                      // burst of 1 request

	return &MCPHostClient{
		host:                  runner,
		skillsPromptTemplate:  config.SkillsPromptTemplate,
		domainsPromptTemplate: config.DomainsPromptTemplate,
		rateLimiter:           rateLimiter,
	}, nil
}

func (e *Enricher) Enrich(ctx context.Context, inputCh <-chan *corev1.Record, result *types.Result) (<-chan *corev1.Record, <-chan error) {
	outputCh := make(chan *corev1.Record)
	errCh := make(chan error)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			case record, ok := <-inputCh:
				if !ok {
					return
				}

				err := e.enrichRecord(ctx, record.GetData())
				if err != nil {
					result.Mu.Lock()
					result.FailedCount++
					result.Mu.Unlock()

					errCh <- fmt.Errorf("failed to enrich record: %w", err)

					return
				}

				select {
				case outputCh <- record:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return outputCh, errCh
}

// applyEnvVarsFromConfig reads environment variables from mcphost.json and sets them
// in the current process environment. This ensures they're inherited by spawned processes.
// Environment variables already set in the current process are not overridden.
func applyEnvVarsFromConfig(configFile string) error {
	configBytes, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg map[string]any
	if err := json.Unmarshal(configBytes, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	servers, ok := cfg["mcpServers"].(map[string]any)
	if !ok {
		return nil // No mcpServers section
	}

	dirServer, ok := servers["dir-mcp-server"].(map[string]any)
	if !ok {
		return nil // No dir-mcp-server configuration
	}

	env, ok := dirServer["env"].(map[string]any)
	if !ok {
		return nil // No env section
	}

	// Set environment variables from config (only if not already set)
	for key, value := range env {
		strValue, ok := value.(string)
		if !ok {
			continue // Skip non-string values
		}

		// Only set if not already set (allow override from shell)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, strValue); err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", key, err)
			}

			logger.Debug("Set environment variable from config", "key", key)
		}
	}

	return nil
}

// fieldType represents the type of field being enriched (skills or domains).
type fieldType string

const (
	fieldTypeSkills  fieldType = "skills"
	fieldTypeDomains fieldType = "domains"
)

// EnrichWithSkills enriches the record with OASF skills using the LLM and MCP tools.
func (c *MCPHostClient) EnrichWithSkills(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	return c.enrichField(ctx, record, fieldTypeSkills, c.skillsPromptTemplate)
}

// EnrichWithDomains enriches the record with OASF domains using the LLM and MCP tools.
func (c *MCPHostClient) EnrichWithDomains(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	return c.enrichField(ctx, record, fieldTypeDomains, c.domainsPromptTemplate)
}

// EnrichWithSkillsV1 enriches the record with OASF skills using the LLM and MCP tools (for v1 records).
func (c *MCPHostClient) EnrichWithSkillsV1(ctx context.Context, record *typesv1.Record) (*typesv1.Record, error) {
	return c.enrichFieldV1(ctx, record, fieldTypeSkills, c.skillsPromptTemplate)
}

// EnrichWithDomainsV1 enriches the record with OASF domains using the LLM and MCP tools (for v1 records).
func (c *MCPHostClient) EnrichWithDomainsV1(ctx context.Context, record *typesv1.Record) (*typesv1.Record, error) {
	return c.enrichFieldV1(ctx, record, fieldTypeDomains, c.domainsPromptTemplate)
}

// enrichField is the generic enrichment method that handles both skills and domains.
//
//nolint:dupl // Similar structure to enrichFieldV1 but uses different types
func (c *MCPHostClient) enrichField(
	ctx context.Context,
	record *typesv1alpha1.Record,
	fType fieldType,
	promptTemplate string,
) (*typesv1alpha1.Record, error) {
	enrichedFields, err := c.runEnrichmentPrompt(ctx, record, fType, promptTemplate)
	if err != nil {
		return nil, err
	}

	// Add enriched fields to record
	for _, field := range enrichedFields {
		if field.Confidence >= DefaultConfidenceThreshold {
			switch fType {
			case fieldTypeSkills:
				record.Skills = append(record.Skills, &typesv1alpha1.Skill{
					Name: field.Name,
					Id:   field.ID,
				})
			case fieldTypeDomains:
				record.Domains = append(record.Domains, &typesv1alpha1.Domain{
					Name: field.Name,
					Id:   field.ID,
				})
			}

			logger.Debug(fmt.Sprintf("Added %s", fType), "name", field.Name, "id", field.ID, "confidence", field.Confidence, "reasoning", field.Reasoning)
		} else {
			logger.Debug(fmt.Sprintf("Skipped low-confidence %s", fType), "name", field.Name, "confidence", field.Confidence, "threshold", DefaultConfidenceThreshold)
		}
	}

	enrichedRecordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
	}

	logger.Debug(fmt.Sprintf("Enriched record with %s", fType), "record", string(enrichedRecordJSON))

	return record, nil
}

// enrichFieldV1 is the generic enrichment method that handles both skills and domains for v1 records.
//
//nolint:dupl // Similar structure to enrichField but uses different types
func (c *MCPHostClient) enrichFieldV1(
	ctx context.Context,
	record *typesv1.Record,
	fType fieldType,
	promptTemplate string,
) (*typesv1.Record, error) {
	enrichedFields, err := c.runEnrichmentPrompt(ctx, record, fType, promptTemplate)
	if err != nil {
		return nil, err
	}

	// Add enriched fields to record
	for _, field := range enrichedFields {
		if field.Confidence >= DefaultConfidenceThreshold {
			switch fType {
			case fieldTypeSkills:
				record.Skills = append(record.Skills, &typesv1.Skill{
					Name: field.Name,
					Id:   field.ID,
				})
			case fieldTypeDomains:
				record.Domains = append(record.Domains, &typesv1.Domain{
					Name: field.Name,
					Id:   field.ID,
				})
			}

			logger.Debug(fmt.Sprintf("Added %s", fType), "name", field.Name, "id", field.ID, "confidence", field.Confidence, "reasoning", field.Reasoning)
		} else {
			logger.Debug(fmt.Sprintf("Skipped low-confidence %s", fType), "name", field.Name, "confidence", field.Confidence, "threshold", DefaultConfidenceThreshold)
		}
	}

	enrichedRecordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
	}

	logger.Debug(fmt.Sprintf("Enriched record with %s", fType), "record", string(enrichedRecordJSON))

	return record, nil
}

// runEnrichmentPrompt runs the enrichment prompt and parses the response.
// This is shared between enrichField and enrichFieldV1 to avoid code duplication.
func (c *MCPHostClient) runEnrichmentPrompt(ctx context.Context, record any, fType fieldType, promptTemplate string) ([]EnrichedField, error) {
	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Run prompt with the specified template
	response, err := c.runPrompt(ctx, promptTemplate, recordJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to run prompt for %s: %w", fType, err)
	}

	// Parse response to get enriched fields
	enrichedFields, err := c.parseResponse(response, fType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", fType, err)
	}

	return enrichedFields, nil
}

func runGetSchemaToolsPrompt(ctx context.Context, host *sdk.MCPHost) {
	// Get 3 OASF skills
	resp, err := host.Prompt(ctx, "Call the tool 'dir-mcp-server__agntcy_oasf_get_schema_skills' and return 3 skill names)")
	if err != nil {
		logger.Error("failed to get 3 OASF skills", "error", err)
	}

	logger.Info("3 OASF skills", "skills", resp)

	// Get 3 sub-skills for the skill natural_language_processing
	resp, err = host.Prompt(ctx, "Call the tool 'dir-mcp-server__agntcy_oasf_get_schema_skills' and return 3 sub-skills for the skill natural_language_processing")
	if err != nil {
		logger.Error("failed to get 3 sub-skills for natural_language_processing", "error", err)
	}

	logger.Info("3 sub-skills for natural_language_processing", "sub-skills", resp)
}

func (c *MCPHostClient) runPrompt(ctx context.Context, promptTemplate string, recordJSON []byte) (string, error) {
	prompt := promptTemplate + string(recordJSON)

	// Apply rate limiting before making the LLM API call
	// This blocks until a token is available or context is cancelled
	startWait := time.Now()

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter wait failed: %w", err)
	}

	waitDuration := time.Since(startWait)
	if waitDuration > time.Second {
		logger.Debug("Rate limiter delayed request", "wait_duration", waitDuration.Round(time.Millisecond))
	}

	var (
		response string
		err      error
	)

	if DebugMode {
		logger.Info("Original record", "record", string(recordJSON))

		// Send a prompt and get response with callbacks to see tool usage
		response, err = c.host.PromptWithCallbacks(
			ctx,
			prompt,
			func(name, args string) {
				logger.Info("Calling tool", "tool", name)
			},
			func(name, args, result string, isError bool) {
				if isError {
					logger.Error("Tool failed", "tool", name)
				} else {
					logger.Info("Tool completed", "tool", name)
				}
			},
			func(chunk string) {
			},
		)
		if err != nil {
			return "", fmt.Errorf("failed to send prompt: %w", err)
		}

		logger.Info("Response", "response", response)

		// Clear session after each prompt to prevent context accumulation
		c.host.ClearSession()

		return response, nil
	}

	// No debug, just send the prompt and get the response
	response, err = c.host.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	// Clear session after each prompt to prevent context accumulation.
	// Without this, the mcphost SDK accumulates conversation history across
	// all prompts, eventually hitting the model's context window limit.
	c.host.ClearSession()

	return response, nil
}

func (c *MCPHostClient) parseResponse(response string, fType fieldType) ([]EnrichedField, error) {
	// Trim the entire response first to remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Try to parse as structured JSON first
	var enrichmentResp EnrichmentResponse

	err := json.Unmarshal([]byte(response), &enrichmentResp)
	if err == nil {
		// Get the appropriate field list based on type
		var fields []EnrichedField

		switch fType {
		case fieldTypeSkills:
			fields = enrichmentResp.Skills
		case fieldTypeDomains:
			fields = enrichmentResp.Domains
		default:
			return nil, fmt.Errorf("unknown field type: %s", fType)
		}

		// Validate and filter fields
		validFields := make([]EnrichedField, 0, len(fields))
		for _, field := range fields {
			// Basic validation: must contain exactly one forward slash
			if strings.Count(field.Name, "/") != 1 {
				logger.Warn(fmt.Sprintf("Skipping invalid %s format (must be parent/child)", fType), "name", field.Name)

				continue
			}

			// Validate ID is provided
			if field.ID == 0 {
				logger.Warn(fmt.Sprintf("Skipping %s without valid ID", fType), "name", field.Name)

				continue
			}

			// Validate confidence is in valid range
			if field.Confidence < 0.0 || field.Confidence > 1.0 {
				logger.Warn(fmt.Sprintf("Skipping %s with invalid confidence", fType), "name", field.Name, "confidence", field.Confidence)

				continue
			}

			validFields = append(validFields, field)
		}

		if len(validFields) == 0 {
			return nil, fmt.Errorf("no valid %s found in JSON response", fType)
		}

		return validFields, nil
	}

	return nil, fmt.Errorf("failed to parse response: %w", err)
}

// enrichRecord handles the enrichment of a record with skills and domains.
func (e *Enricher) enrichRecord(ctx context.Context, recordStruct *structpb.Struct) error {
	// Detect schema version and convert structpb.Struct to appropriate OASF record type
	schemaVersion, err := getSchemaVersion(recordStruct)
	if err != nil {
		return fmt.Errorf("failed to get schema version: %w", err)
	}

	// Enrich based on schema version
	enrichedSkills, enrichedDomains, err := e.enrichRecordByVersion(ctx, recordStruct, schemaVersion)
	if err != nil {
		return err
	}

	// Update both skills and domains fields, preserve everything else from the original record
	if err := updateSkillsInStruct(recordStruct, enrichedSkills); err != nil {
		return fmt.Errorf("failed to update skills in record: %w", err)
	}

	if err := updateDomainsInStruct(recordStruct, enrichedDomains); err != nil {
		return fmt.Errorf("failed to update domains in record: %w", err)
	}

	return nil
}

// enrichRecordByVersion enriches a record based on its schema version.
func (e *Enricher) enrichRecordByVersion(ctx context.Context, recordStruct *structpb.Struct, schemaVersion string) ([]enrichedItem, []enrichedItem, error) {
	switch schemaVersion {
	case "1.0.0", "1.0.0-rc.1":
		return e.enrichV1Record(ctx, recordStruct)
	default:
		return e.enrichV1Alpha1Record(ctx, recordStruct)
	}
}

// enrichV1Record enriches a v1 (1.0.0) record.
//
//nolint:dupl // Similar structure to enrichV1Alpha1Record but uses different types
func (e *Enricher) enrichV1Record(ctx context.Context, recordStruct *structpb.Struct) ([]enrichedItem, []enrichedItem, error) {
	oasfRecord, err := structToOASFRecordV1(recordStruct)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert struct to OASF v1 record for enrichment: %w", err)
	}

	// Clear default skills and domains before enrichment - let the LLM select appropriate ones
	oasfRecord.Skills = nil
	oasfRecord.Domains = nil

	// Context with timeout for enrichment operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute) //nolint:mnd
	defer cancel()

	// Enrich with skills
	enrichedRecord, err := e.host.EnrichWithSkillsV1(ctxWithTimeout, oasfRecord)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enrich record with skills: %w", err)
	}

	// Enrich with domains (using the already skill-enriched record)
	enrichedRecord, err = e.host.EnrichWithDomainsV1(ctxWithTimeout, enrichedRecord)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enrich record with domains: %w", err)
	}

	// Extract skills and domains
	enrichedSkills := make([]enrichedItem, 0, len(enrichedRecord.GetSkills()))
	for _, skill := range enrichedRecord.GetSkills() {
		enrichedSkills = append(enrichedSkills, skill)
	}

	enrichedDomains := make([]enrichedItem, 0, len(enrichedRecord.GetDomains()))
	for _, domain := range enrichedRecord.GetDomains() {
		enrichedDomains = append(enrichedDomains, domain)
	}

	return enrichedSkills, enrichedDomains, nil
}

// enrichV1Alpha1Record enriches a v1alpha1 (0.7.0, 0.8.0) record.
//
//nolint:dupl // Similar structure to enrichV1Record but uses different types
func (e *Enricher) enrichV1Alpha1Record(ctx context.Context, recordStruct *structpb.Struct) ([]enrichedItem, []enrichedItem, error) {
	oasfRecord, err := structToOASFRecord(recordStruct)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert struct to OASF record for enrichment: %w", err)
	}

	// Clear default skills and domains before enrichment - let the LLM select appropriate ones
	oasfRecord.Skills = nil
	oasfRecord.Domains = nil

	// Context with timeout for enrichment operations
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute) //nolint:mnd
	defer cancel()

	// Enrich with skills
	enrichedRecord, err := e.host.EnrichWithSkills(ctxWithTimeout, oasfRecord)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enrich record with skills: %w", err)
	}

	// Enrich with domains (using the already skill-enriched record)
	enrichedRecord, err = e.host.EnrichWithDomains(ctxWithTimeout, enrichedRecord)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to enrich record with domains: %w", err)
	}

	// Extract skills and domains
	enrichedSkills := make([]enrichedItem, 0, len(enrichedRecord.GetSkills()))
	for _, skill := range enrichedRecord.GetSkills() {
		enrichedSkills = append(enrichedSkills, skill)
	}

	enrichedDomains := make([]enrichedItem, 0, len(enrichedRecord.GetDomains()))
	for _, domain := range enrichedRecord.GetDomains() {
		enrichedDomains = append(enrichedDomains, domain)
	}

	return enrichedSkills, enrichedDomains, nil
}

// getSchemaVersion extracts the schema_version from a structpb.Struct.
func getSchemaVersion(s *structpb.Struct) (string, error) {
	if s == nil {
		return "", errors.New("struct is nil")
	}

	fields := s.GetFields()
	if fields == nil {
		return "", errors.New("struct has no fields")
	}

	schemaVersionField, ok := fields["schema_version"]
	if !ok {
		return "", errors.New("schema_version field not found")
	}

	return schemaVersionField.GetStringValue(), nil
}

// structToOASFRecord converts a structpb.Struct to typesv1alpha1.Record for enrichment.
func structToOASFRecord(s *structpb.Struct) (*typesv1alpha1.Record, error) {
	if s == nil {
		return nil, errors.New("struct is nil")
	}
	// Marshal struct to JSON
	jsonBytes, err := protojson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	// Unmarshal JSON into typesv1alpha1.Record (discard unknown fields e.g. __mcp_debug_source on Data)
	var record typesv1alpha1.Record

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(jsonBytes, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to OASF record: %w", err)
	}

	return &record, nil
}

// structToOASFRecordV1 converts a structpb.Struct to typesv1.Record for enrichment.
func structToOASFRecordV1(s *structpb.Struct) (*typesv1.Record, error) {
	if s == nil {
		return nil, errors.New("struct is nil")
	}
	// Marshal struct to JSON
	jsonBytes, err := protojson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct to JSON: %w", err)
	}

	// Unmarshal JSON into typesv1.Record (discard unknown fields e.g. __mcp_debug_source on Data)
	var record typesv1.Record

	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(jsonBytes, &record); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to OASF v1 record: %w", err)
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
func updateSkillsInStruct(recordStruct *structpb.Struct, enrichedSkills []enrichedItem) error {
	return updateFieldsInStruct(recordStruct, "skills", enrichedSkills)
}

// updateDomainsInStruct updates the domains field in a structpb.Struct with enriched domains.
// This preserves all other fields including schema_version, name, version, etc.
func updateDomainsInStruct(recordStruct *structpb.Struct, enrichedDomains []enrichedItem) error {
	return updateFieldsInStruct(recordStruct, "domains", enrichedDomains)
}
