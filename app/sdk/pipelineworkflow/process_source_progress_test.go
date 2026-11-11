package workflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestProcessSourceWorkflow_QueryNamesTheCanceledStep(t *testing.T) {
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
	marked := false
	env.RegisterActivityWithOptions(func(context.Context, MarkCanceledInput) error {
		marked = true
		return nil
	}, activity.RegisterOptions{Name: ActivityMarkCanceled})

	env.RegisterDelayedCallback(func() {
		encoded, err := env.QueryWorkflow(QueryProgress)
		if err != nil {
			t.Errorf("query while transcribing: %v", err)
			return
		}
		var progress ProcessSourceProgress
		if err := encoded.Get(&progress); err != nil {
			t.Errorf("decode progress: %v", err)
			return
		}
		if progress.Activity != ActivityTranscribe {
			t.Errorf("progress while transcribing = %+v", progress)
		}
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

	encoded, err := env.QueryWorkflow(QueryProgress)
	if err != nil {
		t.Fatalf("query after cancel: %v", err)
	}
	var progress ProcessSourceProgress
	if err := encoded.Get(&progress); err != nil {
		t.Fatalf("decode progress: %v", err)
	}
	if progress.Activity != ActivityTranscribe {
		t.Fatalf("progress after cancel = %+v", progress)
	}
	if marked {
		t.Fatal("mark canceled ran for an activity that was already in progress")
	}
}

func TestCallActivity_StopsBeforeSchedulingWhenContextIsCanceled(t *testing.T) {
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
		progress := ProcessSourceProgress{Activity: ActivityTranscribe}
		err := callActivity(canceled, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		}, &progress, ActivityExtractKeyFrames, ExtractKeyFramesInput{}, &ExtractKeyFramesOutput{})
		if !errors.Is(err, workflow.ErrCanceled) {
			return fmt.Errorf("callActivity error = %v", err)
		}
		if progress.Activity != ActivityTranscribe {
			return fmt.Errorf("progress = %+v", progress)
		}
		return err
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

func TestProcessSourceWorkflow_ReturnsTheActivityFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.RegisterActivityWithOptions(func(context.Context, ResolveSourceInput) (ResolveSourceOutput, error) {
		return ResolveSourceOutput{}, errors.New("source missing")
	}, activity.RegisterOptions{Name: ActivityResolveSource})
	marked := false
	env.RegisterActivityWithOptions(func(context.Context, MarkCanceledInput) error {
		marked = true
		return nil
	}, activity.RegisterOptions{Name: ActivityMarkCanceled})

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
	if marked {
		t.Fatal("mark canceled ran for a failed activity")
	}
}

func TestRunStep_MarksCancelBeforeTheNextSchedule(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	framesStarted := false
	env.RegisterActivityWithOptions(func(context.Context, ExtractKeyFramesInput) (ExtractKeyFramesOutput, error) {
		framesStarted = true
		return ExtractKeyFramesOutput{}, nil
	}, activity.RegisterOptions{Name: ActivityExtractKeyFrames})
	var marked MarkCanceledInput
	env.RegisterActivityWithOptions(func(_ context.Context, in MarkCanceledInput) error {
		marked = in
		return nil
	}, activity.RegisterOptions{Name: ActivityMarkCanceled})

	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		canceled, cancel := workflow.WithCancel(ctx)
		cancel()
		progress := ProcessSourceProgress{Activity: ActivityTranscribe}
		return runStep(canceled, baselineJobID, &progress, workflow.ActivityOptions{
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
	if marked.JobID != baselineJobID || marked.Step != ActivityTranscribe {
		t.Fatalf("mark canceled = %+v", marked)
	}
}
