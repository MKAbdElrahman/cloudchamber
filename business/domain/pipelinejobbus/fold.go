package pipelinejobbus

import (
	"slices"

	"github.com/google/uuid"
)

// submitFactNamespace and startFactNamespace are the UUID v5 namespaces for the
// accepted and started facts. They stay fixed so a retry writes the same message id.
var (
	submitFactNamespace = uuid.MustParse("c1a5e8b0-7d34-4a19-8f62-0e5b9c3d71a4")
	startFactNamespace  = uuid.MustParse("d2b6f9c1-8e45-4b2a-9073-1f6c0d4e82b5")
)

// SubmitFactID is the message id of the accepted fact for a job.
func SubmitFactID(jobID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(submitFactNamespace, jobID[:])
}

// StartFactID is the message id of the started fact for a job.
func StartFactID(jobID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(startFactNamespace, jobID[:])
}

// WorkflowID is the Temporal workflow id derived from the job id.
func WorkflowID(jobID uuid.UUID) string {
	return "cloudchamber-job-" + jobID.String()
}

// Fold projects a job from its facts alone. It does not read a previous row.
// The first accepted fact, the first start, the first ending, and the first
// published file of each kind are the causes. A later copy does not replace them.
// The last fact owns the updated time. A log with no accepted fact is not a job.
func Fold(jobID uuid.UUID, facts []Fact) Job {
	job := Job{
		ID:         jobID,
		WorkflowID: WorkflowID(jobID),
		Status:     MustParseStatus(StatusSubmitted),
	}

	var files publishedFiles
	var hasAccepted bool
	var hasStarted bool
	var ending Cause
	for _, fact := range facts {
		job.DateUpdated = fact.Time.UTC()
		switch cause := fact.Cause.(type) {
		case JobAccepted:
			if hasAccepted {
				continue
			}
			hasAccepted = true
			job.OwnerID = cause.OwnerID
			job.ProjectID = cause.ProjectID
			job.InputFileID = cause.InputFileID
			job.IdempotencyKeyHash = cause.IdempotencyKeyHash
			job.DateCreated = fact.Time.UTC()
		case RunStarted:
			if hasStarted {
				continue
			}
			hasStarted = true
			job.RunID = cause.RunID
		case RunCompleted, RunFailed, RunCanceled:
			if ending != nil {
				continue
			}
			ending = cause
		case AudioPublished:
			files.audio = keepFile(files.audio, cause.FileID)
		case TranscriptPublished:
			files.transcript = keepFile(files.transcript, cause.FileID)
		case KeyFramesPublished:
			files.keyFrames = keepFile(files.keyFrames, cause.FileID)
		case DocumentPublished:
			files.document = keepFile(files.document, cause.FileID)
		}
	}

	switch cause := ending.(type) {
	case RunCompleted:
		job.Status = MustParseStatus(StatusCompleted)
	case RunFailed:
		job.Status = MustParseStatus(StatusFailed)
		job.FailureMessage = new(failureText(cause.Note))
	case RunCanceled:
		job.Status = MustParseStatus(StatusCanceled)
	default:
		if hasStarted {
			job.Status = MustParseStatus(StatusRunning)
		}
	}

	job.AudioFileID = files.audio
	job.TranscriptFileID = files.transcript
	job.KeyFramesFileID = files.keyFrames
	job.DocumentFileID = files.document
	return job
}

func failureText(message *string) string {
	if kept := note(message); kept != nil {
		return *kept
	}
	return StatusFailed
}

type publishedFiles struct {
	audio      uuid.UUID
	transcript uuid.UUID
	keyFrames  uuid.UUID
	document   uuid.UUID
}

func keepFile(current, id uuid.UUID) uuid.UUID {
	if id == uuid.Nil || current != uuid.Nil {
		return current
	}
	return id
}

func normalizeEnding(cause Cause) Cause {
	failed, ok := cause.(RunFailed)
	if !ok {
		return cause
	}
	return RunFailed{Note: note(failed.Note)}
}

func factIsEnding(fact Fact) bool {
	switch fact.Cause.(type) {
	case RunCompleted, RunFailed, RunCanceled:
		return true
	default:
		return false
	}
}

func hasEndingFact(facts []Fact) bool {
	return slices.ContainsFunc(facts, factIsEnding)
}

func hasAcceptedFact(facts []Fact) bool {
	return slices.ContainsFunc(facts, func(fact Fact) bool {
		_, ok := fact.Cause.(JobAccepted)
		return ok
	})
}
