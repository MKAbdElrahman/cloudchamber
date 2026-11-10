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
	ID                 uuid.UUID `db:"job_id"`
	OwnerID            uuid.UUID `db:"owner_id"`
	WorkflowID         string    `db:"workflow_id"`
	RunID              string    `db:"run_id"`
	SourceRef          string    `db:"source_ref"`
	IdempotencyKeyHash *string   `db:"idempotency_key_hash"`
	Status             string    `db:"status"`
	ErrorMessage       *string   `db:"error_message"`
	AudioRef           *string   `db:"audio_ref"`
	TranscriptRef      *string   `db:"transcript_ref"`
	KeyFramesRef       *string   `db:"key_frames_ref"`
	DocumentRef        *string   `db:"document_ref"`
	DateCreated        time.Time `db:"date_created"`
	DateUpdated        time.Time `db:"date_updated"`
}

// ToDBJob converts a business pipeline job to a database model.
func ToDBJob(bus pipelinejobbus.Job) Job {
	db := Job{
		ID:                 bus.ID,
		OwnerID:            bus.OwnerID,
		WorkflowID:         bus.WorkflowID,
		RunID:              bus.RunID,
		SourceRef:          bus.SourceRef,
		IdempotencyKeyHash: bus.IdempotencyKeyHash,
		Status:             bus.Status.String(),
		ErrorMessage:       bus.ErrorMessage,
		AudioRef:           bus.AudioRef,
		TranscriptRef:      bus.TranscriptRef,
		KeyFramesRef:       bus.KeyFramesRef,
		DocumentRef:        bus.DocumentRef,
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
		WorkflowID:         db.WorkflowID,
		RunID:              db.RunID,
		SourceRef:          db.SourceRef,
		IdempotencyKeyHash: db.IdempotencyKeyHash,
		Status:             status,
		ErrorMessage:       db.ErrorMessage,
		AudioRef:           db.AudioRef,
		TranscriptRef:      db.TranscriptRef,
		KeyFramesRef:       db.KeyFramesRef,
		DocumentRef:        db.DocumentRef,
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
