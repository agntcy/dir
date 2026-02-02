// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import "google.golang.org/protobuf/proto"

func (x *Workload) DeepCopy() *Workload {
	if x == nil {
		return nil
	}

	// Create via reflection
	cloned, _ := proto.Clone(x).(*Workload)

	return cloned
}
