// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	namingv1 "github.com/agntcy/dir/api/naming/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// VerifyNameInput defines the input parameters for verifying a record's name ownership.
type VerifyNameInput struct {
	CID     string `json:"cid,omitempty"     jsonschema:"Content Identifier (CID) of the record to verify"`
	Name    string `json:"name,omitempty"    jsonschema:"Human-readable name of the record (e.g., 'example.com/my-agent')"`
	Version string `json:"version,omitempty" jsonschema:"Optional version to verify (e.g., 'v1.0.0'). If omitted, verifies the latest version."`
}

// DomainVerificationDetail provides details about the domain verification.
type DomainVerificationDetail struct {
	Domain       string `json:"domain"                   jsonschema:"The domain that was verified"`
	Method       string `json:"method"                   jsonschema:"Verification method used (e.g., 'wellknown')"`
	MatchedKeyID string `json:"matched_key_id,omitempty" jsonschema:"The key ID that matched from the domain's well-known file"`
	VerifiedAt   string `json:"verified_at,omitempty"    jsonschema:"Timestamp when verification occurred"`
}

// VerifyNameOutput defines the output of name verification.
type VerifyNameOutput struct {
	Verified           bool                      `json:"verified"                      jsonschema:"Whether the name ownership was verified"`
	DomainVerification *DomainVerificationDetail `json:"domain_verification,omitempty" jsonschema:"Details about the domain verification (only present if verified)"`
	ErrorMessage       string                    `json:"error_message,omitempty"       jsonschema:"Error message if verification failed"`
}

// VerifyName verifies that a record's name is owned by the domain it claims.
// This checks that the record was signed with a key published in the domain's well-known JWKS file.
func (t *Tools) VerifyName(ctx context.Context, _ *mcp.CallToolRequest, input VerifyNameInput) (
	*mcp.CallToolResult,
	VerifyNameOutput,
	error,
) {
	// Validate input - need either CID or Name
	if input.CID == "" && input.Name == "" {
		return nil, VerifyNameOutput{
			ErrorMessage: "Either 'cid' or 'name' is required",
		}, nil
	}

	// Call the appropriate verification method
	var resp *namingv1.GetVerificationInfoResponse

	var err error

	if input.CID != "" {
		// Verify by CID
		resp, err = t.Client.GetVerificationInfo(ctx, input.CID)
		if err != nil {
			return nil, VerifyNameOutput{
				ErrorMessage: fmt.Sprintf("Failed to verify name: %v", err),
			}, nil
		}
	} else {
		// Verify by name (and optional version)
		resp, err = t.Client.GetVerificationInfoByName(ctx, input.Name, input.Version)
		if err != nil {
			return nil, VerifyNameOutput{
				ErrorMessage: fmt.Sprintf("Failed to verify name: %v", err),
			}, nil
		}
	}

	// Build output
	output := VerifyNameOutput{
		Verified: resp.GetVerified(),
	}

	if !resp.GetVerified() && resp.GetErrorMessage() != "" {
		output.ErrorMessage = resp.GetErrorMessage()
	}

	// Add domain verification details if available
	if v := resp.GetVerification(); v != nil {
		if dv := v.GetDomain(); dv != nil {
			output.DomainVerification = &DomainVerificationDetail{
				Domain:       dv.GetDomain(),
				Method:       dv.GetMethod(),
				MatchedKeyID: dv.GetKeyId(),
			}
			if dv.GetVerifiedAt() != nil {
				output.DomainVerification.VerifiedAt = dv.GetVerifiedAt().AsTime().String()
			}
		}
	}

	return nil, output, nil
}
