// Package commondb provides engine-agnostic database mapping for pipeline jobs.
package commondb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/messagedb"
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
	FailureMessage     *string       `db:"failure_message"`
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
		FailureMessage:     bus.FailureMessage,
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
		FailureMessage:     db.FailureMessage,
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

type jobAcceptedDB struct {
	Fact               string  `json:"fact"`
	OwnerID            *string `json:"ownerId,omitempty"`
	ProjectID          *string `json:"projectId,omitempty"`
	InputFileID        *string `json:"inputFileId,omitempty"`
	IdempotencyKeyHash *string `json:"idempotencyKeyHash,omitempty"`
}

type runStartedDB struct {
	Fact  string `json:"fact"`
	RunID string `json:"runId,omitempty"`
}

type publishedDB struct {
	Fact   string  `json:"fact"`
	FileID *string `json:"fileId,omitempty"`
}

type notedDB struct {
	Fact    string  `json:"fact"`
	Message *string `json:"message,omitempty"`
}

type bareDB struct {
	Fact string `json:"fact"`
}

// emptyMetadata is the message metadata. The job row owns owner and project.
var emptyMetadata = []byte("{}")

// ToDBFact converts a business fact into the message body.
// The body is that sentence. Metadata stays empty.
func ToDBFact(fact pipelinejobbus.Fact) ([]byte, []byte, error) {
	if fact.Cause == nil {
		return nil, nil, fmt.Errorf("fact is required")
	}
	body, err := sentenceBody(fact.Cause)
	if err != nil {
		return nil, nil, err
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, nil, fmt.Errorf("encode fact: %w", err)
	}
	return data, emptyMetadata, nil
}

func sentenceBody(cause pipelinejobbus.Cause) (any, error) {
	switch cause := cause.(type) {
	case pipelinejobbus.JobAccepted:
		return jobAcceptedDB{
			Fact:               cause.Name(),
			OwnerID:            uuidTextValue(cause.OwnerID),
			ProjectID:          uuidTextValue(cause.ProjectID),
			InputFileID:        uuidTextValue(cause.InputFileID),
			IdempotencyKeyHash: cause.IdempotencyKeyHash,
		}, nil
	case pipelinejobbus.RunStarted:
		return runStartedDB{Fact: cause.Name(), RunID: cause.RunID}, nil
	case pipelinejobbus.AudioPublished, pipelinejobbus.TranscriptPublished, pipelinejobbus.KeyFramesPublished, pipelinejobbus.DocumentPublished:
		return publishedDB{Fact: cause.Name(), FileID: uuidTextValue(cause.PublishedFile())}, nil
	case pipelinejobbus.RunFailed, pipelinejobbus.SourceFailed, pipelinejobbus.AudioFailed, pipelinejobbus.TranscriptFailed, pipelinejobbus.KeyFramesFailed, pipelinejobbus.DocumentFailed:
		return notedDB{Fact: cause.Name(), Message: cause.FailureNote()}, nil
	case pipelinejobbus.RunCompleted, pipelinejobbus.RunCanceled, pipelinejobbus.SourceResolved, pipelinejobbus.SourceCanceled, pipelinejobbus.AudioCanceled, pipelinejobbus.TranscriptCanceled, pipelinejobbus.KeyFramesCanceled, pipelinejobbus.KeyFramesSkipped, pipelinejobbus.DocumentCanceled:
		return bareDB{Fact: cause.Name()}, nil
	default:
		return nil, fmt.Errorf("fact %T", cause)
	}
}

// ToBusFact converts one message-store row to a business fact.
func ToBusFact(msg messagedb.Message) (pipelinejobbus.Fact, error) {
	cause, err := causeFromData(msg.Data)
	if err != nil {
		return pipelinejobbus.Fact{}, err
	}
	jobID, err := pipelinejobbus.JobIDFromStream(msg.StreamName)
	if err != nil {
		return pipelinejobbus.Fact{}, err
	}
	id, err := uuid.Parse(msg.ID)
	if err != nil {
		return pipelinejobbus.Fact{}, fmt.Errorf("parse fact id: %w", err)
	}
	return pipelinejobbus.Fact{
		ID:             id,
		JobID:          jobID,
		Position:       msg.Position,
		GlobalPosition: msg.GlobalPosition,
		Time:           msg.Time.UTC(),
		Cause:          cause,
	}, nil
}

func causeFromData(data []byte) (pipelinejobbus.Cause, error) {
	if len(data) == 0 || string(data) == "null" {
		return nil, fmt.Errorf("fact name is required")
	}
	var head bareDB
	if err := json.Unmarshal(data, &head); err != nil {
		return nil, fmt.Errorf("decode fact: %w", err)
	}
	if head.Fact == "" {
		return nil, fmt.Errorf("fact name is required")
	}
	switch head.Fact {
	case pipelinejobbus.FactJobAccepted:
		return jobAcceptedFrom(data)
	case pipelinejobbus.FactRunStarted:
		return runStartedFrom(data)
	case pipelinejobbus.FactAudioPublished, pipelinejobbus.FactTranscriptPublished, pipelinejobbus.FactKeyFramesPublished, pipelinejobbus.FactDocumentPublished:
		return publishedFrom(head.Fact, data)
	case pipelinejobbus.FactRunFailed, pipelinejobbus.FactSourceFailed, pipelinejobbus.FactAudioFailed, pipelinejobbus.FactTranscriptFailed, pipelinejobbus.FactKeyFramesFailed, pipelinejobbus.FactDocumentFailed:
		return notedFrom(head.Fact, data)
	case pipelinejobbus.FactRunCompleted:
		return pipelinejobbus.RunCompleted{}, nil
	case pipelinejobbus.FactRunCanceled:
		return pipelinejobbus.RunCanceled{}, nil
	case pipelinejobbus.FactSourceResolved:
		return pipelinejobbus.SourceResolved{}, nil
	case pipelinejobbus.FactSourceCanceled:
		return pipelinejobbus.SourceCanceled{}, nil
	case pipelinejobbus.FactAudioCanceled:
		return pipelinejobbus.AudioCanceled{}, nil
	case pipelinejobbus.FactTranscriptCanceled:
		return pipelinejobbus.TranscriptCanceled{}, nil
	case pipelinejobbus.FactKeyFramesCanceled:
		return pipelinejobbus.KeyFramesCanceled{}, nil
	case pipelinejobbus.FactKeyFramesSkipped:
		return pipelinejobbus.KeyFramesSkipped{}, nil
	case pipelinejobbus.FactDocumentCanceled:
		return pipelinejobbus.DocumentCanceled{}, nil
	default:
		return nil, fmt.Errorf("fact %s", head.Fact)
	}
}

func jobAcceptedFrom(data []byte) (pipelinejobbus.Cause, error) {
	var body jobAcceptedDB
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("decode fact: %w", err)
	}
	ownerID, err := parseOptionalUUID(stringOrEmpty(body.OwnerID))
	if err != nil {
		return nil, fmt.Errorf("parse owner id: %w", err)
	}
	projectID, err := parseOptionalUUID(stringOrEmpty(body.ProjectID))
	if err != nil {
		return nil, fmt.Errorf("parse project id: %w", err)
	}
	inputFileID, err := parseOptionalUUID(stringOrEmpty(body.InputFileID))
	if err != nil {
		return nil, fmt.Errorf("parse input file id: %w", err)
	}
	return pipelinejobbus.JobAccepted{
		OwnerID:            ownerID,
		ProjectID:          projectID,
		InputFileID:        inputFileID,
		IdempotencyKeyHash: body.IdempotencyKeyHash,
	}, nil
}

func runStartedFrom(data []byte) (pipelinejobbus.Cause, error) {
	var body runStartedDB
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("decode fact: %w", err)
	}
	return pipelinejobbus.RunStarted{RunID: body.RunID}, nil
}

func publishedFrom(name string, data []byte) (pipelinejobbus.Cause, error) {
	var body publishedDB
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("decode fact: %w", err)
	}
	if body.FileID == nil || *body.FileID == "" {
		return nil, fmt.Errorf("file id is required")
	}
	id, err := uuid.Parse(*body.FileID)
	if err != nil {
		return nil, fmt.Errorf("parse file id: %w", err)
	}
	if id == uuid.Nil {
		return nil, fmt.Errorf("file id is required")
	}
	switch name {
	case pipelinejobbus.FactAudioPublished:
		return pipelinejobbus.AudioPublished{FileID: id}, nil
	case pipelinejobbus.FactTranscriptPublished:
		return pipelinejobbus.TranscriptPublished{FileID: id}, nil
	case pipelinejobbus.FactKeyFramesPublished:
		return pipelinejobbus.KeyFramesPublished{FileID: id}, nil
	case pipelinejobbus.FactDocumentPublished:
		return pipelinejobbus.DocumentPublished{FileID: id}, nil
	default:
		return nil, fmt.Errorf("fact %s", name)
	}
}

func notedFrom(name string, data []byte) (pipelinejobbus.Cause, error) {
	var body notedDB
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("decode fact: %w", err)
	}
	kept := pipelinejobbus.CleanNote(body.Message)
	switch name {
	case pipelinejobbus.FactRunFailed:
		return pipelinejobbus.RunFailed{Note: kept}, nil
	case pipelinejobbus.FactSourceFailed:
		return pipelinejobbus.SourceFailed{Note: kept}, nil
	case pipelinejobbus.FactAudioFailed:
		return pipelinejobbus.AudioFailed{Note: kept}, nil
	case pipelinejobbus.FactTranscriptFailed:
		return pipelinejobbus.TranscriptFailed{Note: kept}, nil
	case pipelinejobbus.FactKeyFramesFailed:
		return pipelinejobbus.KeyFramesFailed{Note: kept}, nil
	case pipelinejobbus.FactDocumentFailed:
		return pipelinejobbus.DocumentFailed{Note: kept}, nil
	default:
		return nil, fmt.Errorf("fact %s", name)
	}
}

// ToBusFacts converts message-store rows to business events.
func ToBusFacts(msgs []messagedb.Message) ([]pipelinejobbus.Fact, error) {
	bus := make([]pipelinejobbus.Fact, 0, len(msgs))
	for _, msg := range msgs {
		fact, err := ToBusFact(msg)
		if err != nil {
			return nil, err
		}
		bus = append(bus, fact)
	}
	return bus, nil
}

type progressDB struct {
	Stage    string   `json:"stage"`
	Fraction *float64 `json:"fraction,omitempty"`
	Headline *string  `json:"headline,omitempty"`
}

// ToDBProgress converts a progress update into the message body.
// Metadata stays empty. The project list joins the job row.
func ToDBProgress(progress pipelinejobbus.Progress) ([]byte, []byte, error) {
	data, err := json.Marshal(progressDB{
		Stage:    progress.Stage.String(),
		Fraction: progress.Fraction,
		Headline: progress.Headline,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("encode progress: %w", err)
	}
	return data, emptyMetadata, nil
}

// ToBusProgress converts one progress message.
func ToBusProgress(msg messagedb.Message) (pipelinejobbus.Progress, error) {
	var body progressDB
	if len(msg.Data) > 0 && string(msg.Data) != "null" {
		if err := json.Unmarshal(msg.Data, &body); err != nil {
			return pipelinejobbus.Progress{}, fmt.Errorf("decode progress: %w", err)
		}
	}
	stage, err := pipelinejobbus.ParseStage(body.Stage)
	if err != nil {
		return pipelinejobbus.Progress{}, err
	}
	jobID, err := pipelinejobbus.JobIDFromStream(msg.StreamName)
	if err != nil {
		return pipelinejobbus.Progress{}, err
	}
	id, err := uuid.Parse(msg.ID)
	if err != nil {
		return pipelinejobbus.Progress{}, fmt.Errorf("parse progress id: %w", err)
	}
	return pipelinejobbus.Progress{
		ID:             id,
		JobID:          jobID,
		Position:       msg.Position,
		GlobalPosition: msg.GlobalPosition,
		Time:           msg.Time.UTC(),
		Stage:          stage,
		Fraction:       body.Fraction,
		Headline:       body.Headline,
	}, nil
}

// ToBusLog splits a stream page into business events and progress updates.
func ToBusLog(msgs []messagedb.Message) (pipelinejobbus.Log, error) {
	var log pipelinejobbus.Log
	for _, msg := range msgs {
		switch msg.Type {
		case pipelinejobbus.ProgressType:
			update, err := ToBusProgress(msg)
			if err != nil {
				return pipelinejobbus.Log{}, err
			}
			log.Progress = append(log.Progress, update)
		case pipelinejobbus.FactType:
			fact, err := ToBusFact(msg)
			if err != nil {
				return pipelinejobbus.Log{}, err
			}
			log.Facts = append(log.Facts, fact)
		default:
			return pipelinejobbus.Log{}, fmt.Errorf("message type %q", msg.Type)
		}
	}
	return log, nil
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseOptionalUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, nil
	}
	return uuid.Parse(value)
}

func uuidTextValue(id uuid.UUID) *string {
	if id == uuid.Nil {
		return nil
	}
	s := id.String()
	return &s
}
