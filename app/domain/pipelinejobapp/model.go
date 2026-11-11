package pipelinejobapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/errs"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

// Job represents a pipeline job returned to clients.
type Job struct {
	ID            string  `json:"id"`
	OwnerID       string  `json:"ownerId"`
	WorkflowID    string  `json:"workflowId"`
	RunID         string  `json:"runId"`
	FileID        string  `json:"fileId,omitempty"`
	SourceRef     string  `json:"sourceRef"`
	Status        string  `json:"status"`
	ErrorMessage  *string `json:"errorMessage,omitempty"`
	AudioRef      *string `json:"audioRef,omitempty"`
	TranscriptRef *string `json:"transcriptRef,omitempty"`
	KeyFramesRef  *string `json:"keyFramesRef,omitempty"`
	DocumentRef   *string `json:"documentRef,omitempty"`
	DateCreated   string  `json:"dateCreated"`
	DateUpdated   string  `json:"dateUpdated"`
}

// Encode implements the encoder interface.
func (j Job) Encode() ([]byte, string, error) {
	data, err := json.Marshal(j)
	return data, "application/json", err
}

func toAppJob(bus pipelinejobbus.Job, fileID string) Job {
	return Job{
		ID:            bus.ID.String(),
		OwnerID:       bus.OwnerID.String(),
		WorkflowID:    bus.WorkflowID,
		RunID:         bus.RunID,
		FileID:        fileID,
		SourceRef:     bus.SourceRef,
		Status:        bus.Status.String(),
		ErrorMessage:  bus.ErrorMessage,
		AudioRef:      bus.AudioRef,
		TranscriptRef: bus.TranscriptRef,
		KeyFramesRef:  bus.KeyFramesRef,
		DocumentRef:   bus.DocumentRef,
		DateCreated:   bus.DateCreated.Format(time.RFC3339),
		DateUpdated:   bus.DateUpdated.Format(time.RFC3339),
	}
}

func toAppJobs(jobs []pipelinejobbus.Job, fileIDs map[string]string) []Job {
	app := make([]Job, len(jobs))
	for i, job := range jobs {
		app[i] = toAppJob(job, fileIDs[job.SourceRef])
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

// WriteResponse writes the created job with HTTP 202 status.
func WriteResponse(w http.ResponseWriter, job Job) error {
	data, contentType, err := job.Encode()
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusAccepted)
	_, err = w.Write(data)
	return err
}
