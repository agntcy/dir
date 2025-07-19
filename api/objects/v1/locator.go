// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package objectsv1

import "fmt"

func init() {
	// Override allowed names for locator types
	LocatorType_name = map[int32]string{
		0: "unspecified",
		1: "helm-chart",
		2: "docker-image",
		3: "python-package",
		4: "source-code",
		5: "binary",
	}
	LocatorType_value = map[string]int32{
		"":               0,
		"unspecified":    0,
		"helm-chart":     1,
		"docker-image":   2,
		"python-package": 3,
		"source-code":    4,
		"binary":         5,
	}
}

func (l *Locator) Key() string {
	return fmt.Sprintf("%s/%s", l.GetType(), l.GetUrl())
}
