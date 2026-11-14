package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestProcessSourceWorkflow_CancelDuringActivityRecordsTheRun(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(resolveSourceActivity, activity.RegisterOptions{Name: ActivityResolveSource})
	env.RegisterActivityWithOptions(extractAudioActivity, activity.RegisterOptions{Name: ActivityExtractAudio})
	env.RegisterActivityWithOptions(func(ctx context.Context, _ TranscribeInput) (TranscribeOutput, error) {
		<-ctx.Done()
		return TranscribeOutput{}, ctx.Err()
	}, activity.RegisterOptions{Name: ActivityTranscribe})
	env.RegisterActivityWithOptions(extractKeyFramesActivity, activity.RegisterOptions{Name: ActivityExtractKeyFrames})
	env.RegisterActivityWithOptions(composeExportActivity, activity.RegisterOptions{Name: ActivityComposeExport})
	var recorded RecordRunInput
	env.RegisterActivityWithOptions(func(_ context.Context, in RecordRunInput) error {
		recorded = in
		return nil
	}, activity.RegisterOptions{Name: ActivityRecordRun})

	env.RegisterDelayedCallback(func() {
		env.CancelWorkflow()
	}, time.Second)

	env.ExecuteWorkflow(ProcessSourceWorkflow, ProcessSourceInput{
		JobID:   baselineJobID,
		OwnerID: baselineOwnerID,
		FileID:  baselineFileID,
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); !temporal.IsCanceledError(err) {
		t.Fatalf("workflow error = %v", err)
	}
	if recorded.JobID != baselineJobID || recorded.Fact != "runCanceled" || recorded.Message != "" {
		t.Fatalf("record run = %+v", recorded)
	}
}

func TestProcessSourceWorkflow_ReturnsTheActivityFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, ResolveSourceInput) (ResolveSourceOutput, error) {
		return ResolveSourceOutput{}, errors.New("source missing")
	}, activity.RegisterOptions{Name: ActivityResolveSource})
	var recorded RecordRunInput
	env.RegisterActivityWithOptions(func(_ context.Context, in RecordRunInput) error {
		recorded = in
		return nil
	}, activity.RegisterOptions{Name: ActivityRecordRun})

	env.ExecuteWorkflow(ProcessSourceWorkflow, ProcessSourceInput{
		JobID:   baselineJobID,
		OwnerID: baselineOwnerID,
		FileID:  baselineFileID,
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	err := env.GetWorkflowError()
	if err == nil || !strings.Contains(err.Error(), "source missing") {
		t.Fatalf("workflow error = %v", err)
	}
	if recorded.JobID != baselineJobID || recorded.Fact != "runFailed" || !strings.Contains(recorded.Message, "source missing") {
		t.Fatalf("record run = %+v", recorded)
	}
}

func TestRunStep_DoesNotStartWhenTheRunIsCanceled(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	framesStarted := false
	env.RegisterActivityWithOptions(func(context.Context, ExtractKeyFramesInput) (ExtractKeyFramesOutput, error) {
		framesStarted = true
		return ExtractKeyFramesOutput{}, nil
	}, activity.RegisterOptions{Name: ActivityExtractKeyFrames})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		canceled, cancel := workflow.WithCancel(ctx)
		cancel()
		return runStep(canceled, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		}, ActivityExtractKeyFrames, ExtractKeyFramesInput{}, &ExtractKeyFramesOutput{})
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); !temporal.IsCanceledError(err) {
		t.Fatalf("workflow error = %v", err)
	}
	if framesStarted {
		t.Fatal("ExtractKeyFrames started after the workflow context was canceled")
	}
}
