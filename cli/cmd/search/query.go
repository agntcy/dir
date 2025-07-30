// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package search

import (
	"errors"
	"fmt"
	"strings"

	searchtypes "github.com/agntcy/dir/api/search/v1"
)

type Query []string

func (q *Query) String() string {
	return strings.Join(*q, ", ")
}

func (q *Query) Set(value string) error {
	if value == "" {
		return errors.New("empty query not allowed")
	}

	parts := strings.SplitN(value, "=", 2)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)

		if part == "" {
			return errors.New("invalid query format, empty field or value")
		}
	}

	if len(parts) < 2 {
		return errors.New("invalid query format, expected 'field=value'")
	}

	validQueryType := false

	for _, queryType := range searchtypes.ValidQueryTypes {
		if parts[0] == queryType {
			validQueryType = true

			break
		}
	}

	if !validQueryType {
		return fmt.Errorf(
			"invalid query type: %s, valid types are: %v",
			parts[0],
			strings.Join(searchtypes.ValidQueryTypes, ", "),
		)
	}

	*q = append(*q, value)

	return nil
}

func (q *Query) Type() string {
	return "query"
}

func (q *Query) ToAPIQueries() []*searchtypes.RecordQuery {
	queries := []*searchtypes.RecordQuery{}

	for _, item := range *q {
		parts := strings.SplitN(item, "=", 2)

		queries = append(queries, &searchtypes.RecordQuery{
			Type:  searchtypes.RecordQueryType(searchtypes.RecordQueryType_value[parts[0]]),
			Value: parts[1],
		})
	}

	return queries
}
