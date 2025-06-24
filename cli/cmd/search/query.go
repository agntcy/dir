// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:mnd
package search

import (
	"errors"
	"fmt"
	"strings"

	searchtypesv1alpha2 "github.com/agntcy/dir/api/search/v1alpha2"
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

	for _, queryType := range searchtypesv1alpha2.ValidQueryTypes {
		if parts[0] == queryType {
			validQueryType = true

			break
		}
	}

	if !validQueryType {
		return fmt.Errorf(
			"invalid query type: %s, valid types are: %v",
			parts[0],
			strings.Join(searchtypesv1alpha2.ValidQueryTypes, ", "),
		)
	}

	*q = append(*q, value)

	return nil
}

func (q *Query) Type() string {
	return "query"
}

func (q *Query) ToAPIQueries() []*searchtypesv1alpha2.RecordQuery {
	queries := []*searchtypesv1alpha2.RecordQuery{}

	for _, item := range *q {
		parts := strings.SplitN(item, "=", 2)

		queries = append(queries, &searchtypesv1alpha2.RecordQuery{
			Type:  searchtypesv1alpha2.RecordQueryType(searchtypesv1alpha2.RecordQueryType_value[parts[0]]),
			Value: parts[1],
		})
	}

	return queries
}
