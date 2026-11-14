package commondb

import (
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/messagedb"
	"github.com/google/uuid"
)

func TestFactBodyOwnsTheSentence(t *testing.T) {
	jobID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	owner := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	project := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	input := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	hash := "hash"
	accepted := pipelinejobbus.Fact{
		ID:    uuid.MustParse("55555555-5555-5555-5555-555555555555"),
		JobID: jobID,
		Cause: pipelinejobbus.JobAccepted{
			OwnerID:            owner,
			ProjectID:          project,
			InputFileID:        input,
			IdempotencyKeyHash: &hash,
		},
	}
	got := roundTrip(t, accepted)
	cause, ok := got.Cause.(pipelinejobbus.JobAccepted)
	if !ok || cause.OwnerID != owner || cause.ProjectID != project || cause.InputFileID != input || cause.IdempotencyKeyHash == nil || *cause.IdempotencyKeyHash != hash {
		t.Fatalf("accepted = %+v", got.Cause)
	}

	started := pipelinejobbus.Fact{
		ID:    uuid.MustParse("66666666-6666-6666-6666-666666666666"),
		JobID: jobID,
		Cause: pipelinejobbus.RunStarted{RunID: "run-1"},
	}
	got = roundTrip(t, started)
	run, ok := got.Cause.(pipelinejobbus.RunStarted)
	if !ok || run.RunID != "run-1" {
		t.Fatalf("started = %+v", got.Cause)
	}
	audio := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	published := pipelinejobbus.Fact{
		ID:    uuid.MustParse("99999999-9999-9999-9999-999999999999"),
		JobID: jobID,
		Cause: pipelinejobbus.AudioPublished{FileID: audio},
	}
	got = roundTrip(t, published)
	file, ok := got.Cause.(pipelinejobbus.AudioPublished)
	if !ok || file.FileID != audio {
		t.Fatalf("published = %+v", got.Cause)
	}

	blank := "  "
	failed := pipelinejobbus.Fact{
		ID:    uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		JobID: jobID,
		Cause: pipelinejobbus.RunFailed{Note: &blank},
	}
	got = roundTrip(t, failed)
	runFailed, ok := got.Cause.(pipelinejobbus.RunFailed)
	if !ok || runFailed.Note != nil {
		t.Fatalf("blank note = %+v", got.Cause)
	}
}

func TestProgressBodyHasNoIdentity(t *testing.T) {
	jobID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	headline := "Writing the audio."
	fraction := 0.4
	update := pipelinejobbus.Progress{
		ID:       uuid.MustParse("77777777-7777-7777-7777-777777777777"),
		JobID:    jobID,
		Stage:    pipelinejobbus.AudioStage,
		Fraction: &fraction,
		Headline: &headline,
	}
	data, meta, err := ToDBProgress(update)
	if err != nil {
		t.Fatal(err)
	}
	if string(meta) != "{}" {
		t.Fatalf("metadata = %s", meta)
	}
	got, err := ToBusProgress(messagedb.Message{
		ID:         update.ID.String(),
		StreamName: pipelinejobbus.StreamName(jobID),
		Type:       pipelinejobbus.ProgressType,
		Data:       data,
		Metadata:   meta,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Stage != pipelinejobbus.AudioStage || got.Fraction == nil || *got.Fraction != fraction || got.Headline == nil || *got.Headline != headline {
		t.Fatalf("progress = %+v", got)
	}
}

func roundTrip(t *testing.T, fact pipelinejobbus.Fact) pipelinejobbus.Fact {
	t.Helper()
	data, meta, err := ToDBFact(fact)
	if err != nil {
		t.Fatal(err)
	}
	if string(meta) != "{}" {
		t.Fatalf("metadata = %s", meta)
	}
	got, err := ToBusFact(messagedb.Message{
		ID:         fact.ID.String(),
		StreamName: pipelinejobbus.StreamName(fact.JobID),
		Type:       pipelinejobbus.FactType,
		Data:       data,
		Metadata:   meta,
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}
