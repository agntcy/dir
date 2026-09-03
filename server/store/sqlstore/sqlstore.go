// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package sqlstore implements the content store over a SQL database, holding
// records as canonical bytes and referrers as rows keyed by subject CID.
package sqlstore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/glebarez/sqlite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	gormlogger "gorm.io/gorm/logger"
)

var logger = logging.Logger("store/sqlstore")

// Metadata keys are mirrored from the OCI store so that RecordMeta is identical
// whichever backend served it. They are duplicated rather than imported to keep
// this package independent of the OCI implementation.
const (
	metadataKeyName        = "name"
	metadataKeyVersion     = "version"
	metadataKeyOASFVersion = "oasf-version"
	metadataKeyPreviousCID = "previous-cid"

	fallbackSchemaVersion = "0.7.0"
)

type store struct {
	db *gorm.DB
}

var (
	_ types.StoreAPI         = (*store)(nil)
	_ types.ReferrerStoreAPI = (*store)(nil)
	_ types.FullStore        = (*store)(nil)
)

func New(cfg dbconfig.Config) (types.StoreAPI, error) {
	var (
		db  *gorm.DB
		err error
	)

	switch dbType := cfg.Type; dbType {
	case "", string(database.Postgres):
		logger.Info("Initializing SQL store on PostgreSQL",
			"host", cfg.Postgres.Host, "port", cfg.Postgres.Port, "database", cfg.Postgres.Database)

		db, err = database.NewPostgresGormDb(cfg.Postgres)
	case string(database.SQLite):
		logger.Info("Initializing SQL store on SQLite", "path", cfg.SQLite.Path)

		db, err = openSQLite(cfg.SQLite)
	default:
		return nil, fmt.Errorf("unsupported sql store type=%s", dbType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect sql store: %w", err)
	}

	if err := db.AutoMigrate(&Record{}, &Referrer{}); err != nil {
		return nil, fmt.Errorf("failed to migrate sql store schema: %w", err)
	}

	return &store{db: db}, nil
}

func openSQLite(cfg dbconfig.SQLiteConfig) (*gorm.DB, error) {
	path := cfg.Path
	if path == "" {
		path = dbconfig.DefaultSQLitePath
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{Logger: newLogger()})
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite store at %s: %w", path, err)
	}

	// Required for the referrer foreign key to cascade on record deletion.
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("failed to enable SQLite foreign keys: %w", err)
	}

	return db, nil
}

// newLogger keeps "record not found" out of the log: Lookup is used to test for
// existence, so a miss is an expected outcome rather than a fault.
func newLogger() gormlogger.Interface {
	return gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             200 * time.Millisecond, //nolint:mnd
			LogLevel:                  gormlogger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)
}

func (s *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	if record == nil {
		return nil, status.Error(codes.InvalidArgument, "record is required") //nolint:wrapcheck
	}

	canonicalBytes, err := record.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	recordCID, err := cidFromBytes(canonicalBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive CID: %v", err)
	}

	if expected := record.GetCid(); recordCID != expected {
		return nil, status.Errorf(codes.Internal,
			"CID mismatch: derived CID (%s) != record CID (%s)", recordCID, expected)
	}

	row := &Record{RecordCID: recordCID, CanonicalBytes: canonicalBytes}
	populateMetadata(row, record)

	// Content addressing means an existing row under this CID already holds
	// these exact bytes, so a conflict is a satisfied push rather than an error.
	res := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(row)
	if res.Error != nil {
		return nil, status.Errorf(codes.Internal, "failed to store record %s: %v", recordCID, res.Error)
	}

	logger.Debug("Record pushed to SQL store",
		"cid", recordCID, "bytes", len(canonicalBytes), "inserted", res.RowsAffected == 1)

	return &corev1.RecordRef{Cid: recordCID}, nil
}

func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}

	var row Record

	err := s.db.WithContext(ctx).
		Select("record_cid", "canonical_bytes").
		First(&row, "record_cid = ?", ref.GetCid()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "record not found for CID %s", ref.GetCid())
		}

		return nil, status.Errorf(codes.Internal, "failed to read record %s: %v", ref.GetCid(), err)
	}

	// Re-derive the CID from the bytes before returning them. Content
	// addressing is only a guarantee if the binding is verified on read.
	actualCID, err := cidFromBytes(row.CanonicalBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive CID for %s: %v", ref.GetCid(), err)
	}

	if actualCID != ref.GetCid() {
		return nil, status.Errorf(codes.DataLoss,
			"content integrity failure: stored bytes hash to %s, not %s", actualCID, ref.GetCid())
	}

	record, err := corev1.UnmarshalRecord(row.CanonicalBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record %s: %v", ref.GetCid(), err)
	}

	return record, nil
}

func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	if err := validateRef(ref); err != nil {
		return nil, err
	}

	var row Record

	err := s.db.WithContext(ctx).
		Select("record_cid", "name", "version", "schema_version", "oasf_version", "oasf_created_at", "previous_cid", "annotations").
		First(&row, "record_cid = ?", ref.GetCid()).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Errorf(codes.NotFound, "record not found for CID %s", ref.GetCid())
		}

		return nil, status.Errorf(codes.Internal, "failed to look up record %s: %v", ref.GetCid(), err)
	}

	return row.toRecordMeta(), nil
}

func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	if err := validateRef(ref); err != nil {
		return err
	}

	// Referrers are removed by the foreign key on subject_cid.
	res := s.db.WithContext(ctx).Delete(&Record{}, "record_cid = ?", ref.GetCid())
	if res.Error != nil {
		return status.Errorf(codes.Internal, "failed to delete record %s: %v", ref.GetCid(), res.Error)
	}

	logger.Debug("Record deleted from SQL store", "cid", ref.GetCid(), "existed", res.RowsAffected == 1)

	return nil
}

func (s *store) IsReady(ctx context.Context) bool {
	sqlDB, err := s.db.DB()
	if err != nil {
		logger.Debug("Store not ready: cannot access connection", "error", err)

		return false
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		logger.Debug("Store not ready: ping failed", "error", err)

		return false
	}

	return true
}

func cidFromBytes(data []byte) (string, error) {
	digest, err := corev1.CalculateDigest(data)
	if err != nil {
		return "", fmt.Errorf("failed to calculate digest: %w", err)
	}

	cid, err := corev1.ConvertDigestToCID(digest)
	if err != nil {
		return "", fmt.Errorf("failed to convert digest to CID: %w", err)
	}

	return cid, nil
}

func validateRef(ref *corev1.RecordRef) error {
	if ref == nil || ref.GetCid() == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	return nil
}

func populateMetadata(row *Record, record *corev1.Record) {
	data, err := record.Decode()
	if err != nil {
		return
	}

	row.Name = data.GetName()
	row.Version = data.GetVersion()
	row.SchemaVersion = data.GetSchemaVersion()
	row.OASFVersion = data.GetSchemaVersion()
	row.OASFCreatedAt = data.GetCreatedAt()
	row.PreviousCID = data.GetPreviousRecordCid()
	row.Annotations = data.GetAnnotations()
}

func (r *Record) toRecordMeta() *corev1.RecordMeta {
	meta := &corev1.RecordMeta{
		Cid:           r.RecordCID,
		SchemaVersion: fallbackSchemaVersion,
		CreatedAt:     r.OASFCreatedAt,
		Annotations:   make(map[string]string, len(r.Annotations)+4),
	}

	if r.SchemaVersion != "" {
		meta.SchemaVersion = r.SchemaVersion
	}

	for key, value := range map[string]string{
		metadataKeyName:        r.Name,
		metadataKeyVersion:     r.Version,
		metadataKeyOASFVersion: r.OASFVersion,
		metadataKeyPreviousCID: r.PreviousCID,
	} {
		if value != "" {
			meta.Annotations[key] = value
		}
	}

	for key, value := range r.Annotations {
		meta.Annotations[key] = value
	}

	return meta
}
