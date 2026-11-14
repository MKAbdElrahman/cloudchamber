package pipelinejobbus

import (
	"testing"
	"time"
	"unicode"

	"github.com/google/uuid"
)

func TestRunFactIDStable(t *testing.T) {
	jobID := uuid.MustParse("9613ff77-bef0-5b9a-9925-123e82aca8d1")
	first := RunFactID(jobID)
	if first != RunFactID(jobID) {
		t.Fatal("run fact id changed")
	}
	if first == jobID {
		t.Fatal("run fact id reused the job id")
	}
	other := uuid.MustParse("79b64abf-27f9-5638-8761-cf14f5c906f1")
	if first == RunFactID(other) {
		t.Fatal("run fact id collided")
	}
}

func TestStreamNameKeepsTheJobCategory(t *testing.T) {
	id := uuid.MustParse("9613ff77-bef0-5b9a-9925-123e82aca8d1")
	name := StreamName(id)
	if name[StreamIDOffset()-1:] != id.String() {
		t.Fatalf("stream = %s", name)
	}
	got, err := JobIDFromStream(name)
	if err != nil || got != id {
		t.Fatalf("job = %s err=%v", got, err)
	}
}

func TestSubmitAndStartFactIDsStayStable(t *testing.T) {
	jobID := uuid.MustParse("9613ff77-bef0-5b9a-9925-123e82aca8d1")
	if SubmitFactID(jobID) != SubmitFactID(jobID) || StartFactID(jobID) != StartFactID(jobID) {
		t.Fatal("fact id changed")
	}
	if SubmitFactID(jobID) == StartFactID(jobID) || SubmitFactID(jobID) == RunFactID(jobID) {
		t.Fatal("fact ids collided")
	}
	if WorkflowID(jobID) != "cloudchamber-job-"+jobID.String() {
		t.Fatal("workflow id")
	}
}

func TestFoldUsesTheEndingWhereverItSits(t *testing.T) {
	jobID := uuid.MustParse("9613ff77-bef0-5b9a-9925-123e82aca8d1")
	owner := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	project := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	input := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	audio := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	document := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	created := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	hash := "abc"

	facts := []Fact{
		submitted(owner, project, input, &hash, created),
		started("run-1", created.Add(time.Minute)),
		audioPublished(audio, time.Time{}),
		documentPublished(document, created.Add(2*time.Minute)),
	}
	open := Fold(jobID, facts)
	if open.Status.String() != StatusRunning || open.RunID != "run-1" || open.DocumentFileID != document || open.AudioFileID != audio {
		t.Fatalf("open job = %+v", open)
	}
	if open.FailureMessage != nil || open.OwnerID != owner || open.InputFileID != input || open.DateCreated != created {
		t.Fatalf("identity = %+v", open)
	}
	if open.IdempotencyKeyHash == nil || *open.IdempotencyKeyHash != hash || open.WorkflowID != WorkflowID(jobID) {
		t.Fatalf("identity = %+v", open)
	}

	ended := Fold(jobID, append(facts,
		failed(new("source missing"), created.Add(3*time.Minute)),
		started("run-1", created.Add(4*time.Minute)),
	))
	if ended.Status.String() != StatusFailed || ended.FailureMessage == nil || *ended.FailureMessage != "source missing" || ended.RunID != "run-1" {
		t.Fatalf("ended job = %+v", ended)
	}

	blank := Fold(jobID, []Fact{failed(nil, time.Time{})})
	if blank.Status.String() != StatusFailed || blank.FailureMessage == nil || *blank.FailureMessage != StatusFailed {
		t.Fatalf("blank failure = %+v", blank)
	}

	done := Fold(jobID, append(facts, closed(StatusCompleted, time.Time{})))
	if done.Status.String() != StatusCompleted || done.FailureMessage != nil || done.DocumentFileID != document {
		t.Fatalf("completed job = %+v", done)
	}

	// The last fact owns the updated time when its clock is earlier than an older fact.
	// The earlier start remains the cause of the run.
	reordered := Fold(jobID, []Fact{
		started("run-1", created.Add(time.Hour)),
		submitted(owner, project, input, &hash, created),
	})
	if reordered.Status.String() != StatusRunning || reordered.RunID != "run-1" || !reordered.DateUpdated.Equal(created) || !reordered.DateCreated.Equal(created) {
		t.Fatalf("reordered = %+v", reordered)
	}

	spaced := "  "
	kept := Fold(jobID, []Fact{{Cause: RunFailed{Note: &spaced}}})
	if kept.FailureMessage == nil || *kept.FailureMessage != StatusFailed {
		t.Fatalf("spaced failure = %+v", kept.FailureMessage)
	}
	nbsp := "\u00a0\u2003"
	blankNote := Fold(jobID, []Fact{{Cause: RunFailed{Note: &nbsp}}})
	if blankNote.FailureMessage == nil || *blankNote.FailureMessage != StatusFailed {
		t.Fatalf("unicode space failure = %+v", blankNote.FailureMessage)
	}
	keptNote := "\u00a0boom"
	keptText := Fold(jobID, []Fact{{Cause: RunFailed{Note: &keptNote}}})
	if keptText.FailureMessage == nil || *keptText.FailureMessage != keptNote {
		t.Fatalf("kept note = %+v", keptText.FailureMessage)
	}
	empty := ""
	blankText := Fold(jobID, []Fact{failed(&empty, time.Time{})})
	if blankText.FailureMessage == nil || *blankText.FailureMessage != StatusFailed {
		t.Fatalf("empty failure = %+v", blankText.FailureMessage)
	}

	keptEnding := Fold(jobID, []Fact{
		failed(new("boom"), time.Time{}),
		closed(StatusCanceled, time.Time{}),
	})
	if keptEnding.Status.String() != StatusFailed || keptEnding.FailureMessage == nil || *keptEnding.FailureMessage != "boom" {
		t.Fatalf("kept ending = %+v", keptEnding)
	}

	secondStart := Fold(jobID, []Fact{
		started("run-1", time.Time{}),
		started("run-2", time.Time{}),
	})
	if secondStart.RunID != "run-1" || secondStart.Status.String() != StatusRunning {
		t.Fatalf("second start = %+v", secondStart)
	}

	secondOwner := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	secondSubmit := Fold(jobID, []Fact{
		submitted(owner, project, input, nil, time.Time{}),
		submitted(secondOwner, project, input, nil, time.Time{}),
	})
	if secondSubmit.OwnerID != owner {
		t.Fatalf("second submit = %+v", secondSubmit.OwnerID)
	}

	replacement := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	firstFile := Fold(jobID, []Fact{
		audioPublished(audio, time.Time{}),
		audioPublished(replacement, time.Time{}),
	})
	if firstFile.AudioFileID != audio {
		t.Fatalf("second file = %+v", firstFile.AudioFileID)
	}
}

func submitted(owner, project, input uuid.UUID, hash *string, at time.Time) Fact {
	return Fact{
		Time:  at,
		Cause: JobAccepted{OwnerID: owner, ProjectID: project, InputFileID: input, IdempotencyKeyHash: hash},
	}
}

func TestNoteSpaceMatchesUnicode(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if isNoteSpace(r) != unicode.IsSpace(r) {
			t.Fatalf("space mismatch at %U", r)
		}
	}
}

func started(runID string, at time.Time) Fact {
	return Fact{Time: at, Cause: RunStarted{RunID: runID}}
}

func failed(message *string, at time.Time) Fact {
	return Fact{Time: at, Cause: RunFailed{Note: note(message)}}
}

func closed(status string, at time.Time) Fact {
	var cause Cause
	switch status {
	case StatusCompleted:
		cause = RunCompleted{}
	case StatusCanceled:
		cause = RunCanceled{}
	default:
		cause = RunFailed{}
	}
	return Fact{Time: at, Cause: cause}
}

func audioPublished(id uuid.UUID, at time.Time) Fact {
	return Fact{Time: at, Cause: AudioPublished{FileID: id}}
}

func documentPublished(id uuid.UUID, at time.Time) Fact {
	return Fact{Time: at, Cause: DocumentPublished{FileID: id}}
}

func TestSentenceOfNamesTheStage(t *testing.T) {
	file := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	cases := []struct {
		cause   Cause
		stage   string
		outcome string
	}{
		{JobAccepted{}, "", ""},
		{RunStarted{RunID: "run-1"}, "", ""},
		{RunCompleted{}, "", ""},
		{RunFailed{}, "", ""},
		{RunCanceled{}, "", ""},
		{SourceResolved{}, StageSource, outcomeDone},
		{SourceFailed{}, StageSource, outcomeFailed},
		{SourceCanceled{}, StageSource, outcomeCanceled},
		{AudioPublished{FileID: file}, StageAudio, outcomeDone},
		{AudioFailed{}, StageAudio, outcomeFailed},
		{AudioCanceled{}, StageAudio, outcomeCanceled},
		{TranscriptPublished{FileID: file}, StageTranscript, outcomeDone},
		{TranscriptFailed{}, StageTranscript, outcomeFailed},
		{TranscriptCanceled{}, StageTranscript, outcomeCanceled},
		{KeyFramesPublished{FileID: file}, StageKeyFrames, outcomeDone},
		{KeyFramesFailed{}, StageKeyFrames, outcomeFailed},
		{KeyFramesCanceled{}, StageKeyFrames, outcomeCanceled},
		{KeyFramesSkipped{}, StageKeyFrames, outcomeSkipped},
		{DocumentPublished{FileID: file}, StageDocument, outcomeDone},
		{DocumentFailed{}, StageDocument, outcomeFailed},
		{DocumentCanceled{}, StageDocument, outcomeCanceled},
	}
	for _, tc := range cases {
		got := tc.cause.Sentence()
		if got.Stage.String() != tc.stage || got.Outcome.String() != tc.outcome {
			t.Fatalf("%T = %s %s", tc.cause, got.Stage, got.Outcome)
		}
	}
}
