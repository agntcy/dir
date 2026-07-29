// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"fmt"

	extractorv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/extractor/v1/extractorv1grpc"
	extractorv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/extractor/v1"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// remoteExtractor implements Extractor by calling a gRPC OASF-SDK server. The
// heavy model and taxonomy live in the server, so this backend carries none of
// the local assets.
type remoteExtractor struct {
	client extractorv1grpc.ExtractorServiceClient
}

// Extract forwards the text and any version pins to the server and maps the
// response back into the SDK Result shape.
func (r *remoteExtractor) Extract(ctx context.Context, text string, opts ExtractOptions) (Result, error) {
	resp, err := r.client.Extract(ctx, extractorv1.ExtractRequest_builder{
		Text:     text,
		Versions: opts.Versions,
	}.Build())
	if err != nil {
		return Result{}, fmt.Errorf("remote extract: %w", err)
	}

	return responseToResult(resp), nil
}

// responseToResult maps a gRPC ExtractResponse into the SDK Result the rest of
// the code expects. The shapes are near-identical, so this is a field copy.
func responseToResult(resp *extractorv1.ExtractResponse) Result {
	return Result{
		Skills:   scoredClasses(resp.GetSkills()),
		Domains:  scoredClasses(resp.GetDomains()),
		Modules:  scoredClasses(resp.GetModules()),
		Keywords: keywords(resp.GetKeywords()),
	}
}

func scoredClasses(in []*extractorv1.ScoredClass) []sdk.ScoredClass {
	if len(in) == 0 {
		return nil
	}

	out := make([]sdk.ScoredClass, 0, len(in))
	for _, c := range in {
		out = append(out, sdk.ScoredClass{
			Class: sdk.Class{
				ID:          c.GetId(),
				Name:        c.GetName(),
				Caption:     c.GetCaption(),
				Description: c.GetDescription(),
			},
			Kind:     classKind(c.GetKind()),
			Versions: c.GetVersions(),
			Score:    c.GetScore(),
			Semantic: c.GetSemantic(),
			Lexical:  c.GetLexical(),
			Tier:     int(c.GetTier()),
		})
	}

	return out
}

func keywords(in []*extractorv1.Keyword) []sdk.Keyword {
	if len(in) == 0 {
		return nil
	}

	out := make([]sdk.Keyword, 0, len(in))
	for _, k := range in {
		out = append(out, sdk.Keyword{Text: k.GetText(), Score: k.GetScore()})
	}

	return out
}

// classKind maps the proto ClassType enum onto the SDK Kind string.
func classKind(k extractorv1.ClassType) sdk.Kind {
	switch k {
	case extractorv1.ClassType_CLASS_TYPE_SKILL:
		return sdk.KindSkill
	case extractorv1.ClassType_CLASS_TYPE_DOMAIN:
		return sdk.KindDomain
	case extractorv1.ClassType_CLASS_TYPE_MODULE:
		return sdk.KindModule
	case extractorv1.ClassType_CLASS_TYPE_UNSPECIFIED:
		return ""
	}

	return ""
}
