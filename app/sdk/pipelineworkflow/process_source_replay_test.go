package workflow

import (
	"bytes"
	"context"
	"os"
	"testing"

	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

const baselineHistoryFile = "testdata/process_source_baseline.json"

const (
	baselineJobID        = "job-baseline"
	baselineOwnerID      = "11111111-1111-1111-1111-111111111111"
	baselineProjectID    = "33333333-3333-3333-3333-333333333333"
	baselineFileID       = "22222222-2222-2222-2222-222222222222"
	baselineAudioID      = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	baselineTranscriptID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	baselineFramesID     = "cccccccc-cccc-cccc-cccc-cccccccccccc"
	baselineDocumentID   = "dddddddd-dddd-dddd-dddd-dddddddddddd"
)

// baselineOrder is the execution order of a successful video run.
// Each step is one activity, scheduled by its own workflow task.
var baselineOrder = []activityStep{
	{name: ActivityResolveSource, queue: TaskQueueMedia},
	{name: ActivityExtractAudio, queue: TaskQueueMedia},
	{name: ActivityTranscribe, queue: TaskQueueASR},
	{name: ActivityExtractKeyFrames, queue: TaskQueueFrames},
	{name: ActivityComposeExport, queue: TaskQueueRender},
	{name: ActivityRecordRun, queue: TaskQueueWorkflow},
}

type activityStep struct {
	name  string
	queue string
}

func TestProcessSourceWorkflow_ReplaysBaseline(t *testing.T) {
	recorded := executeProcessSource(t)
	if diff := stepDiff(baselineOrder, recorded); diff != "" {
		t.Fatalf("executed order:\n%s", diff)
	}

	hist := loadBaselineHistory(t)
	scheduled := scheduledActivities(t, hist)
	if diff := stepDiff(baselineOrder, scheduled); diff != "" {
		t.Fatalf("baseline history order:\n%s", diff)
	}
	assertOneWorkflowTaskPerActivity(t, hist)
	assertCompletionAfterLastActivity(t, hist)

	replayer := worker.NewWorkflowReplayer()
	replayer.RegisterWorkflow(ProcessSourceWorkflow)
	if err := replayer.ReplayWorkflowHistory(nil, hist); err != nil {
		t.Fatalf("replay baseline history: %v", err)
	}
}

func executeProcessSource(t *testing.T) []activityStep {
	t.Helper()

	var started []activityStep
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	env.SetOnActivityStartedListener(func(info *activity.Info, _ context.Context, _ converter.EncodedValues) {
		started = append(started, activityStep{name: info.ActivityType.Name, queue: info.TaskQueue})
	})
	registerStubActivities(env)
	env.ExecuteWorkflow(ProcessSourceWorkflow, ProcessSourceInput{
		JobID:   baselineJobID,
		OwnerID: baselineOwnerID,
		FileID:  baselineFileID,
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	return started
}

func loadBaselineHistory(t *testing.T) *historypb.History {
	t.Helper()

	raw, err := os.ReadFile(baselineHistoryFile)
	if err != nil {
		t.Fatalf("read baseline history: %v (record it with go test -tags updatebaseline -run TestRecordProcessSourceBaseline)", err)
	}
	hist, err := client.HistoryFromJSON(bytes.NewReader(raw), client.HistoryJSONOptions{})
	if err != nil {
		t.Fatalf("decode baseline history: %v", err)
	}
	return hist
}

func scheduledActivities(t *testing.T, hist *historypb.History) []activityStep {
	t.Helper()

	var steps []activityStep
	for _, event := range hist.GetEvents() {
		if event.GetEventType() != enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED {
			continue
		}
		attrs := event.GetActivityTaskScheduledEventAttributes()
		steps = append(steps, activityStep{
			name:  attrs.GetActivityType().GetName(),
			queue: attrs.GetTaskQueue().GetName(),
		})
	}
	if len(steps) == 0 {
		t.Fatal("baseline history scheduled no activities")
	}
	return steps
}

func assertOneWorkflowTaskPerActivity(t *testing.T, hist *historypb.History) {
	t.Helper()

	var prev int64
	for _, event := range hist.GetEvents() {
		if event.GetEventType() != enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED {
			continue
		}
		taskID := event.GetActivityTaskScheduledEventAttributes().GetWorkflowTaskCompletedEventId()
		name := event.GetActivityTaskScheduledEventAttributes().GetActivityType().GetName()
		if taskID <= prev {
			t.Fatalf("%s was scheduled by workflow task event %d, which is not after %d", name, taskID, prev)
		}
		prev = taskID
	}
}

func assertCompletionAfterLastActivity(t *testing.T, hist *historypb.History) {
	t.Helper()

	var lastActivityID int64
	var completionTaskID int64
	var completed bool
	for _, event := range hist.GetEvents() {
		switch event.GetEventType() {
		case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
			lastActivityID = event.GetEventId()
		case enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED:
			if event.GetEventId() > lastActivityID {
				completionTaskID = event.GetEventId()
			}
		case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
			completed = true
		}
	}
	if lastActivityID == 0 || completionTaskID == 0 || !completed {
		t.Fatalf("history missing the completion workflow task after the last activity (last activity %d, completion task %d, completed %v)", lastActivityID, completionTaskID, completed)
	}
}

func stepDiff(want, got []activityStep) string {
	if len(want) != len(got) {
		return formatSteps(want, got)
	}
	for i := range want {
		if want[i] != got[i] {
			return formatSteps(want, got)
		}
	}
	return ""
}

func formatSteps(want, got []activityStep) string {
	var buf bytes.Buffer
	buf.WriteString("want:\n")
	for _, step := range want {
		buf.WriteString("  ")
		buf.WriteString(step.name)
		buf.WriteString(" on ")
		buf.WriteString(step.queue)
		buf.WriteByte('\n')
	}
	buf.WriteString("got:\n")
	for _, step := range got {
		buf.WriteString("  ")
		buf.WriteString(step.name)
		buf.WriteString(" on ")
		buf.WriteString(step.queue)
		buf.WriteByte('\n')
	}
	return buf.String()
}

func registerStubActivities(env *testsuite.TestWorkflowEnvironment) {
	env.RegisterActivityWithOptions(resolveSourceActivity, activity.RegisterOptions{Name: ActivityResolveSource})
	env.RegisterActivityWithOptions(extractAudioActivity, activity.RegisterOptions{Name: ActivityExtractAudio})
	env.RegisterActivityWithOptions(transcribeActivity, activity.RegisterOptions{Name: ActivityTranscribe})
	env.RegisterActivityWithOptions(extractKeyFramesActivity, activity.RegisterOptions{Name: ActivityExtractKeyFrames})
	env.RegisterActivityWithOptions(composeExportActivity, activity.RegisterOptions{Name: ActivityComposeExport})
	env.RegisterActivityWithOptions(func(context.Context, RecordRunInput) error { return nil }, activity.RegisterOptions{Name: ActivityRecordRun})
}

func resolveSourceActivity(_ context.Context, in ResolveSourceInput) (ResolveSourceOutput, error) {
	return ResolveSourceOutput{
		JobID:       in.JobID,
		OwnerID:     in.OwnerID,
		ProjectID:   baselineProjectID,
		FileID:      in.FileID,
		Name:        "week-03.mp4",
		ContentType: "video/mp4",
	}, nil
}

func extractAudioActivity(_ context.Context, in ExtractAudioInput) (ExtractAudioOutput, error) {
	return ExtractAudioOutput{
		JobID:   in.JobID,
		OwnerID: in.OwnerID,
		FileID:  baselineAudioID,
		Name:    "week-03.wav",
	}, nil
}

func transcribeActivity(_ context.Context, in TranscribeInput) (TranscribeOutput, error) {
	return TranscribeOutput{JobID: in.JobID, FileID: baselineTranscriptID}, nil
}

func extractKeyFramesActivity(_ context.Context, in ExtractKeyFramesInput) (ExtractKeyFramesOutput, error) {
	return ExtractKeyFramesOutput{JobID: in.JobID, FileID: baselineFramesID}, nil
}

func composeExportActivity(_ context.Context, in ComposeExportInput) (ComposeExportOutput, error) {
	return ComposeExportOutput{JobID: in.JobID, FileID: baselineDocumentID}, nil
}
