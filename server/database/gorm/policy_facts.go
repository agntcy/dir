// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/server/types"
	"gorm.io/gorm"
)

// GetPolicyFacts returns the stored observations of cid. A CID that is not
// indexed returns a zero value and no error: there is nothing to project.
func (d *DB) GetPolicyFacts(cid string) (types.PolicyFacts, error) {
	var record Record

	err := d.gormDB.
		Preload("Signatures").
		Preload("ScanReports").
		Preload("IdentityClaims").
		Where("record_cid = ?", cid).
		Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return types.PolicyFacts{}, nil
	}

	if err != nil {
		return types.PolicyFacts{}, fmt.Errorf("get policy facts for %s: %w", cid, err)
	}

	facts := types.PolicyFacts{
		Signed:     record.Signed,
		Signatures: make([]types.SignatureVerificationObject, len(record.Signatures)),
		Scans:      make([]types.ScanReportObject, len(record.ScanReports)),
		Claims:     make([]types.IdentityClaimObject, len(record.IdentityClaims)),
	}

	for i := range record.Signatures {
		facts.Signatures[i] = &record.Signatures[i]
	}

	for i := range record.ScanReports {
		facts.Scans[i] = &record.ScanReports[i]
	}

	for i := range record.IdentityClaims {
		facts.Claims[i] = &record.IdentityClaims[i]
	}

	return facts, nil
}
