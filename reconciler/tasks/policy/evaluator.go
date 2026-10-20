// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import serverpolicy "github.com/agntcy/dir/server/policy"

// Evaluator checks records against one evaluation policy. The policy registry
// builds them. This task schedules them and stores the verdicts.
type Evaluator = serverpolicy.Evaluator

// RecordSource reads a record's content from the store. The task selects
// records from the index, which holds only the fields searches use, and the
// policy decides on the record as published.
type RecordSource = serverpolicy.RecordSource
