// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sqlite

import (
	"fmt"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

type Publication struct {
	GormID         uint `gorm:"primarykey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	ID             string                      `gorm:"not null;index"`
	RequestJSON    string                      `gorm:"not null"` // JSON-encoded PublishRequest
	Status         routingv1.PublicationStatus `gorm:"not null"`
	CreatedTime    string                      `gorm:"not null"`
	LastUpdateTime string                      `gorm:"not null"`
}

func (pub *Publication) GetID() string {
	return pub.ID
}

func (pub *Publication) GetRequest() *routingv1.PublishRequest {
	var request routingv1.PublishRequest
	if err := protojson.Unmarshal([]byte(pub.RequestJSON), &request); err != nil {
		logger.Error("Failed to unmarshal publish request", "error", err)

		return nil
	}

	return &request
}

func (pub *Publication) GetStatus() routingv1.PublicationStatus {
	return pub.Status
}

func (pub *Publication) GetCreatedTime() string {
	return pub.CreatedTime
}

func (pub *Publication) GetLastUpdateTime() string {
	return pub.LastUpdateTime
}

func (d *DB) CreatePublication(request *routingv1.PublishRequest) (string, error) {
	requestJSON, err := protojson.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal publish request: %w", err)
	}

	now := time.Now().Format(time.RFC3339)
	publication := &Publication{
		ID:             uuid.NewString(),
		RequestJSON:    string(requestJSON),
		Status:         routingv1.PublicationStatus_PUBLICATION_STATUS_PENDING,
		CreatedTime:    now,
		LastUpdateTime: now,
	}

	if err := d.gormDB.Create(publication).Error; err != nil {
		return "", fmt.Errorf("failed to create publication: %w", err)
	}

	logger.Debug("Added publication to SQLite database", "publication_id", publication.ID)

	return publication.ID, nil
}

func (d *DB) GetPublicationByID(publicationID string) (types.PublicationObject, error) {
	var publication Publication
	if err := d.gormDB.Where("id = ?", publicationID).First(&publication).Error; err != nil {
		return nil, err
	}

	return &publication, nil
}

func (d *DB) GetPublications(offset, limit int) ([]types.PublicationObject, error) {
	var publications []Publication

	query := d.gormDB.Offset(offset)

	// Only apply limit if it's greater than 0
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&publications).Error; err != nil {
		return nil, err
	}

	// convert to types.PublicationObject
	publicationObjects := make([]types.PublicationObject, len(publications))
	for i, publication := range publications {
		publicationObjects[i] = &publication
	}

	return publicationObjects, nil
}

func (d *DB) GetPublicationsByStatus(status routingv1.PublicationStatus) ([]types.PublicationObject, error) {
	var publications []Publication
	if err := d.gormDB.Where("status = ?", status).Find(&publications).Error; err != nil {
		return nil, err
	}

	// convert to types.PublicationObject
	publicationObjects := make([]types.PublicationObject, len(publications))
	for i, publication := range publications {
		publicationObjects[i] = &publication
	}

	return publicationObjects, nil
}

func (d *DB) UpdatePublicationStatus(publicationID string, status routingv1.PublicationStatus) error {
	publicationObj, err := d.GetPublicationByID(publicationID)
	if err != nil {
		return err
	}

	publication, ok := publicationObj.(*Publication)
	if !ok {
		return gorm.ErrInvalidData
	}

	publication.Status = status
	publication.LastUpdateTime = time.Now().Format(time.RFC3339)

	if err := d.gormDB.Save(publication).Error; err != nil {
		return err
	}

	logger.Debug("Updated publication in SQLite database", "publication_id", publication.GetID(), "status", publication.GetStatus())

	return nil
}

func (d *DB) DeletePublication(publicationID string) error {
	if err := d.gormDB.Where("id = ?", publicationID).Delete(&Publication{}).Error; err != nil {
		return err
	}

	logger.Debug("Deleted publication from SQLite database", "publication_id", publicationID)

	return nil
}
