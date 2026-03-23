// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package test_utils

import (
	"time"

	"github.com/brianvoe/gofakeit/v7"
)

var GofakeitGenericLookups = map[string]gofakeit.Info{
	"pastdaterfc3339": {
		Display:     "PastDateRFC3339",
		Category:    "datetime",
		Description: "PastDateRFC3339 is the same as PastDate but in RFC-3339 format",
		Example:     "2024-09-10T23:20:50.520Z",
		Output:      "string",
		Generate: func(f *gofakeit.Faker, m *gofakeit.MapParams, info *gofakeit.Info) (any, error) {
			return f.PastDate().Format(time.RFC3339), nil
		},
	},
}

func InitGofakeit() {
	AddFuncLookups(GofakeitGenericLookups)
	AddFuncLookups(GofakeitOASF100Lookups)
}

func AddFuncLookups(lookups map[string]gofakeit.Info) {
	for functionName, info := range lookups {
		gofakeit.AddFuncLookup(functionName, info)
	}
}
