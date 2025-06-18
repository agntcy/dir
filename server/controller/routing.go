// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/api/objectmanager"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/opencontainers/go-digest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var routingLogger = logging.Logger("controller/routing")

type RoutingController struct {
	routing types.RoutingAPI
	store   types.StoreAPI
}

func NewRoutingController(routing types.RoutingAPI, store types.StoreAPI) *RoutingController {
	return &RoutingController{
		routing: routing,
		store:   store,
	}
}

// Publish handles the business logic for publishing an object to a network
func (c *RoutingController) Publish(ctx context.Context, recordObject *objectmanager.RecordObject, network bool) error {
	routingLogger.Debug("Called common routing controller's Publish method", "recordObject", recordObject, "network", network)

	// For now, we need to convert the RecordObject to a coretypes.Object for the routing API
	// This is a temporary approach until the routing API is updated to work directly with RecordObject
	ref, agent, err := c.getAgent(ctx, recordObject)
	if err != nil {
		return err
	}

	// Convert back to RecordObject for the routing API
	// This is a workaround until we have proper converters
	// For now, we'll create a minimal RecordObject
	recordObj := &objectmanager.RecordObject{
		Cid:  ref.GetDigest(), // Using digest as CID for now
		Type: objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON,
		Record: &objectmanager.RecordObjectData{
			Data: &objectmanager.RecordObjectData_RecordV1Alpha1{
				RecordV1Alpha1: agent,
			},
		},
	}

	err = c.routing.Publish(ctx, recordObj, network)
	if err != nil {
		st := status.Convert(err)
		return status.Errorf(st.Code(), "failed to publish: %s", st.Message())
	}

	return nil
}

// Unpublish handles the business logic for unpublishing an object from a network
func (c *RoutingController) Unpublish(ctx context.Context, recordObject *objectmanager.RecordObject, network bool) error {
	routingLogger.Debug("Called common routing controller's Unpublish method", "recordObject", recordObject, "network", network)

	// For now, we need to convert the RecordObject to a coretypes.Object for the routing API
	// This is a temporary approach until the routing API is updated to work directly with RecordObject
	ref, agent, err := c.getAgent(ctx, recordObject)
	if err != nil {
		return err
	}

	// Convert back to RecordObject for the routing API
	// This is a workaround until we have proper converters
	// For now, we'll create a minimal RecordObject
	recordObj := &objectmanager.RecordObject{
		Cid:  ref.GetDigest(), // Using digest as CID for now
		Type: objectmanager.RecordObjectType_RECORD_OBJECT_TYPE_OASF_V1ALPHA1_JSON,
		Record: &objectmanager.RecordObjectData{
			Data: &objectmanager.RecordObjectData_RecordV1Alpha1{
				RecordV1Alpha1: agent,
			},
		},
	}

	err = c.routing.Unpublish(ctx, recordObj, network)
	if err != nil {
		st := status.Convert(err)
		return status.Errorf(st.Code(), "failed to unpublish: %s", st.Message())
	}

	return nil
}

// List handles the business logic for listing objects in a network
func (c *RoutingController) List(ctx context.Context, listRequest *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	routingLogger.Debug("Called common routing controller's List method", "listRequest", listRequest)

	itemChan, err := c.routing.List(ctx, listRequest)
	if err != nil {
		st := status.Convert(err)
		return nil, status.Errorf(st.Code(), "failed to list: %s", st.Message())
	}

	return itemChan, nil
}

// Search handles the business logic for searching objects in a network
func (c *RoutingController) Search(ctx context.Context, searchRequest interface{}) (<-chan interface{}, error) {
	routingLogger.Debug("Called common routing controller's Search method", "searchRequest", searchRequest)

	// This will be implemented when the routing API supports search
	return nil, status.Errorf(codes.Unimplemented, "search not implemented in common controller")
}

// getAgent handles the business logic for retrieving and validating an agent
func (c *RoutingController) getAgent(ctx context.Context, recordObject *objectmanager.RecordObject) (*coretypes.ObjectRef, *coretypes.Agent, error) {
	routingLogger.Debug("Called common routing controller's getAgent method", "recordObject", recordObject)

	// Convert objectmanager.RecordObject to coretypes.ObjectRef for validation
	// This is a temporary approach until we have proper converters
	if recordObject == nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "record object is required")
	}

	// Extract agent from RecordObject
	var agent *coretypes.Agent
	if recordObject.GetRecord() != nil && recordObject.GetRecord().GetRecordV1Alpha1() != nil {
		agent = recordObject.GetRecord().GetRecordV1Alpha1()
	} else {
		return nil, nil, status.Errorf(codes.InvalidArgument, "record object must contain a v1alpha1 agent")
	}

	// Create a temporary ObjectRef for validation
	// TODO: Use proper converter once implemented
	ref := &coretypes.ObjectRef{
		Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest: recordObject.GetCid(), // Using CID as digest for now
		Size:   0,                     // Will be updated after lookup
	}

	if ref.GetType() == "" {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object reference is required and must have a type")
	}

	if ref.GetDigest() == "" {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object reference must have a digest")
	}

	_, err := digest.Parse(ref.GetDigest())
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid digest: %v", err)
	}

	// Use the store controller to lookup the object
	lookupRef, err := c.store.Lookup(ctx, recordObject)
	if err != nil {
		st := status.Convert(err)
		return nil, nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	// Update ref with lookup results if available
	// Note: lookupRef is also a RecordObject, so we need to handle this properly
	// For now, we'll use the original ref and assume the lookup was successful

	// Validate agent size (approximate)
	agentJSON, err := json.Marshal(agent)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to marshal agent: %v", err)
	}

	if len(agentJSON) > 4*1024*1024 {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object size exceeds maximum limit of 4MB")
	}

	if ref.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object type must be %s", coretypes.ObjectType_OBJECT_TYPE_AGENT.String())
	}

	// Use the store controller to pull the object for validation
	reader, err := c.store.Pull(ctx, lookupRef)
	if err != nil {
		st := status.Convert(err)
		return nil, nil, status.Errorf(st.Code(), "failed to pull object: %s", st.Message())
	}
	defer reader.Close()

	// Validate that we can load the agent from the reader
	validatedAgent := &coretypes.Agent{}
	_, err = validatedAgent.LoadFromReader(reader)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load agent from reader: %v", err)
	}

	return ref, agent, nil
}
