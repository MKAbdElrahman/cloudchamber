package videoworkflow

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/workflows/videoworkflow/audioextract"
	"github.com/MKAbdElrahman/cloudchamber/app/sdk/runcontrol"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func input(fileID string) runcontrol.Input {
	body, _ := json.Marshal(Input{FileID: fileID})
	return runcontrol.Input{RunID: "run-1", Workflow: Name, OwnerID: "owner-1", ProjectID: "project-1", Input: body}
}

// calls records which activities the workflow scheduled, and with what.
type calls struct {
	order     []string
	resolve   ResolveInput
	transcrib TranscribeInput
	compose   ComposeInput
}

func register(env *testsuite.TestWorkflowEnvironment, contentType string, failAt string) *calls {
	seen := &calls{}
	source := Source{RunID: "run-1", OwnerID: "owner-1", ProjectID: "project-1", FileID: "file-1", Name: "week-03.mp4", ContentType: contentType}
	fail := func(name string) error {
		seen.order = append(seen.order, name)
		if name == failAt {
			return temporal.NewNonRetryableApplicationError("stopped here", "Test", nil)
		}
		return nil
	}

	env.RegisterActivityWithOptions(func(in ResolveInput) (Source, error) {
		seen.resolve = in
		return source, fail(ActivityResolveSource)
	}, activity.RegisterOptions{Name: ActivityResolveSource})
	env.RegisterActivityWithOptions(func(Source) (File, error) {
		return File{FileID: "audio-1"}, fail(ActivityExtractAudio)
	}, activity.RegisterOptions{Name: ActivityExtractAudio})
	env.RegisterActivityWithOptions(func(in TranscribeInput) (File, error) {
		seen.transcrib = in
		return File{FileID: "transcript-1"}, fail(ActivityTranscribe)
	}, activity.RegisterOptions{Name: ActivityTranscribe})
	env.RegisterActivityWithOptions(func(Source) (File, error) {
		return File{FileID: "frames-1"}, fail(ActivityExtractKeyFrames)
	}, activity.RegisterOptions{Name: ActivityExtractKeyFrames})
	env.RegisterActivityWithOptions(func(in ComposeInput) (File, error) {
		seen.compose = in
		return File{FileID: "document-1"}, fail(ActivityComposeDocument)
	}, activity.RegisterOptions{Name: ActivityComposeDocument})

	return seen
}

func TestAVideoRunsEveryStepInOrder(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	seen := register(env, "video/mp4", "")

	env.ExecuteWorkflow(ToDocument, input(" file-1 "))

	var result Result
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("workflow: %s", err)
	}
	want := Result{SourceFileID: "file-1", AudioFileID: "audio-1", TranscriptFileID: "transcript-1", KeyFramesFileID: "frames-1", DocumentFileID: "document-1"}
	if result != want {
		t.Errorf("result = %+v", result)
	}

	order := []string{ActivityResolveSource, ActivityExtractAudio, ActivityTranscribe, ActivityExtractKeyFrames, ActivityComposeDocument}
	if len(seen.order) != len(order) {
		t.Fatalf("steps = %v", seen.order)
	}
	for i := range order {
		if seen.order[i] != order[i] {
			t.Fatalf("steps = %v", seen.order)
		}
	}

	if seen.resolve != (ResolveInput{RunID: "run-1", OwnerID: "owner-1", ProjectID: "project-1", FileID: "file-1"}) {
		t.Errorf("resolve input = %+v", seen.resolve)
	}
	if seen.transcrib.AudioFileID != "audio-1" || seen.compose.TranscriptFileID != "transcript-1" || seen.compose.KeyFramesFileID != "frames-1" {
		t.Errorf("transcribe = %+v compose = %+v", seen.transcrib, seen.compose)
	}
}

func TestAnAudioRecordingHasNoKeyFrameStep(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	seen := register(env, "audio/mpeg", "")

	env.ExecuteWorkflow(ToDocument, input("file-1"))

	var result Result
	if err := env.GetWorkflowResult(&result); err != nil {
		t.Fatalf("workflow: %s", err)
	}
	if result.KeyFramesFileID != "" || result.DocumentFileID != "document-1" || seen.compose.KeyFramesFileID != "" {
		t.Errorf("result = %+v compose = %+v", result, seen.compose)
	}
	for _, name := range seen.order {
		if name == ActivityExtractKeyFrames {
			t.Fatalf("key frames ran for audio: %v", seen.order)
		}
	}
}

func TestAFailedStepStopsTheRun(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	seen := register(env, "video/mp4", ActivityTranscribe)

	env.ExecuteWorkflow(ToDocument, input("file-1"))

	err := env.GetWorkflowError()
	if err == nil {
		t.Fatal("the workflow succeeded")
	}
	appErr, ok := errors.AsType[*temporal.ApplicationError](err)
	if !ok || appErr.Message() != "stopped here" {
		t.Errorf("err = %v", err)
	}
	if last := seen.order[len(seen.order)-1]; last != ActivityTranscribe || len(seen.order) != 3 {
		t.Errorf("steps = %v", seen.order)
	}
}

func TestInputWithoutAFileIsRefused(t *testing.T) {
	for _, body := range []string{`{}`, `{"fileId": "  "}`, `[]`} {
		var suite testsuite.WorkflowTestSuite
		env := suite.NewTestWorkflowEnvironment()
		seen := register(env, "video/mp4", "")

		env.ExecuteWorkflow(ToDocument, runcontrol.Input{RunID: "run-1", Input: json.RawMessage(body)})

		if err := env.GetWorkflowError(); err == nil {
			t.Errorf("input %s was accepted", body)
		}
		if len(seen.order) != 0 {
			t.Errorf("input %s scheduled %v", body, seen.order)
		}
	}
}

func TestStem(t *testing.T) {
	for name, want := range map[string]string{"week-03.mp4": "week-03", "a.b.wav": "a.b", "noext": "noext", "  ": "recording", ".hidden": ".hidden"} {
		if got := stem(name); got != want {
			t.Errorf("stem(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestAudioProgressFillsTheBarInTwoHalves(t *testing.T) {
	var p audioProgress

	if fraction, _ := p.sample(0, 10); fraction >= 0 {
		t.Errorf("unknown size reported %v", fraction)
	}
	if fraction, message := p.sample(100, 50); fraction != 0.25 || message != "Downloading the recording." {
		t.Errorf("half downloaded = %v %q", fraction, message)
	}

	p.startEncode()
	if fraction, message := p.sample(100, 100); fraction != 0.5 || message != "Writing the audio." {
		t.Errorf("encode start = %v %q", fraction, message)
	}

	p.setEncode(audioextractProgress(0.5))
	if fraction, _ := p.sample(100, 100); fraction != 0.75 {
		t.Errorf("half encoded = %v", fraction)
	}
}

func audioextractProgress(fraction float64) audioextract.Progress {
	return audioextract.Progress{Fraction: fraction}
}
