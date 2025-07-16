// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package corev1alpha1

import (
	"fmt"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
)

type Locator struct {
	*objectsv1.Locator
}

func (l *Locator) Key() string {
	return fmt.Sprintf("%s/%s", l.GetType(), l.GetUrl())
}

func (l *Locator) ToOASFSchema() *objectsv1.Locator {
	if l == nil || l.Locator == nil {
		return nil
	}

	return l.Locator
}
