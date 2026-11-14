package sim

import (
	"testing"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/app/domain/pipelineapp/activities/audioextract"
)

func TestRemainingText(t *testing.T) {
	tests := []struct {
		left time.Duration
		want string
	}{
		{0, ""},
		{20 * time.Second, "Less than a minute left."},
		{time.Minute, "About a minute left."},
		{2*time.Minute + 10*time.Second, "About 2 min left."},
	}
	for _, test := range tests {
		if got := remainingText(test.left); got != test.want {
			t.Fatalf("remaining %s = %q, want %q", test.left, got, test.want)
		}
	}
}

func TestStubFractionReachesOneAtTheEnd(t *testing.T) {
	if got := stubFraction(0, stubRun); got != 0 {
		t.Fatalf("start = %v", got)
	}
	if got := stubFraction(stubRun/2, stubRun); got != 0.5 {
		t.Fatalf("middle = %v", got)
	}
	if got := stubFraction(stubRun+time.Second, stubRun); got != 1 {
		t.Fatalf("end = %v", got)
	}
}

func TestAudioProgressDoesNotLeap(t *testing.T) {
	var live audioLive
	live.noteDownload(100, 100)
	measured, ok := live.measured(-1)
	if !ok || measured != 1 {
		t.Fatalf("download = %v %v", measured, ok)
	}
	shown := advanceProgress(-1, false, measured)
	if shown != audioProgressStep {
		t.Fatalf("first shown = %v", shown)
	}

	live.startEncode()
	held, ok := live.measured(shown)
	if !ok || held != shown {
		t.Fatalf("held = %v %v", held, ok)
	}
	live.setEncode(audioextract.Progress{Fraction: 0.8})
	target, ok := live.measured(shown)
	if !ok || target != shown+0.8*(1-shown) {
		t.Fatalf("target = %v", target)
	}
	if next := advanceProgress(shown, true, target); next != shown+audioProgressStep {
		t.Fatalf("step = %v", next)
	}
	if back := advanceProgress(0.4, true, 0.1); back != 0.4 {
		t.Fatalf("back = %v", back)
	}
}

func TestAudioLiveKeepsTheEncodeSample(t *testing.T) {
	var live audioLive
	live.noteDownload(100, 40)
	live.setEncode(audioextract.Progress{Fraction: 0.5, Remaining: 2 * time.Minute})

	live.noteDownload(100, 80)
	fraction, hasFraction, message, ok := live.snapshot()
	if !ok || !hasFraction || fraction != 0.5 || message != "Writing the audio. About 2 min left." {
		t.Fatalf("sample = %v %v %q %v", fraction, hasFraction, message, ok)
	}
}
