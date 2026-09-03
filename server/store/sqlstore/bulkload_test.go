// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

// TestBulkLoad fills the store with synthetic records derived from a real one
// and then times the reads that matter, so the numbers can be compared with the
// registry measurements taken against Zot, Distribution, Harbor and Quay.
//
// It is skipped unless SQLSTORE_BULKLOAD is set, because it needs a Postgres
// instance and takes minutes:
//
//	SQLSTORE_BULKLOAD=1 SQLSTORE_PG_PASSWORD=... SQLSTORE_RECORDS=1000000 \
//	SQLSTORE_TEMPLATE=/path/to/record.json go test -run TestBulkLoad -timeout 4h ./...
func TestBulkLoad(t *testing.T) {
	if os.Getenv("SQLSTORE_BULKLOAD") == "" {
		t.Skip("set SQLSTORE_BULKLOAD=1 to run the scale test")
	}

	var (
		total         = envInt(t, "SQLSTORE_RECORDS", 100_000)
		perRecord     = envInt(t, "SQLSTORE_REFERRERS", 8)
		batchSize     = envInt(t, "SQLSTORE_BATCH", 500)
		reportEvery   = envInt(t, "SQLSTORE_REPORT", 50_000)
		templatePath  = os.Getenv("SQLSTORE_TEMPLATE")
		ctx           = context.Background()
		insertedCIDs  []string
		referrerTypes = []string{
			"agntcy.dir.sign.v1.Signature",
			"agntcy.dir.sign.v1.PublicKey",
			"agntcy.dir.scan.v1.Result",
		}
	)

	require.NotEmpty(t, templatePath, "SQLSTORE_TEMPLATE must point at a real record JSON")

	template, err := os.ReadFile(filepath.Clean(templatePath))
	require.NoError(t, err)

	api, err := New(dbconfig.Config{
		Type: string(database.Postgres),
		Postgres: dbconfig.PostgresConfig{
			Host:     envStr("SQLSTORE_PG_HOST", "localhost"),
			Port:     envInt(t, "SQLSTORE_PG_PORT", 5433),
			Database: envStr("SQLSTORE_PG_DATABASE", "dirstore"),
			Username: envStr("SQLSTORE_PG_USER", "postgres"),
			Password: os.Getenv("SQLSTORE_PG_PASSWORD"),
			SSLMode:  "disable",
		},
	})
	require.NoError(t, err)

	s, ok := api.(*store)
	require.True(t, ok)

	// Batch inserts of this size trip the slow-query threshold, and the default
	// logger then prints every statement in full.
	s.db.Logger = gormlogger.Discard

	t.Logf("loading %d records with %d referrers each (batch %d)", total, perRecord, batchSize)

	loadStart := time.Now()

	for offset := 0; offset < total; offset += batchSize {
		size := min(batchSize, total-offset)
		records := make([]Record, 0, size)
		referrers := make([]Referrer, 0, size*perRecord)

		for i := offset; i < offset+size; i++ {
			row, cid := synthRecord(t, template, i)
			records = append(records, row)

			// Keep a sample of CIDs spread across the whole range so the read
			// measurements are not all served from the same index pages.
			if i%(max(total/1000, 1)) == 0 {
				insertedCIDs = append(insertedCIDs, cid)
			}

			for j := range perRecord {
				referrers = append(referrers, synthReferrer(t, cid, referrerTypes[j%len(referrerTypes)], j))
			}
		}

		// DoNothing on conflict makes the loader resumable: re-running it tops
		// the table up instead of failing on rows already present.
		skip := clause.OnConflict{DoNothing: true}
		require.NoError(t, s.db.WithContext(ctx).Clauses(skip).CreateInBatches(records, batchSize).Error)
		require.NoError(t, s.db.WithContext(ctx).Clauses(skip).CreateInBatches(referrers, batchSize).Error)

		if done := offset + size; done%reportEvery == 0 {
			elapsed := time.Since(loadStart)
			t.Logf("  %d records in %s (%.0f records/s)", done, elapsed.Truncate(time.Second),
				float64(done)/elapsed.Seconds())
		}
	}

	t.Logf("load complete in %s", time.Since(loadStart).Truncate(time.Second))

	var recordCount, referrerCount int64
	require.NoError(t, s.db.Model(&Record{}).Count(&recordCount).Error)
	require.NoError(t, s.db.Model(&Referrer{}).Count(&referrerCount).Error)
	t.Logf("rows: %d records, %d referrers", recordCount, referrerCount)

	for _, q := range []struct{ label, sql string }{
		{"records table", "select pg_size_pretty(pg_total_relation_size('store_records'))"},
		{"referrers table", "select pg_size_pretty(pg_total_relation_size('store_referrers'))"},
		{"database", "select pg_size_pretty(pg_database_size(current_database()))"},
	} {
		var size string
		require.NoError(t, s.db.Raw(q.sql).Scan(&size).Error)
		t.Logf("size %-16s %s", q.label, size)
	}

	// Reads, each sampled over many random CIDs so one lucky cache hit cannot
	// carry the result.
	rng := rand.New(rand.NewSource(1)) //nolint:gosec
	sample := func() string { return insertedCIDs[rng.Intn(len(insertedCIDs))] }

	measure(t, "Lookup", 200, func() error {
		_, err := s.Lookup(ctx, &corev1.RecordRef{Cid: sample()})

		return err
	})

	measure(t, "Pull (re-derives and verifies CID)", 200, func() error {
		_, err := s.Pull(ctx, &corev1.RecordRef{Cid: sample()})

		return err
	})

	measure(t, "WalkReferrers all types", 200, func() error {
		n := 0

		return s.WalkReferrers(ctx, sample(), "", func(*corev1.RecordReferrer) error { n++; return nil })
	})

	measure(t, "WalkReferrers filtered by type", 200, func() error {
		n := 0

		return s.WalkReferrers(ctx, sample(), referrerTypes[0], func(*corev1.RecordReferrer) error { n++; return nil })
	})

	measure(t, "Push one more record", 200, func() error {
		row, _ := synthRecord(t, template, total+rng.Intn(1_000_000))

		return s.db.WithContext(ctx).Create(&row).Error
	})
}

// synthRecord derives a unique record from a real one by rewriting its name and
// version, so every row has genuine canonical bytes and a genuine CID.
func synthRecord(t *testing.T, template []byte, i int) (Record, string) {
	t.Helper()

	var doc map[string]any
	require.NoError(t, json.Unmarshal(template, &doc))

	doc["name"] = fmt.Sprintf("bulk.example.org/agent-%08d", i)
	doc["version"] = fmt.Sprintf("1.0.%d", i%1000)

	mutated, err := json.Marshal(doc)
	require.NoError(t, err)

	record, err := corev1.UnmarshalRecord(mutated)
	require.NoError(t, err)

	canonical, err := record.Marshal()
	require.NoError(t, err)

	cid, err := cidFromBytes(canonical)
	require.NoError(t, err)

	row := Record{RecordCID: cid, CanonicalBytes: canonical}
	populateMetadata(&row, record)

	return row, cid
}

func synthReferrer(t *testing.T, subjectCID, refType string, j int) Referrer {
	t.Helper()

	referrer := &corev1.RecordReferrer{
		Type:      refType,
		RecordRef: &corev1.RecordRef{Cid: subjectCID},
		Annotations: map[string]string{
			"index":   fmt.Sprintf("%d", j),
			"payload": fmt.Sprintf("synthetic-%s-%s", refType, subjectCID),
		},
	}

	payload, err := referrer.Marshal()
	require.NoError(t, err)

	cid, err := cidFromBytes(payload)
	require.NoError(t, err)

	return Referrer{ReferrerCID: cid, SubjectCID: subjectCID, Type: refType, Payload: payload}
}

// measure runs op n times and reports the distribution, since a mean alone
// hides the tail that matters for a request path.
func measure(t *testing.T, label string, n int, op func() error) {
	t.Helper()

	samples := make([]time.Duration, 0, n)

	for range n {
		start := time.Now()
		require.NoError(t, op())
		samples = append(samples, time.Since(start))
	}

	slices.Sort(samples)

	ms := func(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
	t.Logf("%-38s n=%d  p50 %.3f ms  p95 %.3f ms  max %.3f ms",
		label, n, ms(samples[n/2]), ms(samples[n*95/100]), ms(samples[n-1]))
}

func envInt(t *testing.T, key string, fallback int) int {
	t.Helper()

	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	var parsed int
	_, err := fmt.Sscanf(raw, "%d", &parsed)
	require.NoError(t, err)

	return parsed
}

func envStr(key, fallback string) string {
	if raw := os.Getenv(key); raw != "" {
		return raw
	}

	return fallback
}
