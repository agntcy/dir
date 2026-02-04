// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"strings"

	"google.golang.org/protobuf/proto"
)

func (x *Workload) DeepCopy() *Workload {
	if x == nil {
		return nil
	}

	// Create via reflection
	cloned, _ := proto.Clone(x).(*Workload)

	return cloned
}

// GetName returns a short lowercased name of the RuntimeType enum value.
// For example, RUNTIME_TYPE_DOCKER becomes "docker".
func (x RuntimeType) GetName() string {
	return strings.TrimPrefix(strings.ToLower(x.String()), "runtime_type_")
}

// GetName returns a short lowercased name of the WorkloadType enum value.
// For example, WORKLOAD_TYPE_PROCESS becomes "process".
func (x WorkloadType) GetName() string {
	return strings.TrimPrefix(strings.ToLower(x.String()), "workload_type_")
}
