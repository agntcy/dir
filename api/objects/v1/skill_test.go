// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package objectsv1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkill_Key(t *testing.T) {
	testCases := []struct {
		name         string
		categoryName string
		className    string
		expectedKey  string
	}{
		{
			name:         "Category and Class Name present",
			categoryName: "category1",
			className:    "class1",
			expectedKey:  "category1/class1",
		},
		{
			name:         "Only Category Name present",
			categoryName: "category2",
			className:    "",
			expectedKey:  "category2",
		},
		{
			name:         "Empty Category and Class Name",
			categoryName: "",
			className:    "",
			expectedKey:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a Skill instance with test data
			skill := &Skill{
				CategoryName: &tc.categoryName,
				ClassName:    &tc.className,
			}

			// Call the Key method
			key := skill.Key()

			// Assert the result
			assert.Equal(t, tc.expectedKey, key)
		})
	}
}
