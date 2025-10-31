// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package service provides reusable business logic for record operations in the Record Hub CLI and related applications.
package service

import (
	"context"
	"encoding/json"
	"fmt"

	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	authUtils "github.com/agntcy/dir/hub/auth/utils"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	"github.com/agntcy/dir/hub/sessionstore"
)

// PullRecord pulls an record from the hub and returns the pretty-printed JSON.
// It uses the provided session for authentication.
func PullRecord(
	ctx context.Context,
	hc hubClient.Client,
	cid string,
	session *sessionstore.HubSession,
) ([]byte, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	model, err := hc.PullRecord(ctx, cid)
	if err != nil {
		return nil, fmt.Errorf("failed to pull record: %w", err)
	}

	var modelObj map[string]interface{}
	if err = json.Unmarshal(model, &modelObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal record: %w", err)
	}

	prettyModel, err := json.MarshalIndent(modelObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	return prettyModel, nil
}

// PushRecord pushes a record to the hub and returns the response.
// It uses the provided session for authentication.
func PushRecord(
	ctx context.Context,
	hc hubClient.Client,
	organization string,
	recordBytes []byte,
	session *sessionstore.HubSession,
) (*v1alpha1.PushRecordResponse, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	resp, err := hc.PushRecord(ctx, organization, recordBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to push record: %w", err)
	}

	return resp, nil
}

func PushRecordSignature(ctx context.Context, hc hubClient.Client, organization, recordCID, signature, publicKey string, session *sessionstore.HubSession) error {
	ctx = authUtils.AddAuthToContext(ctx, session)

	err := hc.PushRecordSignature(ctx, organization, recordCID, signature, publicKey)
	if err != nil {
		return fmt.Errorf("unable to push signature: %w", err)
	}

	return nil
}

func GetRecordSignatures(ctx context.Context, hc hubClient.Client, recordCID string, session *sessionstore.HubSession) ([]*v1alpha1.RecordSignature, error) {
	ctx = authUtils.AddAuthToContext(ctx, session)

	sigs, err := hc.GetRecordSignatures(ctx, recordCID)
	if err != nil {
		return nil, fmt.Errorf("unable to get signatures: %w", err)
	}

	return sigs, nil
}
