package pipelinejobapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/query"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/google/uuid"
)

// Job represents a pipeline job returned to clients.
type Job struct {
	ID               string     `json:"id"`
	OwnerID          string     `json:"ownerId"`
	WorkflowID       string     `json:"workflowId"`
	RunID            string     `json:"runId"`
	ProjectID        string     `json:"projectId,omitempty"`
	FileID           string     `json:"fileId,omitempty"`
	FileName         string     `json:"fileName,omitempty"`
	Status           string     `json:"status"`
	FailureMessage   *string    `json:"failureMessage,omitempty"`
	Facts            []Fact     `json:"facts"`
	Progress         []Progress `json:"progress"`
	AudioFileID      string     `json:"audioFileId,omitempty"`
	TranscriptFileID string     `json:"transcriptFileId,omitempty"`
	KeyFramesFileID  string     `json:"keyFramesFileId,omitempty"`
	DocumentFileID   string     `json:"documentFileId,omitempty"`
	DocumentName     string     `json:"documentName,omitempty"`
	DateCreated      string     `json:"dateCreated"`
	DateUpdated      string     `json:"dateUpdated"`
}

// Fact is one business event in the job log.
type Fact struct {
	JobID          string  `json:"jobId"`
	Position       int64   `json:"position"`
	GlobalPosition int64   `json:"globalPosition"`
	Fact           string  `json:"fact"`
	Stage          string  `json:"stage,omitempty"`
	Outcome        string  `json:"outcome,omitempty"`
	Message        *string `json:"message,omitempty"`
	FileID         string  `json:"fileId,omitempty"`
	Time           string  `json:"time"`
}

// Progress is one live report while an activity runs.
type Progress struct {
	JobID          string   `json:"jobId"`
	Position       int64    `json:"position"`
	GlobalPosition int64    `json:"globalPosition"`
	Stage          string   `json:"stage"`
	Fraction       *float64 `json:"fraction,omitempty"`
	Headline       *string  `json:"headline,omitempty"`
	Time           string   `json:"time"`
}

// JobProjection is the folded row for one job. The screen reads these fields
// instead of folding the facts it has on hand.
type JobProjection struct {
	ID               string  `json:"id"`
	Status           string  `json:"status"`
	FailureMessage   *string `json:"failureMessage"`
	RunID            string  `json:"runId"`
	AudioFileID      string  `json:"audioFileId"`
	TranscriptFileID string  `json:"transcriptFileId"`
	KeyFramesFileID  string  `json:"keyFramesFileId"`
	DocumentFileID   string  `json:"documentFileId"`
	DateUpdated      string  `json:"dateUpdated"`
}

// FactPage is a catch-up page. Cursor is the position the client follows from next.
// Projections are the folded rows for the jobs in Facts.
type FactPage struct {
	Facts       []Fact          `json:"facts"`
	Progress    []Progress      `json:"progress"`
	Cursor      int64           `json:"cursor"`
	Projections []JobProjection `json:"projections,omitempty"`
}

// Encode implements the encoder interface.
func (p FactPage) Encode() ([]byte, string, error) {
	data, err := json.Marshal(p)
	return data, "application/json", err
}

// JobList is a page of jobs plus the project fact cursor taken before the page was read.
type JobList struct {
	query.Result[Job]
	FactsCursor int64 `json:"factsCursor"`
}

// Encode implements the encoder interface.
func (l JobList) Encode() ([]byte, string, error) {
	data, err := json.Marshal(l)
	return data, "application/json", err
}

// Encode implements the encoder interface.
func (j Job) Encode() ([]byte, string, error) {
	data, err := json.Marshal(j)
	return data, "application/json", err
}

func toAppJob(bus pipelinejobbus.Job, names map[uuid.UUID]string) Job {
	return Job{
		ID:               bus.ID.String(),
		OwnerID:          bus.OwnerID.String(),
		WorkflowID:       bus.WorkflowID,
		RunID:            bus.RunID,
		ProjectID:        fileIDString(bus.ProjectID),
		FileID:           fileIDString(bus.InputFileID),
		FileName:         names[bus.InputFileID],
		Status:           bus.Status.String(),
		FailureMessage:   bus.FailureMessage,
		Facts:            toAppFacts(bus.Facts),
		Progress:         toAppProgress(bus.Progress),
		AudioFileID:      fileIDString(bus.AudioFileID),
		TranscriptFileID: fileIDString(bus.TranscriptFileID),
		KeyFramesFileID:  fileIDString(bus.KeyFramesFileID),
		DocumentFileID:   fileIDString(bus.DocumentFileID),
		DocumentName:     names[bus.DocumentFileID],
		DateCreated:      bus.DateCreated.Format(time.RFC3339),
		DateUpdated:      bus.DateUpdated.Format(time.RFC3339),
	}
}

func toAppProjection(job pipelinejobbus.Job) JobProjection {
	return JobProjection{
		ID:               job.ID.String(),
		Status:           job.Status.String(),
		FailureMessage:   job.FailureMessage,
		RunID:            job.RunID,
		AudioFileID:      fileIDString(job.AudioFileID),
		TranscriptFileID: fileIDString(job.TranscriptFileID),
		KeyFramesFileID:  fileIDString(job.KeyFramesFileID),
		DocumentFileID:   fileIDString(job.DocumentFileID),
		DateUpdated:      job.DateUpdated.UTC().Format(time.RFC3339),
	}
}

func toAppFacts(facts []pipelinejobbus.Fact) []Fact {
	out := make([]Fact, len(facts))
	for i, fact := range facts {
		sentence := fact.Cause.Sentence()
		out[i] = Fact{
			JobID:          fact.JobID.String(),
			Position:       fact.Position,
			GlobalPosition: fact.GlobalPosition,
			Fact:           fact.Cause.Name(),
			Stage:          sentence.Stage.String(),
			Outcome:        sentence.Outcome.String(),
			Message:        fact.Cause.FailureNote(),
			FileID:         fileIDString(fact.Cause.PublishedFile()),
			Time:           fact.Time.UTC().Format(time.RFC3339),
		}
	}
	return out
}

func toAppProgress(updates []pipelinejobbus.Progress) []Progress {
	out := make([]Progress, len(updates))
	for i, update := range updates {
		out[i] = Progress{
			JobID:          update.JobID.String(),
			Position:       update.Position,
			GlobalPosition: update.GlobalPosition,
			Stage:          update.Stage.String(),
			Fraction:       update.Fraction,
			Headline:       update.Headline,
			Time:           update.Time.UTC().Format(time.RFC3339),
		}
	}
	return out
}

func fileIDString(id uuid.UUID) string {
	if id == uuid.Nil {
		return ""
	}
	return id.String()
}

func toAppJobs(jobs []pipelinejobbus.Job, names map[uuid.UUID]string) []Job {
	app := make([]Job, len(jobs))
	for i, job := range jobs {
		app[i] = toAppJob(job, names)
	}
	return app
}

// NewJob represents the data required to create a pipeline job.
type NewJob struct {
	FileID         string `json:"fileId"`
	IdempotencyKey string `json:"idempotencyKey"`
}

// Decode implements the decoder interface.
func (nj *NewJob) Decode(data []byte) error {
	return json.Unmarshal(data, nj)
}

// Validate checks the data in the model is considered clean.
func (nj NewJob) Validate() error {
	if nj.FileID == "" {
		return errs.NewFieldErrors("fileId", fmt.Errorf("fileId is required"))
	}
	return nil
}

// CreatedJob wraps a Job response with HTTP 202 status.
type CreatedJob struct {
	Job Job
}

// Encode implements the encoder interface.
func (cj CreatedJob) Encode() ([]byte, string, error) {
	return cj.Job.Encode()
}

// HTTPStatus implements the web status interface. Creating a job is accepted work.
func (CreatedJob) HTTPStatus() int {
	return http.StatusAccepted
}
