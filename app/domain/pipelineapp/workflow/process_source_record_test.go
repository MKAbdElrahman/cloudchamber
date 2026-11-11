//go:build updatebaseline

package workflow

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/temporalproto"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
)

// TestRecordProcessSourceBaseline runs ProcessSourceWorkflow against the
// Temporal dev server and writes the history file the replay test keeps.
// Later slices must keep that file replaying.
func TestRecordProcessSourceBaseline(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()

	srv, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		LogLevel: "error",
		Stdout:   io.Discard,
		Stderr:   io.Discard,
	})
	if err != nil {
		t.Fatalf("start dev server: %v", err)
	}
	t.Cleanup(func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("stop dev server: %v", err)
		}
	})

	c := srv.Client()
	startWorkflowWorker(t, c)
	startActivityWorker(t, c, TaskQueueMedia, ActivityResolveSource, ActivityExtractAudio)
	startActivityWorker(t, c, TaskQueueASR, ActivityTranscribe)
	startActivityWorker(t, c, TaskQueueFrames, ActivityExtractKeyFrames)
	startActivityWorker(t, c, TaskQueueRender, ActivityComposeExport)

	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        "process-source-baseline",
		TaskQueue: TaskQueueWorkflow,
	}, ProcessSourceWorkflow, ProcessSourceInput{
		JobID:   baselineJobID,
		OwnerID: baselineOwnerID,
		FileID:  baselineFileID,
	})
	if err != nil {
		t.Fatalf("start workflow: %v", err)
	}

	var got ProcessSourceResult
	if err := run.Get(ctx, &got); err != nil {
		t.Fatalf("workflow result: %v", err)
	}
	if got.DocumentFileID != baselineDocumentID {
		t.Fatalf("document file id = %s", got.DocumentFileID)
	}

	hist := readHistory(t, ctx, c, run.GetID(), run.GetRunID())
	scheduled := scheduledActivities(t, hist)
	if diff := stepDiff(baselineOrder, scheduled); diff != "" {
		t.Fatalf("recorded order:\n%s", diff)
	}

	raw, err := temporalproto.CustomJSONMarshalOptions{Indent: "  "}.Marshal(hist)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(baselineHistoryFile), 0o755); err != nil {
		t.Fatalf("mkdir testdata: %v", err)
	}
	if err := os.WriteFile(baselineHistoryFile, raw, 0o644); err != nil {
		t.Fatalf("write baseline history: %v", err)
	}
}

func startWorkflowWorker(t *testing.T, c client.Client) {
	t.Helper()

	w := worker.New(c, TaskQueueWorkflow, worker.Options{})
	w.RegisterWorkflow(ProcessSourceWorkflow)
	if err := w.Start(); err != nil {
		t.Fatalf("start workflow worker: %v", err)
	}
	t.Cleanup(w.Stop)
}

func startActivityWorker(t *testing.T, c client.Client, queue string, names ...string) {
	t.Helper()

	w := worker.New(c, queue, worker.Options{})
	for _, name := range names {
		w.RegisterActivityWithOptions(stubActivity(name), activity.RegisterOptions{Name: name})
	}
	if err := w.Start(); err != nil {
		t.Fatalf("start %s worker: %v", queue, err)
	}
	t.Cleanup(w.Stop)
}

func stubActivity(name string) any {
	switch name {
	case ActivityResolveSource:
		return resolveSourceActivity
	case ActivityExtractAudio:
		return extractAudioActivity
	case ActivityTranscribe:
		return transcribeActivity
	case ActivityExtractKeyFrames:
		return extractKeyFramesActivity
	case ActivityComposeExport:
		return composeExportActivity
	default:
		panic("unknown activity " + name)
	}
}

func readHistory(t *testing.T, ctx context.Context, c client.Client, workflowID, runID string) *historypb.History {
	t.Helper()

	iter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []*historypb.HistoryEvent
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			t.Fatalf("read history: %v", err)
		}
		events = append(events, event)
	}
	return &historypb.History{Events: events}
}
