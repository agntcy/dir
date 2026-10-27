// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	oasfvalidator "github.com/agntcy/oasf-sdk/pkg/validator"
)

// oasfValidator adapts the OASF SDK validator, whose method is ValidateRecord.
type oasfValidator struct {
	inner corev1.Validator
}

func newOASFValidator(schemaURL string) (oasfValidator, error) {
	inner, err := oasfvalidator.New(schemaURL)
	if err != nil {
		return oasfValidator{}, fmt.Errorf("create OASF validator: %w", err)
	}

	return oasfValidator{inner: inner}, nil
}

// ValidateOperation validates the record against the schema. It reads only the
// record out of the input: a schema has no interest in the circumstances of
// the operation, which is why oasf-sdk's own signature needs nothing from us.
func (v oasfValidator) ValidateOperation(ctx context.Context, input *policyv1.OperationInput) (bool, []string, []string, error) {
	if v.inner == nil {
		return false, nil, nil, errors.New("OASF validator is nil")
	}

	valid, errs, warnings, err := v.inner.ValidateRecord(ctx, input.GetRecord())

	// validateOperation wraps the validator error. Wrapping again here would
	// change the text OASF failures already report.
	return valid, errs, warnings, err //nolint:wrapcheck
}

func (v oasfValidator) ValidateEvaluation(context.Context, *policyv1.RecordState) (bool, []string, []string, error) {
	return false, nil, nil, errors.New("provider \"oasf\" cannot define an evaluation policy")
}
