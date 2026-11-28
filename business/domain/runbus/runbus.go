// Package runbus provides business access to runs: long-running workflows a
// user starts, suspends, resumes, cancels, retries and restarts.
//
// A run is declarative. A user command edits the spec. The controller compares
// the spec with the engine and writes the status. Workers report their steps.
// No caller writes a part of the run it does not own.
package runbus

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
)

// Set of error variables for run operations.
var (
	ErrNotFound       = errors.New("run not found")
	ErrConflict       = errors.New("run was changed by another request")
	ErrIdempotencyKey = errors.New("a run with this idempotency key exists")
	ErrProject        = errors.New("the project does not exist or belongs to another owner")
)

// Storer interface declares the behavior this package needs to persist and
// retrieve data.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Create(ctx context.Context, run Run, event Event) error
	UpdateSpec(ctx context.Context, run Run, generation int64, event Event) error
	UpdateStatus(ctx context.Context, run Run, event Event) error
	UpsertStep(ctx context.Context, runID uuid.UUID, step Step) error
	ClearSteps(ctx context.Context, runID uuid.UUID, now time.Time) error
	Delete(ctx context.Context, run Run) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Run, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, runID uuid.UUID) (Run, error)
	QueryUnsettled(ctx context.Context, limit int) ([]Run, error)
	QueryEvents(ctx context.Context, runID uuid.UUID) ([]Event, error)
}

// Commander is what the API needs: create a run and edit its spec.
type Commander interface {
	Create(ctx context.Context, nr NewRun) (Run, error)
	Command(ctx context.Context, run Run, command Command, actorID uuid.UUID) (Run, error)
	Delete(ctx context.Context, run Run) error
}

// Reader is what any caller needs to look at runs.
type Reader interface {
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Run, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, runID uuid.UUID) (Run, error)
	QueryEvents(ctx context.Context, runID uuid.UUID) ([]Event, error)
}

// Observer is what the controller needs: find runs that need work and record what it saw.
type Observer interface {
	QueryByID(ctx context.Context, runID uuid.UUID) (Run, error)
	Unsettled(ctx context.Context, limit int) ([]Run, error)
	RecordStatus(ctx context.Context, run Run, status Status) (Run, error)
	ClearSteps(ctx context.Context, run Run) error
}

// StepReporter is what a worker needs: report the state of the step it runs.
type StepReporter interface {
	ReportStep(ctx context.Context, runID uuid.UUID, report StepReport) error
}

// ExtBusiness interface provides support for extensions that wrap extra functionality
// around the core business logic.
type ExtBusiness interface {
	NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error)
	Commander
	Reader
	Observer
	StepReporter
}

// Extension is a function that wraps a new layer of business logic
// around the existing business logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages the set of APIs for run access.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs a run business API for use.
func NewBusiness(log *logger.Logger, storer Storer, extensions ...Extension) ExtBusiness {
	extBus := ExtBusiness(newBusiness(log, storer))

	for _, ext := range slices.Backward(extensions) {
		if ext != nil {
			extBus = ext(extBus)
		}
	}

	return extBus
}

func newBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{
		log:    log,
		storer: storer,
	}
}

// NewWithTx constructs a new business value that will use the
// specified transaction in any store related calls.
func (b *Business) NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error) {
	storer, err := b.storer.NewWithTx(tx)
	if err != nil {
		return nil, err
	}

	return newBusiness(b.log, storer), nil
}

// Create records a run in the pending phase. The controller starts it.
func (b *Business) Create(ctx context.Context, nr NewRun) (Run, error) {
	now := time.Now()

	input := nr.Input
	if len(input) == 0 {
		input = []byte("{}")
	}

	run := Run{
		ID:                 nr.ID,
		Workflow:           nr.Workflow,
		OwnerID:            nr.OwnerID,
		ProjectID:          nr.ProjectID,
		IdempotencyKeyHash: nr.IdempotencyKeyHash,
		Generation:         1,
		Spec:               Spec{Input: input, Attempt: 1, ResumeFrom: resumefrom.Start},
		Status:             Status{Phase: phase.Pending},
		DateCreated:        now,
		DateUpdated:        now,
	}

	event := Event{RunID: run.ID, ActorID: run.OwnerID, Type: EventCreated, DateCreated: now}
	if err := b.storer.Create(ctx, run, event); err != nil {
		return Run{}, fmt.Errorf("create: %w", err)
	}

	return run, nil
}

// Command applies one user command to the run's spec. It fails with
// ErrNotAllowed when the command does not apply to the run's state, and with
// ErrConflict when another request changed the spec first. A command that
// asks for what the spec already says changes nothing.
func (b *Business) Command(ctx context.Context, run Run, command Command, actorID uuid.UUID) (Run, error) {
	spec, err := command.apply(run)
	if err != nil {
		return Run{}, fmt.Errorf("command: %s: %w", command.event, err)
	}
	if sameIntent(spec, run.Spec) {
		return run, nil
	}

	now := time.Now()
	expected := run.Generation
	run.Spec = spec
	run.Generation++
	run.DateUpdated = now

	event := Event{RunID: run.ID, ActorID: actorID, Type: command.event, DateCreated: now}
	if err := b.storer.UpdateSpec(ctx, run, expected, event); err != nil {
		return Run{}, fmt.Errorf("command: %s: %w", command.event, err)
	}

	return run, nil
}

// Delete removes the run and its events.
func (b *Business) Delete(ctx context.Context, run Run) error {
	if err := b.storer.Delete(ctx, run); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Query retrieves a list of existing runs.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Run, error) {
	runs, err := b.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return runs, nil
}

// Count returns the total number of runs.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.storer.Count(ctx, filter)
}

// QueryByID finds the run by the specified ID.
func (b *Business) QueryByID(ctx context.Context, runID uuid.UUID) (Run, error) {
	run, err := b.storer.QueryByID(ctx, runID)
	if err != nil {
		return Run{}, fmt.Errorf("query: runID[%s]: %w", runID, err)
	}
	return run, nil
}

// QueryEvents returns the run's events, oldest first.
func (b *Business) QueryEvents(ctx context.Context, runID uuid.UUID) ([]Event, error) {
	events, err := b.storer.QueryEvents(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("query: runID[%s]: %w", runID, err)
	}
	return events, nil
}

// Unsettled returns runs the controller still has work for, least recently updated first.
func (b *Business) Unsettled(ctx context.Context, limit int) ([]Run, error) {
	runs, err := b.storer.QueryUnsettled(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("unsettled: %w", err)
	}
	return runs, nil
}

// RecordStatus stores what the controller observed. A phase change is recorded as an event.
// The steps on the status are not written. The workers own them.
func (b *Business) RecordStatus(ctx context.Context, run Run, status Status) (Run, error) {
	now := time.Now()

	var event Event
	if status.Phase != run.Status.Phase {
		event = Event{RunID: run.ID, Type: EventPhaseChanged, Message: status.Phase.String(), DateCreated: now}
	}

	status.Steps = run.Status.Steps
	run.Status = status
	run.DateUpdated = now

	if err := b.storer.UpdateStatus(ctx, run, event); err != nil {
		return Run{}, fmt.Errorf("recordstatus: %w", err)
	}

	return run, nil
}

// ClearSteps forgets the steps of the attempt that closed, before a restart from the beginning.
func (b *Business) ClearSteps(ctx context.Context, run Run) error {
	if err := b.storer.ClearSteps(ctx, run.ID, time.Now()); err != nil {
		return fmt.Errorf("clearsteps: %w", err)
	}
	return nil
}

// ReportStep stores the state of one step, replacing what was stored for it.
// The first report of a step sets when it started.
func (b *Business) ReportStep(ctx context.Context, runID uuid.UUID, report StepReport) error {
	if report.Key == "" {
		return errors.New("reportstep: step key is required")
	}

	now := time.Now()
	step := Step{
		Key:         report.Key,
		State:       report.State,
		Message:     report.Message,
		Fraction:    min(max(report.Fraction, 0), 1),
		Measured:    report.Measured,
		DateStarted: now,
		DateUpdated: now,
	}
	if err := b.storer.UpsertStep(ctx, runID, step); err != nil {
		return fmt.Errorf("reportstep: %w", err)
	}

	return nil
}
