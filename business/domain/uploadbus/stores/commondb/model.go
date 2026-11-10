// Package commondb provides engine-agnostic database mapping for uploads.
package commondb

import (
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/uploadbus"
	"github.com/google/uuid"
)

// Upload represents an upload row in the database.
type Upload struct {
	ObjectKey    string    `db:"object_key"`
	OwnerID      uuid.UUID `db:"owner_id"`
	Bucket       string    `db:"bucket"`
	ObjectKeyRef string    `db:"object_key_ref"`
	SourceRef    string    `db:"source_ref"`
	SizeBytes    int64     `db:"size_bytes"`
	ETag         *string   `db:"etag"`
	Status       string    `db:"status"`
	TusdUploadID *string   `db:"tusd_upload_id"`
	DateCreated  time.Time `db:"date_created"`
	DateUpdated  time.Time `db:"date_updated"`
}

// ToDBUpload converts a business upload to a database model.
func ToDBUpload(bus uploadbus.Upload) Upload {
	return Upload{
		ObjectKey:    bus.ObjectKey,
		OwnerID:      bus.OwnerID,
		Bucket:       bus.Bucket,
		ObjectKeyRef: bus.ObjectKeyRef,
		SourceRef:    bus.SourceRef,
		SizeBytes:    bus.SizeBytes,
		ETag:         bus.ETag,
		Status:       bus.Status.String(),
		TusdUploadID: bus.TusdUploadID,
		DateCreated:  bus.DateCreated.UTC(),
		DateUpdated:  bus.DateUpdated.UTC(),
	}
}

// ToBusUpload converts a database upload to a business model.
func ToBusUpload(db Upload) (uploadbus.Upload, error) {
	status, err := uploadbus.ParseStatus(db.Status)
	if err != nil {
		return uploadbus.Upload{}, fmt.Errorf("parse status: %w", err)
	}

	return uploadbus.Upload{
		ObjectKey:    db.ObjectKey,
		OwnerID:      db.OwnerID,
		Bucket:       db.Bucket,
		ObjectKeyRef: db.ObjectKeyRef,
		SourceRef:    db.SourceRef,
		SizeBytes:    db.SizeBytes,
		ETag:         db.ETag,
		Status:       status,
		TusdUploadID: db.TusdUploadID,
		DateCreated:  db.DateCreated.In(time.Local),
		DateUpdated:  db.DateUpdated.In(time.Local),
	}, nil
}

// ToBusUploads converts database uploads to business models.
func ToBusUploads(dbs []Upload) ([]uploadbus.Upload, error) {
	bus := make([]uploadbus.Upload, len(dbs))
	for i, db := range dbs {
		var err error
		bus[i], err = ToBusUpload(db)
		if err != nil {
			return nil, err
		}
	}
	return bus, nil
}
