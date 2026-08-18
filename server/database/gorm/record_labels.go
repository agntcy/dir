// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"fmt"

	"github.com/agntcy/dir/server/types"
)

// labelRow is the shared scan target for the four label tables.
type labelRow struct {
	RecordCID string `gorm:"column:record_cid"`
	Name      string `gorm:"column:name"`
}

// GetRecordLabels returns the routing labels of each given record, keyed by CID.
//
// One query per label table rather than preloading whole records: callers want
// the labels and nothing else, and the associations on Record would drag in
// every column of every skill, module, domain and locator row.
func (d *DB) GetRecordLabels(cids []string) (map[string][]types.Label, error) {
	labels := make(map[string][]types.Label, len(cids))

	if len(cids) == 0 {
		return labels, nil
	}

	// Locators are keyed by type rather than name, matching how
	// types.GetLabelsFromRecord builds them.
	sources := []struct {
		table     string
		nameCol   string
		labelType types.LabelType
	}{
		{"skills", "name", types.LabelTypeSkill},
		{"domains", "name", types.LabelTypeDomain},
		{"modules", "name", types.LabelTypeModule},
		{"locators", "type", types.LabelTypeLocator},
	}

	for _, source := range sources {
		var rows []labelRow

		err := d.gormDB.
			Table(source.table).
			Select("record_cid, "+source.nameCol+" AS name").
			Where("record_cid IN ?", cids).
			Find(&rows).Error
		if err != nil {
			return nil, fmt.Errorf("failed to load %s labels: %w", source.table, err)
		}

		for _, row := range rows {
			labels[row.RecordCID] = append(labels[row.RecordCID], source.labelType.LabelKey(row.Name))
		}
	}

	return labels, nil
}
