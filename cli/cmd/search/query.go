// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"errors"
	"fmt"
	"strings"

	searchtypesv1alpha2 "github.com/agntcy/dir/api/search/v1alpha2"
)

type Query []*searchtypesv1alpha2.RecordQuery

func (q *Query) String() string {
	queryStrings := make([]string, 0, len(*q))

	for _, query := range *q {
		queryStrings = append(queryStrings,
			fmt.Sprintf("%s=%s", searchtypesv1alpha2.RecordQueryType_name[int32(query.GetType())], query.GetValue()),
		)
	}

	return strings.Join(queryStrings, " ")
}

func (q *Query) Set(value string) error {
	if value == "" {
		return errors.New("empty query not allowed")
	}

	parts := strings.SplitN(value, "=", 2) //nolint:mnd
	if len(parts) != 2 {                   //nolint:mnd
		return errors.New("invalid query format, expected 'field=value'")
	}

	if _, ok := searchtypesv1alpha2.RecordQueryType_value[parts[0]]; !ok {
		return fmt.Errorf(
			"invalid query type: %s, valid types are: %v",
			parts[0],
			searchtypesv1alpha2.ValidQueryTypes,
		)
	}

	queryType := parts[0]
	queryValues := parts[1]

	if queryType == "" {
		return fmt.Errorf("invalid query type: %s", queryType)
	}

	*q = append(*q, &searchtypesv1alpha2.RecordQuery{
		Type:  searchtypesv1alpha2.RecordQueryType(searchtypesv1alpha2.RecordQueryType_value[queryType]),
		Value: queryValues,
	})

	return nil
}

func (q *Query) Type() string {
	return "query"
}

func (q *Query) ToQuery() []*searchtypesv1alpha2.RecordQuery {
	if q == nil {
		return nil
	}

	return *q
}
