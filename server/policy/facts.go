// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/types"
)

// FactSource reads what this node has stored about a record. The evaluator
// projects it into the meta an evaluation policy sees.
//
// A nil FactSource leaves meta unset, because nothing was loaded. A source
// that returns a zero value means the node looked and found no signature,
// no scan and no claim.
type FactSource interface {
	Facts(ctx context.Context, cid string) (types.PolicyFacts, error)
}

// dbFacts reads facts from the database the node already writes them to.
type dbFacts struct {
	db types.PolicyFactDatabaseAPI
}

// NewFactSource returns a FactSource backed by db. A nil database returns nil,
// which leaves meta unset.
func NewFactSource(db types.PolicyFactDatabaseAPI) FactSource {
	if db == nil {
		return nil
	}

	return dbFacts{db: db}
}

func (d dbFacts) Facts(_ context.Context, cid string) (types.PolicyFacts, error) {
	facts, err := d.db.GetPolicyFacts(cid)
	if err != nil {
		return types.PolicyFacts{}, fmt.Errorf("read policy facts for %s: %w", cid, err)
	}

	return facts, nil
}
