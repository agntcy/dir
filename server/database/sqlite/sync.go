// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sync struct {
	gorm.Model
	ID                 string                   `gorm:"not null;index"`
	RemoteDirectoryURL string                   `gorm:"not null"`
	Status             storev1alpha2.SyncStatus `gorm:"not null"`
}

func (sync *Sync) GetID() string {
	return sync.ID
}

func (sync *Sync) GetRemoteDirectoryURL() string {
	return sync.RemoteDirectoryURL
}

func (sync *Sync) GetStatus() storev1alpha2.SyncStatus {
	return sync.Status
}

func (d *DB) CreateSync(remoteURL string) (string, error) {
	sync := &Sync{
		ID:                 uuid.NewString(),
		RemoteDirectoryURL: remoteURL,
		Status:             storev1alpha2.SyncStatus_SYNC_STATUS_PENDING,
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

func (d *DB) GetSyncs() ([]types.SyncObject, error) {
	var syncs []Sync
	if err := d.gormDB.Find(&syncs).Error; err != nil {
		return nil, err
	}

	// convert to types.SyncObject
	syncObjects := make([]types.SyncObject, len(syncs))
	for i, sync := range syncs {
		syncObjects[i] = &sync
	}

	return syncObjects, nil
}

func (d *DB) GetSyncsByStatus(status storev1alpha2.SyncStatus) ([]types.SyncObject, error) {
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

func (d *DB) UpdateSyncStatus(syncID string, status storev1alpha2.SyncStatus) error {
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

func (d *DB) DeleteSync(syncID string) error {
	if err := d.gormDB.Where("id = ?", syncID).Delete(&Sync{}).Error; err != nil {
		return err
	}

	logger.Debug("Deleted sync from SQLite database", "sync_id", syncID)

	return nil
}
