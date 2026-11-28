package media2docbus_test

import (
	"errors"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/google/uuid"
)

func TestEligible(t *testing.T) {
	owner, project := uuid.New(), uuid.New()
	src := media2docbus.Source{OwnerID: owner, ProjectID: project, Ready: true}

	if err := src.Eligible(owner, project); err != nil {
		t.Errorf("the owner's ready recording was refused: %v", err)
	}
	if err := src.Eligible(uuid.New(), project); !errors.Is(err, media2docbus.ErrSourceNotInProject) {
		t.Errorf("another owner: %v", err)
	}
	if err := src.Eligible(owner, uuid.New()); !errors.Is(err, media2docbus.ErrSourceNotInProject) {
		t.Errorf("another project: %v", err)
	}
	src.Ready = false
	if err := src.Eligible(owner, project); !errors.Is(err, media2docbus.ErrSourceNotReady) {
		t.Errorf("not ready: %v", err)
	}
}

func TestPublishReplacesAStepsOwnFile(t *testing.T) {
	files := media2docbus.NewTestFiles()
	runID := uuid.New()
	out := media2docbus.Output{RunID: runID, Kind: media2docbus.KindAudio, Name: "a.wav", Body: []byte("one")}

	first, err := media2docbus.Publish(t.Context(), files, out)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	out.Body = []byte("two")
	second, err := media2docbus.Publish(t.Context(), files, out)
	if err != nil {
		t.Fatalf("publish again: %v", err)
	}

	if first != second || len(files.Outputs) != 1 || string(files.Outputs[first].Body) != "two" {
		t.Errorf("ids %s %s, outputs %d", first, second, len(files.Outputs))
	}
	if other := media2docbus.OutputFileID(runID, media2docbus.KindTranscript); other == first {
		t.Error("two kinds of file share an id")
	}
	if other := media2docbus.OutputFileID(uuid.New(), media2docbus.KindAudio); other == first {
		t.Error("two runs share an id")
	}
}

func TestStem(t *testing.T) {
	for name, want := range map[string]string{"week-03.mp4": "week-03", "a.b.wav": "a.b", "noext": "noext", "  ": "recording", ".hidden": ".hidden"} {
		if got := media2docbus.Stem(name); got != want {
			t.Errorf("stem(%q) = %q, want %q", name, got, want)
		}
	}
}
