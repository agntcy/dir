// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"errors"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/importer/types"
	"google.golang.org/protobuf/types/known/structpb"
)

// StaticEnricher injects pre-defined OASF skills and domains into every record.
// It is intended for testing environments where no LLM is available.
type StaticEnricher struct{}

// NewStaticEnricher creates a StaticEnricher that injects hardcoded OASF-valid skills and domains.
func NewStaticEnricher() *StaticEnricher { return &StaticEnricher{} }

var staticSkills = &structpb.ListValue{Values: []*structpb.Value{
	structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
		"name": structpb.NewStringValue("natural_language_processing/natural_language_understanding/contextual_comprehension"),
		"id":   structpb.NewNumberValue(10101), //nolint:mnd
	}}),
}}

var staticDomains = &structpb.ListValue{Values: []*structpb.Value{
	structpb.NewStructValue(&structpb.Struct{Fields: map[string]*structpb.Value{
		"name": structpb.NewStringValue("technology/software_engineering"),
		"id":   structpb.NewNumberValue(102), //nolint:mnd
	}}),
}}

// Enrich reads records from inputCh, injects the static skills and domains, and forwards them.
func (se *StaticEnricher) Enrich(ctx context.Context, inputCh <-chan *corev1.Record, result *types.Result) (<-chan *corev1.Record, <-chan error) {
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

				data := record.GetData()
				if data == nil || data.Fields == nil {
					result.Mu.Lock()
					result.FailedCount++
					result.Mu.Unlock()

					errCh <- errors.New("static enricher: record has nil data")

					return
				}

				data.Fields["skills"] = structpb.NewListValue(staticSkills)
				data.Fields["domains"] = structpb.NewListValue(staticDomains)

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
