// Package commondb provides engine-agnostic database mapping for pipeline jobs.
package commondb

import (
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/google/uuid"
)

// Job represents a pipeline job row in the database.
type Job struct {
	ID                 uuid.UUID     `db:"job_id"`
	OwnerID            uuid.UUID     `db:"owner_id"`
	ProjectID          uuid.NullUUID `db:"project_id"`
	WorkflowID         string        `db:"workflow_id"`
	RunID              string        `db:"run_id"`
	InputFileID        uuid.NullUUID `db:"input_file_id"`
	IdempotencyKeyHash *string       `db:"idempotency_key_hash"`
	Status             string        `db:"status"`
	ErrorMessage       *string       `db:"error_message"`
	AudioFileID        uuid.NullUUID `db:"audio_file_id"`
	TranscriptFileID   uuid.NullUUID `db:"transcript_file_id"`
	KeyFramesFileID    uuid.NullUUID `db:"key_frames_file_id"`
	DocumentFileID     uuid.NullUUID `db:"document_file_id"`
	DateCreated        time.Time     `db:"date_created"`
	DateUpdated        time.Time     `db:"date_updated"`
}

// ToDBJob converts a business pipeline job to a database model.
func ToDBJob(bus pipelinejobbus.Job) Job {
	db := Job{
		ID:                 bus.ID,
		OwnerID:            bus.OwnerID,
		ProjectID:          nullFileID(bus.ProjectID),
		WorkflowID:         bus.WorkflowID,
		RunID:              bus.RunID,
		InputFileID:        nullFileID(bus.InputFileID),
		IdempotencyKeyHash: bus.IdempotencyKeyHash,
		Status:             bus.Status.String(),
		ErrorMessage:       bus.ErrorMessage,
		AudioFileID:        nullFileID(bus.AudioFileID),
		TranscriptFileID:   nullFileID(bus.TranscriptFileID),
		KeyFramesFileID:    nullFileID(bus.KeyFramesFileID),
		DocumentFileID:     nullFileID(bus.DocumentFileID),
		DateCreated:        bus.DateCreated.UTC(),
		DateUpdated:        bus.DateUpdated.UTC(),
	}
	return db
}

// ToBusJob converts a database pipeline job to a business model.
func ToBusJob(db Job) (pipelinejobbus.Job, error) {
	status, err := pipelinejobbus.ParseStatus(db.Status)
	if err != nil {
		return pipelinejobbus.Job{}, fmt.Errorf("parse status: %w", err)
	}

	bus := pipelinejobbus.Job{
		ID:                 db.ID,
		OwnerID:            db.OwnerID,
		ProjectID:          db.ProjectID.UUID,
		WorkflowID:         db.WorkflowID,
		RunID:              db.RunID,
		InputFileID:        db.InputFileID.UUID,
		IdempotencyKeyHash: db.IdempotencyKeyHash,
		Status:             status,
		ErrorMessage:       db.ErrorMessage,
		AudioFileID:        db.AudioFileID.UUID,
		TranscriptFileID:   db.TranscriptFileID.UUID,
		KeyFramesFileID:    db.KeyFramesFileID.UUID,
		DocumentFileID:     db.DocumentFileID.UUID,
		DateCreated:        db.DateCreated.In(time.Local),
		DateUpdated:        db.DateUpdated.In(time.Local),
	}

	return bus, nil
}

// ToBusJobs converts database pipeline jobs to business models.
func ToBusJobs(dbs []Job) ([]pipelinejobbus.Job, error) {
	bus := make([]pipelinejobbus.Job, len(dbs))
	for i, db := range dbs {
		var err error
		bus[i], err = ToBusJob(db)
		if err != nil {
			return nil, err
		}
	}
	return bus, nil
}

func nullFileID(id uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: id, Valid: id != uuid.Nil}
}
