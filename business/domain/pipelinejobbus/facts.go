package pipelinejobbus

import (
	"fmt"

	"github.com/google/uuid"
)

// Cause is one business event in the job log. The fold reads every cause.
// A live fraction and headline are a Progress update, not a cause.
// Name is the JSON type tag. Sentence is the stage and outcome the page renders.
// FailureNote is the failure text. PublishedFile is the file a published sentence owns.
type Cause interface {
	isCause()
	Name() string
	Sentence() Sentence
	FailureNote() *string
	PublishedFile() uuid.UUID
}

// JobAccepted is the accepted fact. The first one owns the job.
type JobAccepted struct {
	OwnerID            uuid.UUID
	ProjectID          uuid.UUID
	InputFileID        uuid.UUID
	IdempotencyKeyHash *string
}

// RunStarted records the run id. The first one owns the run.
type RunStarted struct {
	RunID string
}

// RunCompleted closes the run as completed.
type RunCompleted struct{}

// RunFailed closes the run. A blank note is no note.
type RunFailed struct {
	Note *string
}

// RunCanceled closes the run as canceled.
type RunCanceled struct{}

// SourceResolved says the source finished. It publishes no file.
type SourceResolved struct{}

// SourceFailed says resolving the source stopped.
type SourceFailed struct {
	Note *string
}

// SourceCanceled says resolving the source was canceled.
type SourceCanceled struct{}

// AudioPublished is the extracted audio file. The first one owns the column.
type AudioPublished struct {
	FileID uuid.UUID
}

// AudioFailed says audio extraction stopped.
type AudioFailed struct {
	Note *string
}

// AudioCanceled says audio extraction was canceled.
type AudioCanceled struct{}

// TranscriptPublished is the transcript file. The first one owns the column.
type TranscriptPublished struct {
	FileID uuid.UUID
}

// TranscriptFailed says transcription stopped.
type TranscriptFailed struct {
	Note *string
}

// TranscriptCanceled says transcription was canceled.
type TranscriptCanceled struct{}

// KeyFramesPublished is the key-frame file. The first one owns the column.
type KeyFramesPublished struct {
	FileID uuid.UUID
}

// KeyFramesFailed says key-frame extraction stopped.
type KeyFramesFailed struct {
	Note *string
}

// KeyFramesCanceled says key-frame extraction was canceled.
type KeyFramesCanceled struct{}

// KeyFramesSkipped says this recording has no frames.
type KeyFramesSkipped struct{}

// DocumentPublished is the composed document. The first one owns the column.
type DocumentPublished struct {
	FileID uuid.UUID
}

// DocumentFailed says composition stopped.
type DocumentFailed struct {
	Note *string
}

// DocumentCanceled says composition was canceled.
type DocumentCanceled struct{}

func (JobAccepted) isCause()         {}
func (RunStarted) isCause()          {}
func (RunCompleted) isCause()        {}
func (RunFailed) isCause()           {}
func (RunCanceled) isCause()         {}
func (SourceResolved) isCause()      {}
func (SourceFailed) isCause()        {}
func (SourceCanceled) isCause()      {}
func (AudioPublished) isCause()      {}
func (AudioFailed) isCause()         {}
func (AudioCanceled) isCause()       {}
func (TranscriptPublished) isCause() {}
func (TranscriptFailed) isCause()    {}
func (TranscriptCanceled) isCause()  {}
func (KeyFramesPublished) isCause()  {}
func (KeyFramesFailed) isCause()     {}
func (KeyFramesCanceled) isCause()   {}
func (KeyFramesSkipped) isCause()    {}
func (DocumentPublished) isCause()   {}
func (DocumentFailed) isCause()      {}
func (DocumentCanceled) isCause()    {}

func (JobAccepted) Name() string         { return FactJobAccepted }
func (RunStarted) Name() string          { return FactRunStarted }
func (RunCompleted) Name() string        { return FactRunCompleted }
func (RunFailed) Name() string           { return FactRunFailed }
func (RunCanceled) Name() string         { return FactRunCanceled }
func (SourceResolved) Name() string      { return FactSourceResolved }
func (SourceFailed) Name() string        { return FactSourceFailed }
func (SourceCanceled) Name() string      { return FactSourceCanceled }
func (AudioPublished) Name() string      { return FactAudioPublished }
func (AudioFailed) Name() string         { return FactAudioFailed }
func (AudioCanceled) Name() string       { return FactAudioCanceled }
func (TranscriptPublished) Name() string { return FactTranscriptPublished }
func (TranscriptFailed) Name() string    { return FactTranscriptFailed }
func (TranscriptCanceled) Name() string  { return FactTranscriptCanceled }
func (KeyFramesPublished) Name() string  { return FactKeyFramesPublished }
func (KeyFramesFailed) Name() string     { return FactKeyFramesFailed }
func (KeyFramesCanceled) Name() string   { return FactKeyFramesCanceled }
func (KeyFramesSkipped) Name() string    { return FactKeyFramesSkipped }
func (DocumentPublished) Name() string   { return FactDocumentPublished }
func (DocumentFailed) Name() string      { return FactDocumentFailed }
func (DocumentCanceled) Name() string    { return FactDocumentCanceled }

func (JobAccepted) Sentence() Sentence    { return Sentence{} }
func (RunStarted) Sentence() Sentence     { return Sentence{} }
func (RunCompleted) Sentence() Sentence   { return Sentence{} }
func (RunFailed) Sentence() Sentence      { return Sentence{} }
func (RunCanceled) Sentence() Sentence    { return Sentence{} }
func (SourceResolved) Sentence() Sentence { return Sentence{Stage: SourceStage, Outcome: DoneOutcome} }
func (SourceFailed) Sentence() Sentence   { return Sentence{Stage: SourceStage, Outcome: FailedOutcome} }
func (SourceCanceled) Sentence() Sentence {
	return Sentence{Stage: SourceStage, Outcome: CanceledOutcome}
}
func (AudioPublished) Sentence() Sentence { return Sentence{Stage: AudioStage, Outcome: DoneOutcome} }
func (AudioFailed) Sentence() Sentence    { return Sentence{Stage: AudioStage, Outcome: FailedOutcome} }
func (AudioCanceled) Sentence() Sentence {
	return Sentence{Stage: AudioStage, Outcome: CanceledOutcome}
}
func (TranscriptPublished) Sentence() Sentence {
	return Sentence{Stage: TranscriptStage, Outcome: DoneOutcome}
}
func (TranscriptFailed) Sentence() Sentence {
	return Sentence{Stage: TranscriptStage, Outcome: FailedOutcome}
}
func (TranscriptCanceled) Sentence() Sentence {
	return Sentence{Stage: TranscriptStage, Outcome: CanceledOutcome}
}
func (KeyFramesPublished) Sentence() Sentence {
	return Sentence{Stage: KeyFramesStage, Outcome: DoneOutcome}
}
func (KeyFramesFailed) Sentence() Sentence {
	return Sentence{Stage: KeyFramesStage, Outcome: FailedOutcome}
}
func (KeyFramesCanceled) Sentence() Sentence {
	return Sentence{Stage: KeyFramesStage, Outcome: CanceledOutcome}
}
func (KeyFramesSkipped) Sentence() Sentence {
	return Sentence{Stage: KeyFramesStage, Outcome: SkippedOutcome}
}
func (DocumentPublished) Sentence() Sentence {
	return Sentence{Stage: DocumentStage, Outcome: DoneOutcome}
}
func (DocumentFailed) Sentence() Sentence {
	return Sentence{Stage: DocumentStage, Outcome: FailedOutcome}
}
func (DocumentCanceled) Sentence() Sentence {
	return Sentence{Stage: DocumentStage, Outcome: CanceledOutcome}
}

func (JobAccepted) FailureNote() *string         { return nil }
func (RunStarted) FailureNote() *string          { return nil }
func (RunCompleted) FailureNote() *string        { return nil }
func (c RunFailed) FailureNote() *string         { return c.Note }
func (RunCanceled) FailureNote() *string         { return nil }
func (SourceResolved) FailureNote() *string      { return nil }
func (c SourceFailed) FailureNote() *string      { return c.Note }
func (SourceCanceled) FailureNote() *string      { return nil }
func (AudioPublished) FailureNote() *string      { return nil }
func (c AudioFailed) FailureNote() *string       { return c.Note }
func (AudioCanceled) FailureNote() *string       { return nil }
func (TranscriptPublished) FailureNote() *string { return nil }
func (c TranscriptFailed) FailureNote() *string  { return c.Note }
func (TranscriptCanceled) FailureNote() *string  { return nil }
func (KeyFramesPublished) FailureNote() *string  { return nil }
func (c KeyFramesFailed) FailureNote() *string   { return c.Note }
func (KeyFramesCanceled) FailureNote() *string   { return nil }
func (KeyFramesSkipped) FailureNote() *string    { return nil }
func (DocumentPublished) FailureNote() *string   { return nil }
func (c DocumentFailed) FailureNote() *string    { return c.Note }
func (DocumentCanceled) FailureNote() *string    { return nil }

func (JobAccepted) PublishedFile() uuid.UUID      { return uuid.Nil }
func (RunStarted) PublishedFile() uuid.UUID       { return uuid.Nil }
func (RunCompleted) PublishedFile() uuid.UUID     { return uuid.Nil }
func (RunFailed) PublishedFile() uuid.UUID        { return uuid.Nil }
func (RunCanceled) PublishedFile() uuid.UUID      { return uuid.Nil }
func (SourceResolved) PublishedFile() uuid.UUID   { return uuid.Nil }
func (SourceFailed) PublishedFile() uuid.UUID     { return uuid.Nil }
func (SourceCanceled) PublishedFile() uuid.UUID   { return uuid.Nil }
func (c AudioPublished) PublishedFile() uuid.UUID { return c.FileID }
func (AudioFailed) PublishedFile() uuid.UUID      { return uuid.Nil }
func (AudioCanceled) PublishedFile() uuid.UUID    { return uuid.Nil }
func (c TranscriptPublished) PublishedFile() uuid.UUID {
	return c.FileID
}
func (TranscriptFailed) PublishedFile() uuid.UUID     { return uuid.Nil }
func (TranscriptCanceled) PublishedFile() uuid.UUID   { return uuid.Nil }
func (c KeyFramesPublished) PublishedFile() uuid.UUID { return c.FileID }
func (KeyFramesFailed) PublishedFile() uuid.UUID      { return uuid.Nil }
func (KeyFramesCanceled) PublishedFile() uuid.UUID    { return uuid.Nil }
func (KeyFramesSkipped) PublishedFile() uuid.UUID     { return uuid.Nil }
func (c DocumentPublished) PublishedFile() uuid.UUID  { return c.FileID }
func (DocumentFailed) PublishedFile() uuid.UUID       { return uuid.Nil }
func (DocumentCanceled) PublishedFile() uuid.UUID     { return uuid.Nil }

const (
	FactJobAccepted         = "jobAccepted"
	FactRunStarted          = "runStarted"
	FactRunCompleted        = "runCompleted"
	FactRunFailed           = "runFailed"
	FactRunCanceled         = "runCanceled"
	FactSourceResolved      = "sourceResolved"
	FactSourceFailed        = "sourceFailed"
	FactSourceCanceled      = "sourceCanceled"
	FactAudioPublished      = "audioPublished"
	FactAudioFailed         = "audioFailed"
	FactAudioCanceled       = "audioCanceled"
	FactTranscriptPublished = "transcriptPublished"
	FactTranscriptFailed    = "transcriptFailed"
	FactTranscriptCanceled  = "transcriptCanceled"
	FactKeyFramesPublished  = "keyFramesPublished"
	FactKeyFramesFailed     = "keyFramesFailed"
	FactKeyFramesCanceled   = "keyFramesCanceled"
	FactKeyFramesSkipped    = "keyFramesSkipped"
	FactDocumentPublished   = "documentPublished"
	FactDocumentFailed      = "documentFailed"
	FactDocumentCanceled    = "documentCanceled"
)

// Ending is the one terminal sentence of a run.
// The workflow sends the name. A blank note is no note.
func Ending(name string, message *string) (Cause, error) {
	switch name {
	case FactRunCompleted:
		return RunCompleted{}, nil
	case FactRunFailed:
		return RunFailed{Note: note(message)}, nil
	case FactRunCanceled:
		return RunCanceled{}, nil
	default:
		return nil, fmt.Errorf("fact %s", name)
	}
}

// Sentence is the stage and outcome the page renders for one cause.
// A run sentence has an empty stage and an empty outcome.
type Sentence struct {
	Stage   Stage
	Outcome Outcome
}

// Outcome is how a step sentence sits on its stage.
type Outcome struct {
	value string
}

const (
	outcomeDone     = "done"
	outcomeFailed   = "failed"
	outcomeCanceled = "canceled"
	outcomeSkipped  = "skipped"
)

// The outcomes a step sentence can have.
var (
	DoneOutcome     = Outcome{value: outcomeDone}
	FailedOutcome   = Outcome{value: outcomeFailed}
	CanceledOutcome = Outcome{value: outcomeCanceled}
	SkippedOutcome  = Outcome{value: outcomeSkipped}
)

// String returns the outcome name.
func (o Outcome) String() string {
	return o.value
}

// isNoteSpace is Go's unicode.IsSpace set: the Unicode White_Space property.
// The proof uses the same code points.
func isNoteSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ',
		'\u0085', '\u00a0', '\u1680',
		'\u2000', '\u2001', '\u2002', '\u2003', '\u2004',
		'\u2005', '\u2006', '\u2007', '\u2008', '\u2009', '\u200a',
		'\u2028', '\u2029', '\u202f', '\u205f', '\u3000':
		return true
	default:
		return false
	}
}

// CleanNote drops a blank note. Blank is empty or only Unicode White_Space.
func CleanNote(message *string) *string {
	return note(message)
}

// note keeps text that contains a character outside isNoteSpace.
func note(message *string) *string {
	if message == nil {
		return nil
	}
	for _, r := range *message {
		if !isNoteSpace(r) {
			return message
		}
	}
	return nil
}
