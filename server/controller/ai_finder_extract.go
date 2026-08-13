// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// textMaxLen mirrors the proto validator (max_len=1024) on
// ExtractTaxonomyRequest.text.
const textMaxLen = 1024

// ExtractTaxonomy maps free-form text onto the OASF taxonomy.
//
// It is a passthrough over the extractor the gateway already resolved for the
// other extractor-backed RPCs: text in, the skills and domains the extractor
// matched out, with the scores and tiers that ranked them. No search or catalog
// semantics are attached — in particular, the returned classes are not checked
// against what any record in this registry actually carries. Callers that need
// the suggestions to be actionable should intersect them against ListTags.
//
// ExtractOptions is left zero so the backend applies extractor.DefaultTiers,
// keeping the classes offered here in step with the ones search matches on.
func (c *aiFinderController) ExtractTaxonomy(ctx context.Context, req *catalogv1.ExtractTaxonomyRequest) (*catalogv1.ExtractTaxonomyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	// Enforce the proto's max_len in code: the service registers no protovalidate
	// interceptor, so the declared constraint is not applied at runtime and an
	// unbounded string would otherwise reach the extractor. Checked before
	// trimming, so padding cannot smuggle a longer payload past it. Mirrors how
	// ListAgents enforces filterMaxLen.
	if len(req.GetText()) > textMaxLen {
		return nil, status.Errorf(codes.InvalidArgument, "text too long (%d > %d)", len(req.GetText()), textMaxLen)
	}

	text := strings.TrimSpace(req.GetText())
	if text == "" {
		return nil, status.Error(codes.InvalidArgument, "text is required") //nolint:wrapcheck
	}

	// Log the length rather than the text: the input is user-supplied free form
	// and may contain sensitive terms.
	aiFinderLogger.Debug("ExtractTaxonomy called", "text_len", len(text))

	if c.ext == nil {
		return nil, status.Error(codes.Unavailable, //nolint:wrapcheck
			"taxonomy extraction is unavailable: no OASF extractor is configured on this gateway")
	}

	res, err := c.ext.Extract(ctx, text, extractor.ExtractOptions{})
	if err != nil {
		aiFinderLogger.Error("failed to extract taxonomy", "text_len", len(text), "error", err)

		return nil, status.Error(codes.Unavailable, "taxonomy extraction is temporarily unavailable") //nolint:wrapcheck
	}

	// Modules are dropped: they describe how a record is deployed rather than
	// what it does, so they are not discovery labels and carry no catalog tag.
	return &catalogv1.ExtractTaxonomyResponse{
		Skills:   toScoredClasses(res.Skills),
		Domains:  toScoredClasses(res.Domains),
		Keywords: toKeywordTexts(res.Keywords),
	}, nil
}

// toScoredClasses maps extractor classes onto the wire type, preserving the
// extractor's descending-score order.
func toScoredClasses(classes []sdk.ScoredClass) []*catalogv1.ScoredClass {
	if len(classes) == 0 {
		return nil
	}

	out := make([]*catalogv1.ScoredClass, 0, len(classes))
	for _, cl := range classes {
		out = append(out, &catalogv1.ScoredClass{
			// OASF uids are small integers; uint32 keeps them JSON numbers rather
			// than the strings protojson emits for 64-bit fields.
			Id:    uint32(cl.ID), //nolint:gosec
			Name:  cl.Name,
			Score: cl.Score,
			Tier:  uint32(cl.Tier), //nolint:gosec
		})
	}

	return out
}

// toKeywordTexts drops the keyword scores: keywords have no OASF class behind
// them, so there is nothing to threshold or rank them against.
func toKeywordTexts(keywords []sdk.Keyword) []string {
	if len(keywords) == 0 {
		return nil
	}

	out := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		out = append(out, kw.Text)
	}

	return out
}
