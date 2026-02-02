// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import "google.golang.org/protobuf/proto"

func (x *Workload) PatchServices(services *WorkloadServices) {
	if services == nil {
		return
	}

	// Initialize if nil
	if x.GetServices() == nil {
		x.Services = &WorkloadServices{}
	}

	// Merge fields
	if x.GetServices().GetA2A() == nil {
		x.Services.A2A = services.GetA2A()
	}

	if x.GetServices().GetOasf() == nil {
		x.Services.Oasf = services.GetOasf()
	}
}

func (x *Workload) DeepCopy() *Workload {
	if x == nil {
		return nil
	}

	// Create via reflection
	cloned, _ := proto.Clone(x).(*Workload)

	return cloned
}
