// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"fmt"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
)

type Extension struct {
	*objectsv1.Extension
}

func (e *Extension) Key() string {
	return fmt.Sprintf("%s/%s", e.GetName(), e.GetVersion())
}

func (e *Extension) ToOASFSchema() *objectsv1.Extension {
	if e == nil || e.Extension == nil {
		return nil
	}

	return e.Extension
}
