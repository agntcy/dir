// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cids

import (
	"strings"
	"testing"
)

func TestReadFromJSONAndLines(t *testing.T) {
	for name, input := range map[string]string{
		"json":  `["bafy-one", "bafy-two"]`,
		"lines": "bafy-one\n\n bafy-two \n",
	} {
		t.Run(name, func(t *testing.T) {
			values, err := ReadFrom(strings.NewReader(input))
			if err != nil {
				t.Fatal(err)
			}
			got := Deduplicate(values)
			if len(got) != 2 || got[0] != "bafy-one" || got[1] != "bafy-two" {
				t.Fatalf("unexpected CIDs: %#v", got)
			}
		})
	}
}
