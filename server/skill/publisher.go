// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package skill

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("server/skill")

// Publish pushes the embedded SKILL.md as an OASF record. Idempotent on
// name+version. Callers must treat errors as non-fatal — DIR has to start
// even if publishing fails.
func Publish(ctx context.Context, store types.StoreAPI, db types.DatabaseAPI, validator corev1.Validator) error {
	if store == nil || db == nil {
		return errors.New("store and db must be provided")
	}

	if existingCID, ok := findExisting(db); ok {
		logger.Debug("DIR skill record already present, skipping publish",
			"cid", existingCID,
			"name", RecordName,
			"version", RecordVersion(),
		)

		return nil
	}

	record, err := BuildRecord(time.Now().UTC())
	if err != nil {
		return fmt.Errorf("build skill record: %w", err)
	}

	if err := validateRecord(ctx, record, validator); err != nil {
		return err
	}

	ref, err := store.Push(ctx, record)
	if err != nil {
		return fmt.Errorf("push skill record: %w", err)
	}

	// Update the search index in line with the gRPC store controller, so the
	// record is discoverable without waiting for an external push.
	decoded, decodeErr := record.Decode()
	if decodeErr != nil {
		logger.Warn("DIR skill record pushed but could not be decoded for search index",
			"cid", ref.GetCid(),
			"error", decodeErr,
		)
	} else if addErr := db.AddRecord(decoded); addErr != nil {
		logger.Warn("DIR skill record pushed but search index update failed",
			"cid", ref.GetCid(),
			"error", addErr,
		)
	}

	logger.Info("DIR skill record published",
		"cid", ref.GetCid(),
		"name", RecordName,
		"version", RecordVersion(),
		"digest", ContentDigest(),
	)

	return nil
}

func validateRecord(ctx context.Context, record *corev1.Record, validator corev1.Validator) error {
	if validator == nil {
		return nil
	}

	ok, msgs, err := record.ValidateWith(ctx, validator)
	if err != nil {
		return fmt.Errorf("validate skill record: %w", err)
	}

	if !ok {
		return fmt.Errorf("skill record failed OASF validation: %v", msgs)
	}

	return nil
}

func findExisting(db types.DatabaseAPI) (string, bool) {
	cids, err := db.GetRecordCIDs(
		types.WithNames(RecordName),
		types.WithVersions(RecordVersion()),
		types.WithLimit(1),
	)
	if err != nil {
		// Fall through and re-push; the store is content-addressed so an
		// unchanged record is a no-op.
		logger.Debug("skill record lookup failed, will attempt to publish anyway", "error", err)

		return "", false
	}

	if len(cids) == 0 {
		return "", false
	}

	return cids[0], true
}
