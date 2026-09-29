// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "C"

import (
	"context"
	"encoding/json"
)

// --- GetWorkload ---

type getWorkloadRequest struct {
	WorkloadID string `json:"workload_id"`
}

type workloadResponse struct {
	Workload json.RawMessage `json:"workload,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// GetWorkload retrieves a single runtimev1.Workload by ID.
//
//export GetWorkload
func GetWorkload(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(workloadResponse{Error: "unknown or closed client handle"})
	}

	var req getWorkloadRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(workloadResponse{Error: "invalid request: " + err.Error()})
	}

	workload, err := c.GetWorkload(context.Background(), req.WorkloadID)
	if err != nil {
		return toC(workloadResponse{Error: err.Error()})
	}

	raw, err := protoToRaw(workload)
	if err != nil {
		return toC(workloadResponse{Error: err.Error()})
	}

	return toC(workloadResponse{Workload: raw})
}

// --- ListWorkloads ---

type listWorkloadsRequest struct {
	Labels map[string]string `json:"labels,omitempty"`
}

// ListWorkloads lists runtimev1.Workload values matching the given labels.
// This wraps the already-batching client.ListWorkloads convenience method
// (built on the raw ListWorkloadsStream) rather than draining the stream
// itself.
//
//export ListWorkloads
func ListWorkloads(handle C.longlong, reqJSON *C.char) *C.char {
	c, ok := lookupClient(int64(handle))
	if !ok {
		return toC(arrayResponse{Error: "unknown or closed client handle"})
	}

	var req listWorkloadsRequest
	if err := json.Unmarshal([]byte(C.GoString(reqJSON)), &req); err != nil {
		return toC(arrayResponse{Error: "invalid request: " + err.Error()})
	}

	workloads, err := c.ListWorkloads(context.Background(), req.Labels)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	results, err := marshalProtoSlice(workloads)
	if err != nil {
		return toC(arrayResponse{Error: err.Error()})
	}

	return toC(arrayResponse{Results: results})
}
