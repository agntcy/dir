// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	enricherconfig "github.com/agntcy/dir/importer/enricher/config"
	"github.com/agntcy/dir/importer/enricher/toolhost"
	"github.com/agntcy/dir/importer/types"
	"github.com/agntcy/dir/utils/logging"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/structpb"
)

var logger = logging.Logger("importer/enricher")

const (
	defaultConfidenceThreshold = 0.5
	enrichmentTimeout          = 5 * time.Minute
)

// Enricher fills OASF skills and domains on agent records using the tool host and prompt templates.
type Enricher struct {
	toolHost       *toolhost.Host
	skillsPrompt   string
	domainsPrompt  string
	requestLimiter *rate.Limiter
}

// EnrichedField is one model-predicted taxonomy entry with optional reasoning.
type EnrichedField struct {
	Name       string  `json:"name"`
	ID         uint32  `json:"id"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// New builds an Enricher from validated config.
func New(ctx context.Context, cfg enricherconfig.Config) (*Enricher, error) {
	th, err := toolhost.NewFromConfigFile(ctx, cfg.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("enricher tool host: %w", err)
	}

	return &Enricher{
		toolHost:       th,
		skillsPrompt:   cfg.SkillsPromptTemplate,
		domainsPrompt:  cfg.DomainsPromptTemplate,
		requestLimiter: rate.NewLimiter(rate.Limit(float64(cfg.RequestsPerMinute)/60.0), 1), //nolint:mnd
	}, nil
}

// Enrich reads records from inputCh, enriches each, and sends them on the returned channel.
func (e *Enricher) Enrich(ctx context.Context, inputCh <-chan *corev1.Record, result *types.Result) (<-chan *corev1.Record, <-chan error) {
	out := make(chan *corev1.Record)
	errCh := make(chan error)

	go func() {
		defer close(out)
		defer close(errCh)

		for {
			select {
			case <-ctx.Done():
				return
			case rec, ok := <-inputCh:
				if !ok {
					return
				}

				if err := e.enrichRecord(ctx, rec.GetData()); err != nil {
					result.Mu.Lock()
					result.FailedCount++
					result.Mu.Unlock()

					errCh <- fmt.Errorf("enrich record: %w", err)

					return
				}

				select {
				case out <- rec:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, errCh
}

func (e *Enricher) enrichRecord(ctx context.Context, s *structpb.Struct) error {
	wrapped := &corev1.Record{Data: s}

	decoded, err := wrapped.Decode()
	if err != nil {
		return fmt.Errorf("decode OASF record: %w", err)
	}

	if !decoded.HasV1() {
		return errors.New("enricher supports OASF v1 records only")
	}

	rec := decoded.GetV1()
	rec.Skills, rec.Domains = nil, nil

	ctx, cancel := context.WithTimeout(ctx, enrichmentTimeout)
	defer cancel()

	rec, err = e.enrichSkills(ctx, rec)
	if err != nil {
		return fmt.Errorf("skills: %w", err)
	}

	rec, err = e.enrichDomains(ctx, rec)
	if err != nil {
		return fmt.Errorf("domains: %w", err)
	}

	if err := setStructSkills(s, rec.GetSkills()); err != nil {
		return fmt.Errorf("write skills: %w", err)
	}

	if err := setStructDomains(s, rec.GetDomains()); err != nil {
		return fmt.Errorf("write domains: %w", err)
	}

	return nil
}

//nolint:dupl // Parallel skills vs domains enrichment; a shared helper would blur separate flows.
func (e *Enricher) enrichSkills(ctx context.Context, rec *typesv1.Record) (*typesv1.Record, error) {
	payload, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("marshal record: %w", err)
	}

	text, err := e.complete(ctx, e.skillsPrompt, payload)
	if err != nil {
		return nil, err
	}

	fields, err := parseSkillsJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse skills json: %w", err)
	}

	for _, f := range fields {
		if f.Confidence < defaultConfidenceThreshold {
			logger.Debug("skipped low-confidence skill", "name", f.Name, "confidence", f.Confidence)

			continue
		}

		rec.Skills = append(rec.Skills, &typesv1.Skill{Name: f.Name, Id: f.ID})
		logger.Debug("added skill", "name", f.Name, "id", f.ID)
	}

	return rec, nil
}

//nolint:dupl // Parallel skills vs domains enrichment; a shared helper would blur separate flows.
func (e *Enricher) enrichDomains(ctx context.Context, rec *typesv1.Record) (*typesv1.Record, error) {
	payload, err := json.Marshal(rec)
	if err != nil {
		return nil, fmt.Errorf("marshal record: %w", err)
	}

	text, err := e.complete(ctx, e.domainsPrompt, payload)
	if err != nil {
		return nil, err
	}

	fields, err := parseDomainsJSON(text)
	if err != nil {
		return nil, fmt.Errorf("parse domains json: %w", err)
	}

	for _, f := range fields {
		if f.Confidence < defaultConfidenceThreshold {
			logger.Debug("skipped low-confidence domain", "name", f.Name, "confidence", f.Confidence)

			continue
		}

		rec.Domains = append(rec.Domains, &typesv1.Domain{Name: f.Name, Id: f.ID})
		logger.Debug("added domain", "name", f.Name, "id", f.ID)
	}

	return rec, nil
}

// complete sends promptTemplate + JSON record to the LLM (rate-limited).
func (e *Enricher) complete(ctx context.Context, promptTemplate string, recordJSON []byte) (string, error) {
	prompt := promptTemplate + string(recordJSON)

	if err := e.requestLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit: %w", err)
	}

	text, err := e.toolHost.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}

	e.toolHost.ClearSession()

	return text, nil
}

func parseSkillsJSON(response string) ([]EnrichedField, error) {
	var parsed struct {
		Skills []EnrichedField `json:"skills"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &parsed); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	return parsed.Skills, nil
}

func parseDomainsJSON(response string) ([]EnrichedField, error) {
	var parsed struct {
		Domains []EnrichedField `json:"domains"`
	}

	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &parsed); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	return parsed.Domains, nil
}

func setStructSkills(s *structpb.Struct, skills []*typesv1.Skill) error {
	if s.Fields == nil {
		return errors.New("record struct has no fields")
	}

	lv := &structpb.ListValue{Values: make([]*structpb.Value, 0, len(skills))}
	for _, sk := range skills {
		st := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		if sk.GetName() != "" {
			st.Fields["name"] = structpb.NewStringValue(sk.GetName())
		}

		if sk.GetId() != 0 {
			st.Fields["id"] = structpb.NewNumberValue(float64(sk.GetId()))
		}

		lv.Values = append(lv.Values, structpb.NewStructValue(st))
	}

	s.Fields["skills"] = structpb.NewListValue(lv)

	return nil
}

func setStructDomains(s *structpb.Struct, domains []*typesv1.Domain) error {
	if s.Fields == nil {
		return errors.New("record struct has no fields")
	}

	lv := &structpb.ListValue{Values: make([]*structpb.Value, 0, len(domains))}
	for _, d := range domains {
		st := &structpb.Struct{Fields: map[string]*structpb.Value{}}
		if d.GetName() != "" {
			st.Fields["name"] = structpb.NewStringValue(d.GetName())
		}

		if d.GetId() != 0 {
			st.Fields["id"] = structpb.NewNumberValue(float64(d.GetId()))
		}

		lv.Values = append(lv.Values, structpb.NewStructValue(st))
	}

	s.Fields["domains"] = structpb.NewListValue(lv)

	return nil
}
