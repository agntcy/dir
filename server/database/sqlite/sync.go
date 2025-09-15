// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"time"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sync struct {
	GormID             uint `gorm:"primarykey"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ID                 string             `gorm:"not null;index"`
	RemoteDirectoryURL string             `gorm:"not null"`
	RemoteRegistryURL  string             `gorm:"not null"`
	CIDs               []string           `gorm:"serializer:json;not null"`
	Status             storev1.SyncStatus `gorm:"not null"`
}

func (sync *Sync) GetID() string {
	return sync.ID
}

func (sync *Sync) GetRemoteDirectoryURL() string {
	return sync.RemoteDirectoryURL
}

func (sync *Sync) GetRemoteRegistryURL() string {
	return sync.RemoteRegistryURL
}

func (sync *Sync) GetCIDs() []string {
	return sync.CIDs
}

func (sync *Sync) GetStatus() storev1.SyncStatus {
	return sync.Status
}

func (d *DB) CreateSync(remoteURL string, cids []string) (string, error) {
	sync := &Sync{
		ID:                 uuid.NewString(),
		RemoteDirectoryURL: remoteURL,
		CIDs:               cids,
		Status:             storev1.SyncStatus_SYNC_STATUS_PENDING,
	}

	if err := d.gormDB.Create(sync).Error; err != nil {
		return "", err
	}

	logger.Debug("Added sync to SQLite database", "sync_id", sync.ID)

	return sync.ID, nil
}

func (d *DB) GetSyncByID(syncID string) (types.SyncObject, error) {
	var sync Sync
	if err := d.gormDB.Where("id = ?", syncID).First(&sync).Error; err != nil {
		return nil, err
	}

	return &sync, nil
}

func (d *DB) GetSyncs(offset, limit int) ([]types.SyncObject, error) {
	var syncs []Sync

	query := d.gormDB.Offset(offset)

	// Only apply limit if it's greater than 0
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&syncs).Error; err != nil {
		return nil, err
	}

	// convert to types.SyncObject
	syncObjects := make([]types.SyncObject, len(syncs))
	for i, sync := range syncs {
		syncObjects[i] = &sync
	}

	return syncObjects, nil
}

func (d *DB) GetSyncsByStatus(status storev1.SyncStatus) ([]types.SyncObject, error) {
	var syncs []Sync
	if err := d.gormDB.Where("status = ?", status).Find(&syncs).Error; err != nil {
		return nil, err
	}

	// convert to types.SyncObject
	syncObjects := make([]types.SyncObject, len(syncs))
	for i, sync := range syncs {
		syncObjects[i] = &sync
	}

	return syncObjects, nil
}

func (d *DB) UpdateSyncStatus(syncID string, status storev1.SyncStatus) error {
	syncObj, err := d.GetSyncByID(syncID)
	if err != nil {
		return err
	}

	sync, ok := syncObj.(*Sync)
	if !ok {
		return gorm.ErrInvalidData
	}

	sync.Status = status

	if err := d.gormDB.Save(sync).Error; err != nil {
		return err
	}

	logger.Debug("Updated sync in SQLite database", "sync_id", sync.GetID(), "status", sync.GetStatus())

	return nil
}

func (d *DB) UpdateSyncRemoteRegistry(syncID string, remoteRegistry string) error {
	syncObj, err := d.GetSyncByID(syncID)
	if err != nil {
		return err
	}

	sync, ok := syncObj.(*Sync)
	if !ok {
		return gorm.ErrInvalidData
	}

	sync.RemoteRegistryURL = remoteRegistry

	if err := d.gormDB.Save(sync).Error; err != nil {
		return err
	}

	logger.Debug("Updated sync in SQLite database", "sync_id", sync.GetID(), "remote_registry", sync.GetRemoteRegistryURL())

	return nil
}

func (d *DB) GetSyncRemoteRegistry(syncID string) (string, error) {
	syncObj, err := d.GetSyncByID(syncID)
	if err != nil {
		return "", err
	}

	sync, ok := syncObj.(*Sync)
	if !ok {
		return "", gorm.ErrInvalidData
	}

	return sync.GetRemoteRegistryURL(), nil
}

func (d *DB) GetSyncCIDs(syncID string) ([]string, error) {
	syncObj, err := d.GetSyncByID(syncID)
	if err != nil {
		return nil, err
	}

	sync, ok := syncObj.(*Sync)
	if !ok {
		return nil, gorm.ErrInvalidData
	}

	return sync.GetCIDs(), nil
}

func (d *DB) DeleteSync(syncID string) error {
	if err := d.gormDB.Where("id = ?", syncID).Delete(&Sync{}).Error; err != nil {
		return err
	}

	logger.Debug("Deleted sync from SQLite database", "sync_id", syncID)

	return nil
}
