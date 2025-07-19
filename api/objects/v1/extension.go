// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package objectsv1

import "fmt"

func (e *Extension) Key() string {
	return fmt.Sprintf("%s/%s", e.GetName(), e.GetVersion())
}
