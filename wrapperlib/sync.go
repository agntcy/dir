// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client"
)

// --- CreateSync ---

type createSyncRequest struct {
	RemoteURL         string   `json:"remote_url"`
	CIDs              []string `json:"cids,omitempty"`
	RemoteRegistryURL string   `json:"remote_registry_url,omitempty"`
	RepositoryName    string   `json:"repository_name,omitempty"`
}

type createSyncResponse struct {
	SyncID string `json:"sync_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

// CreateSync creates a new sync job for the given remote and returns its
// generated sync ID.
//
//export CreateSync
func CreateSync(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(createSyncResponse{Error: "unknown or closed client handle"})
	}

	var req createSyncRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(createSyncResponse{Error: "invalid request: " + err.Error()})
	}

	var opts *client.CreateSyncOptions
	if req.RemoteRegistryURL != "" || req.RepositoryName != "" {
		opts = &client.CreateSyncOptions{
			RemoteRegistryURL: req.RemoteRegistryURL,
			RepositoryName:    req.RepositoryName,
		}
	}

	syncID, err := c.CreateSync(context.Background(), req.RemoteURL, req.CIDs, opts)
	if err != nil {
		return toC(createSyncResponse{Error: err.Error()})
	}

	return toC(createSyncResponse{SyncID: syncID})
}

// --- ListSyncs ---

// ListSyncs sends a protojson-encoded storev1.ListSyncsRequest and drains the
// resulting channel into an arrayResponse whose Results are
// protojson-encoded storev1.ListSyncsItem values.
//
//export ListSyncs
func ListSyncs(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req storev1.ListSyncsRequest
	if err := unmarshalProto(C.GoString(reqJSON), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	ch, err := c.ListSyncs(context.Background(), &req)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	items := make([]*storev1.ListSyncsItem, 0)
	for item := range ch {
		items = append(items, item)
	}

	results, err := marshalProtoSlice(items)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}

// --- GetSync ---

type getSyncRequest struct {
	SyncID string `json:"sync_id"`
}

type getSyncResponse struct {
	Sync  json.RawMessage `json:"sync,omitempty"`
	Error string          `json:"error,omitempty"`
}

// GetSync retrieves a single sync job's status by ID.
//
//export GetSync
func GetSync(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(getSyncResponse{Error: "unknown or closed client handle"})
	}

	var req getSyncRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(getSyncResponse{Error: "invalid request: " + err.Error()})
	}

	resp, err := c.GetSync(context.Background(), req.SyncID)
	if err != nil {
		return toC(getSyncResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(resp)
	if err != nil {
		return toC(getSyncResponse{Error: err.Error()})
	}

	return toC(getSyncResponse{Sync: raw})
}

// --- DeleteSync ---

type deleteSyncRequest struct {
	SyncID string `json:"sync_id"`
}

// DeleteSync removes a sync job by ID.
//
//export DeleteSync
func DeleteSync(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(errorResponse{Error: "unknown or closed client handle"})
	}

	var req deleteSyncRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(errorResponse{Error: "invalid request: " + err.Error()})
	}

	if err := c.DeleteSync(context.Background(), req.SyncID); err != nil {
		return toC(errorResponse{Error: err.Error()})
	}

	return toC(errorResponse{})
}
