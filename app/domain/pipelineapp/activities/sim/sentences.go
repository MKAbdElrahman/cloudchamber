package sim

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

func published(fileID string, build func(uuid.UUID) pipelinejobbus.Cause) (pipelinejobbus.Cause, error) {
	id, err := uuid.Parse(fileID)
	if err != nil || id == uuid.Nil {
		return nil, fmt.Errorf("file id is required")
	}
	return build(id), nil
}

func sourceDone(string) (pipelinejobbus.Cause, error) {
	return pipelinejobbus.SourceResolved{}, nil
}

func sourceFailed(note *string) pipelinejobbus.Cause {
	return pipelinejobbus.SourceFailed{Note: note}
}

func sourceCanceled() pipelinejobbus.Cause { return pipelinejobbus.SourceCanceled{} }

func audioDone(fileID string) (pipelinejobbus.Cause, error) {
	return published(fileID, func(id uuid.UUID) pipelinejobbus.Cause {
		return pipelinejobbus.AudioPublished{FileID: id}
	})
}

func audioFailed(note *string) pipelinejobbus.Cause {
	return pipelinejobbus.AudioFailed{Note: note}
}

func audioCanceled() pipelinejobbus.Cause { return pipelinejobbus.AudioCanceled{} }

func transcriptDone(fileID string) (pipelinejobbus.Cause, error) {
	return published(fileID, func(id uuid.UUID) pipelinejobbus.Cause {
		return pipelinejobbus.TranscriptPublished{FileID: id}
	})
}

func transcriptFailed(note *string) pipelinejobbus.Cause {
	return pipelinejobbus.TranscriptFailed{Note: note}
}

func transcriptCanceled() pipelinejobbus.Cause { return pipelinejobbus.TranscriptCanceled{} }

func keyFramesDone(fileID string) (pipelinejobbus.Cause, error) {
	return published(fileID, func(id uuid.UUID) pipelinejobbus.Cause {
		return pipelinejobbus.KeyFramesPublished{FileID: id}
	})
}

func keyFramesFailed(note *string) pipelinejobbus.Cause {
	return pipelinejobbus.KeyFramesFailed{Note: note}
}

func keyFramesCanceled() pipelinejobbus.Cause { return pipelinejobbus.KeyFramesCanceled{} }

func documentDone(fileID string) (pipelinejobbus.Cause, error) {
	return published(fileID, func(id uuid.UUID) pipelinejobbus.Cause {
		return pipelinejobbus.DocumentPublished{FileID: id}
	})
}

func documentFailed(note *string) pipelinejobbus.Cause {
	return pipelinejobbus.DocumentFailed{Note: note}
}

func documentCanceled() pipelinejobbus.Cause { return pipelinejobbus.DocumentCanceled{} }
