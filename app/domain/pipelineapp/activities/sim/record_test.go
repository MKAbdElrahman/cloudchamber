package sim

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	workflow "github.com/MKAbdElrahman/cloudchamber/app/sdk/pipelineworkflow"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

func TestFileUpdateAssignsTheColumnForTheKind(t *testing.T) {
	fileID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	tests := []struct {
		kind string
		got  func(pipelinejobbus.UpdateJob) *uuid.UUID
	}{
		{kind: artifactbus.KindAudio, got: func(uj pipelinejobbus.UpdateJob) *uuid.UUID { return uj.AudioFileID }},
		{kind: artifactbus.KindTranscript, got: func(uj pipelinejobbus.UpdateJob) *uuid.UUID { return uj.TranscriptFileID }},
		{kind: artifactbus.KindFrames, got: func(uj pipelinejobbus.UpdateJob) *uuid.UUID { return uj.KeyFramesFileID }},
		{kind: artifactbus.KindDocument, got: func(uj pipelinejobbus.UpdateJob) *uuid.UUID { return uj.DocumentFileID }},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			uj, changed, err := fileUpdate(pipelinejobbus.Job{}, tt.kind, fileID)
			if err != nil || !changed {
				t.Fatalf("changed=%v err=%v", changed, err)
			}
			got := tt.got(uj)
			if got == nil || *got != fileID {
				t.Fatalf("file = %v", got)
			}
		})
	}
}

func TestFileUpdateLeavesAStoredFile(t *testing.T) {
	fileID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	job := pipelinejobbus.Job{AudioFileID: fileID}

	uj, changed, err := fileUpdate(job, artifactbus.KindAudio, fileID)
	if err != nil || changed || uj.AudioFileID != nil {
		t.Fatalf("changed=%v update=%v err=%v", changed, uj.AudioFileID, err)
	}
}

func TestFileUpdateRejectsUnknownKind(t *testing.T) {
	_, changed, err := fileUpdate(pipelinejobbus.Job{}, "video", uuid.New())
	if err == nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}

func TestOutcomeUpdateStoresTheClosedJob(t *testing.T) {
	running := pipelinejobbus.Job{Status: pipelinejobbus.MustParseStatus(pipelinejobbus.StatusRunning), ErrorMessage: new("earlier")}

	completed, changed, err := outcomeUpdate(running, pipelinejobbus.MustParseStatus(pipelinejobbus.StatusCompleted), "", "")
	if err != nil || !changed || !completed.ClearError || completed.Status == nil || completed.Status.String() != pipelinejobbus.StatusCompleted || completed.StoppedStep == nil || *completed.StoppedStep != "" {
		t.Fatalf("completed changed=%v update=%+v err=%v", changed, completed, err)
	}

	canceled, changed, err := outcomeUpdate(running, pipelinejobbus.MustParseStatus(pipelinejobbus.StatusCanceled), workflow.ActivityTranscribe, "")
	if err != nil || !changed || !canceled.ClearError || canceled.StoppedStep == nil || *canceled.StoppedStep != workflow.ActivityTranscribe {
		t.Fatalf("canceled changed=%v update=%+v err=%v", changed, canceled, err)
	}

	failed, changed, err := outcomeUpdate(running, pipelinejobbus.MustParseStatus(pipelinejobbus.StatusFailed), workflow.ActivityResolveSource, "")
	if err != nil || !changed || failed.ErrorMessage == nil || *failed.ErrorMessage != pipelinejobbus.StatusFailed || failed.StoppedStep == nil || *failed.StoppedStep != workflow.ActivityResolveSource {
		t.Fatalf("failed changed=%v update=%+v err=%v", changed, failed, err)
	}
}

func TestOutcomeUpdateLeavesAStoredOutcome(t *testing.T) {
	job := pipelinejobbus.Job{
		Status:      pipelinejobbus.MustParseStatus(pipelinejobbus.StatusCanceled),
		StoppedStep: workflow.ActivityTranscribe,
	}

	_, changed, err := outcomeUpdate(job, job.Status, job.StoppedStep, "")
	if err != nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}

func TestOutcomeUpdateRejectsARunStatus(t *testing.T) {
	_, changed, err := outcomeUpdate(pipelinejobbus.Job{}, pipelinejobbus.MustParseStatus(pipelinejobbus.StatusRunning), "", "")
	if err == nil || changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
}

func TestRecordOutcomeRejectsARunStatus(t *testing.T) {
	err := (&Activities{}).recordOutcome(context.Background(), "11111111-1111-1111-1111-111111111111", "terminated", "", "")

	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	if !ok || !appErr.NonRetryable() || appErr.Type() != errTypeInvalidInput {
		t.Fatalf("record error = %v", err)
	}
}

func TestFinishRecordsAPermanentFailure(t *testing.T) {
	cause := temporal.NewNonRetryableApplicationError("load source", "SourceNotFound", nil)
	err := (&Activities{}).finish(context.Background(), "11111111-1111-1111-1111-111111111111", workflow.ActivityResolveSource, cause)
	if err == nil || !strings.Contains(err.Error(), "pipeline jobs") {
		t.Fatalf("finish error = %v", err)
	}
}

func TestFinishLeavesARetryableFailure(t *testing.T) {
	cause := errors.New("temporary")
	err := (&Activities{}).finish(context.Background(), "job", workflow.ActivityResolveSource, cause)
	if !errors.Is(err, cause) {
		t.Fatalf("finish error = %v", err)
	}
}

func TestFinishDoesNotReportCancelUntilTheJobIsStored(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (&Activities{}).finish(ctx, "11111111-1111-1111-1111-111111111111", workflow.ActivityTranscribe, ctx.Err())
	if err == nil || !strings.Contains(err.Error(), "pipeline jobs") {
		t.Fatalf("finish error = %v", err)
	}
}

func TestRecordFileRejectsABadJobID(t *testing.T) {
	err := (&Activities{}).recordFile(context.Background(), "job", artifactbus.KindAudio, uuid.New())

	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	if !ok || !appErr.NonRetryable() || appErr.Type() != errTypeInvalidInput {
		t.Fatalf("record error = %v", err)
	}
}
