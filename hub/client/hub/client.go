// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package hub provides a client for interacting with the Agent Hub backend API, including agent management and related operations.
package hub

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	v1alpha1 "github.com/agntcy/dir/hub/api/v1alpha1"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Client defines the interface for interacting with the Agent Hub backend for agent operations.
type Client interface {
	// PushRecord uploads a record to the hub and returns the CID or an error.
	PushRecord(ctx context.Context, organization string, record []byte) (*v1alpha1.PushRecordResponse, error)
	// PullRecord downloads a record from the hub and returns the record data or an error.
	PullRecord(ctx context.Context, cid string) ([]byte, error)
	// CreateAPIKey creates an API key for the specified role and returns the (clientId, secret) or an error.
	CreateAPIKey(ctx context.Context, roleName string, organization any) (*v1alpha1.CreateApiKeyResponse, error)
	// DeleteAPIKey deletes an API key from the hub and returns the response or an error.
	DeleteAPIKey(ctx context.Context, clientID string) (*v1alpha1.DeleteApiKeyResponse, error)
	// ListAPIKeys lists all API keys for a specific organization and returns the response or an error.
	ListAPIKeys(ctx context.Context, organization any) (*v1alpha1.ListApiKeyResponse, error)
	PushRecordSignature(ctx context.Context, organization string, recordCID string, signature string, publicKey string) error
	GetRecordSignatures(ctx context.Context, recordCID string) ([]*v1alpha1.RecordSignature, error)
}

// client implements the Client interface for the Agent Hub backend.
type client struct {
	v1alpha1.RecordHubServiceClient
	v1alpha1.ApiKeyServiceClient
	v1alpha1.OrganizationServiceClient
	v1alpha1.UserServiceClient
}

// New creates a new Agent Hub client for the given server address.
// Returns the client or an error if the connection could not be established.
func New(serverAddr string) (*client, error) { //nolint:revive
	// Create connection
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return &client{
		RecordHubServiceClient:    v1alpha1.NewRecordHubServiceClient(conn),
		ApiKeyServiceClient:       v1alpha1.NewApiKeyServiceClient(conn),
		OrganizationServiceClient: v1alpha1.NewOrganizationServiceClient(conn),
		UserServiceClient:         v1alpha1.NewUserServiceClient(conn),
	}, nil
}

func (c *client) PushRecord(ctx context.Context, organization string, record []byte) (*v1alpha1.PushRecordResponse, error) { //nolint:cyclop
	parsedRecord, err := corev1.UnmarshalRecord(record)
	if err != nil {
		return nil, fmt.Errorf("failed to load OASF: %w", err)
	}

	recordName := parsedRecord.GetData().GetFields()["name"].GetStringValue()
	if recordName == "" {
		return nil, errors.New("record name is missing")
	}

	IdName := ParseOrganizationIdOrName(organization)

	req := &v1alpha1.PushRecordRequest{
		Model:    parsedRecord,
		IdOrName: IdName,
	}

	resp, err := c.RecordHubServiceClient.PushRecord(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to push record: %w", err)
	}

	return resp, nil
}

func (c *client) PushRecordSignature(ctx context.Context, organization string, recordCID string, signature string, publicKey string) error {
	IdName := ParseOrganizationIdOrName(organization)

	req := &v1alpha1.PushRecordSignatureRequest{
		IdOrName: IdName,
		Cid:      recordCID,
		Signature: &v1alpha1.RecordSignature{
			PublicKey: publicKey,
			Signature: signature,
		},
	}

	_, err := c.RecordHubServiceClient.PushRecordSignature(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to push record signature: %w", err)
	}

	return nil
}

func (c *client) GetRecordSignatures(ctx context.Context, recordCID string) ([]*v1alpha1.RecordSignature, error) {
	req := &v1alpha1.GetRecordSignaturesRequest{
		Cid: recordCID,
	}

	sigsResponse, err := c.RecordHubServiceClient.GetRecordSignatures(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get record signatures: %w", err)
	}

	return sigsResponse.GetSignatures(), nil
}

func (c *client) PullRecord(ctx context.Context, cid string) ([]byte, error) {
	resp, err := c.RecordHubServiceClient.PullRecord(ctx, &v1alpha1.PullRecordRequest{Cid: cid})
	if err != nil {
		return nil, fmt.Errorf("failed to pull record: %w", err)
	}

	b, err := resp.GetModel().GetData().MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("invalid record: %w", err)
	}

	return b, nil
}

func (c *client) CreateAPIKey(ctx context.Context, roleName string, organization any) (*v1alpha1.CreateApiKeyResponse, error) {
	roleValue, ok := v1alpha1.Role_value[roleName]
	if !ok {
		return nil, fmt.Errorf("unknown role: %d", roleValue)
	}

	req := &v1alpha1.CreateApiKeyRequest{
		Role: v1alpha1.Role(roleValue),
	}

	switch parsedOrg := organization.(type) {
	case *v1alpha1.CreateApiKeyRequest_OrganizationName:
		req.Organization = parsedOrg
	case *v1alpha1.CreateApiKeyRequest_OrganizationId:
		req.Organization = parsedOrg
	default:
		return nil, fmt.Errorf("unknown organization type: %T", organization)
	}

	stream, err := c.ApiKeyServiceClient.CreateAPIKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	var chunk *v1alpha1.CreateApiKeyResponse

	chunk, err = stream.Recv()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to receive chunk: %w", err)
	}

	return chunk, nil
}

func (c *client) DeleteAPIKey(ctx context.Context, clientID string) (*v1alpha1.DeleteApiKeyResponse, error) {
	req := &v1alpha1.DeleteApiKeyRequest{
		ClientId: clientID,
	}

	resp, err := c.ApiKeyServiceClient.DeleteAPIKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to delete API key: %w", err)
	}

	if resp == nil {
		return nil, errors.New("received nil response from delete api key")
	}

	return resp, nil
}

func (c *client) ListAPIKeys(ctx context.Context, organization any) (*v1alpha1.ListApiKeyResponse, error) {
	req := &v1alpha1.ListApiKeyRequest{}

	switch parsedOrg := organization.(type) {
	case *v1alpha1.ListApiKeyRequest_OrganizationName:
		req.Organization = parsedOrg
	case *v1alpha1.ListApiKeyRequest_OrganizationId:
		req.Organization = parsedOrg
	default:
		return nil, fmt.Errorf("unknown organization type: %T", organization)
	}

	resp, err := c.ListApiKey(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}

	if resp == nil {
		return nil, errors.New("received nil response from list api keys")
	}

	return resp, nil
}

func ParseOrganizationIdOrName(value string) *v1alpha1.IdOrName {
	// Try to parse as UUID first
	if _, err := uuid.Parse(value); err == nil {
		return &v1alpha1.IdOrName{
			IdOrName: &v1alpha1.IdOrName_Id{
				Id: value,
			},
		}
	}

	// Otherwise, treat it as a name
	return &v1alpha1.IdOrName{
		IdOrName: &v1alpha1.IdOrName_Name{
			Name: value,
		},
	}
}
