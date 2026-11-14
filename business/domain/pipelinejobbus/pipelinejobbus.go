// Package pipelinejobbus provides business access to pipeline job domain.
package pipelinejobbus

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/MKAbdElrahman/cloudchamber/business/sdk/order"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/page"
	"github.com/MKAbdElrahman/cloudchamber/business/sdk/sqldb"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
)

// Set of error variables for CRUD operations.
var (
	ErrNotFound = errors.New("pipeline job not found")
)

// Storer interface declares the behavior this package needs to persist and
// retrieve data.
type Storer interface {
	NewWithTx(tx sqldb.CommitRollbacker) (Storer, error)
	Begin(ctx context.Context) (sqldb.CommitRollbacker, error)
	InTx() bool
	Create(ctx context.Context, job Job) error
	Update(ctx context.Context, job Job) error
	Delete(ctx context.Context, job Job) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error)
	AppendFact(ctx context.Context, job Job, fact Fact) (Fact, error)
	AppendStableFact(ctx context.Context, job Job, fact Fact) (Fact, error)
	AppendProgress(ctx context.Context, job Job, progress Progress) (Progress, error)
	StreamFacts(ctx context.Context, jobID uuid.UUID, after int64, limit int) ([]Fact, error)
	StreamLog(ctx context.Context, jobID uuid.UUID, after int64, limit int) (Log, error)
	ProjectFacts(ctx context.Context, filter QueryFilter, after int64, limit int) ([]Fact, error)
	ProjectLog(ctx context.Context, filter QueryFilter, after int64, limit int) (Log, error)
	ProjectCursor(ctx context.Context, filter QueryFilter) (int64, error)
	HeadFacts(ctx context.Context, jobIDs []uuid.UUID) ([]Fact, error)
	HeadProgress(ctx context.Context, jobIDs []uuid.UUID) ([]Progress, error)
}

// ExtBusiness interface provides support for extensions that wrap extra functionality
// around the core business logic.
type ExtBusiness interface {
	// NewWithTx returns a transaction-bound copy. Extensions must bind their
	// dependencies to the transaction and wrap the returned business themselves.
	NewWithTx(tx sqldb.CommitRollbacker) (ExtBusiness, error)
	Submit(ctx context.Context, nj NewJob) (Job, error)
	RecordStarted(ctx context.Context, job Job, runID string) (Job, error)
	Delete(ctx context.Context, job Job) error
	Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error)
	Count(ctx context.Context, filter QueryFilter) (int, error)
	QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error)
	Record(ctx context.Context, job Job, cause Cause) (Job, error)
	RecordProgress(ctx context.Context, job Job, stage Stage, fraction *float64, headline *string) error
	RecordRun(ctx context.Context, job Job, cause Cause) (Job, error)
	StreamFacts(ctx context.Context, jobID uuid.UUID, after int64, limit int) ([]Fact, error)
	StreamLog(ctx context.Context, jobID uuid.UUID, after int64, limit int) (Log, error)
	ProjectFacts(ctx context.Context, filter QueryFilter, after int64, limit int) ([]Fact, error)
	ProjectLog(ctx context.Context, filter QueryFilter, after int64, limit int) (Log, error)
	ProjectCursor(ctx context.Context, filter QueryFilter) (int64, error)
	AttachHeadFacts(ctx context.Context, jobs []Job) ([]Job, error)
}

// Extension is a function that wraps a new layer of business logic
// around the existing business logic.
type Extension func(ExtBusiness) ExtBusiness

// Business manages the set of APIs for pipeline job access.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs a pipeline job business API for use.
func NewBusiness(log *logger.Logger, storer Storer, extensions ...Extension) ExtBusiness {
	b := newBusiness(log, storer)

	extBus := ExtBusiness(b)

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

	// Each extension rebuilds its own wrapper in NewWithTx.
	return newBusiness(b.log, storer), nil
}

// Delete removes the specified pipeline job.
func (b *Business) Delete(ctx context.Context, job Job) error {
	if err := b.storer.Delete(ctx, job); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// Query retrieves a list of existing pipeline jobs.
func (b *Business) Query(ctx context.Context, filter QueryFilter, orderBy order.By, page page.Page) ([]Job, error) {
	jobs, err := b.storer.Query(ctx, filter, orderBy, page)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return jobs, nil
}

// Count returns the total number of pipeline jobs.
func (b *Business) Count(ctx context.Context, filter QueryFilter) (int, error) {
	return b.storer.Count(ctx, filter)
}

// QueryByID finds the pipeline job by the specified ID.
// Facts are not loaded. The caller reads the stream when it needs the log.
func (b *Business) QueryByID(ctx context.Context, jobID uuid.UUID) (Job, error) {
	job, err := b.storer.QueryByID(ctx, jobID)
	if err != nil {
		return Job{}, fmt.Errorf("query: jobID[%s]: %w", jobID, err)
	}
	return job, nil
}

// Submit appends the accepted fact and inserts its projection.
func (b *Business) Submit(ctx context.Context, nj NewJob) (Job, error) {
	fact := Fact{
		ID:    SubmitFactID(nj.ID),
		JobID: nj.ID,
		Cause: JobAccepted{
			OwnerID:            nj.OwnerID,
			ProjectID:          nj.ProjectID,
			InputFileID:        nj.InputFileID,
			IdempotencyKeyHash: nj.IdempotencyKeyHash,
		},
	}
	job := Job{ID: nj.ID, OwnerID: nj.OwnerID, ProjectID: nj.ProjectID, WorkflowID: WorkflowID(nj.ID)}
	stored, err := b.append(ctx, job, fact, true)
	if err != nil {
		return Job{}, fmt.Errorf("submit: %w", err)
	}
	return stored, nil
}

// RecordStarted appends the run id. An ending already in the stream keeps the status.
func (b *Business) RecordStarted(ctx context.Context, job Job, runID string) (Job, error) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return Job{}, fmt.Errorf("run id is required")
	}
	fact := Fact{
		ID:    StartFactID(job.ID),
		JobID: job.ID,
		Cause: RunStarted{RunID: runID},
	}
	stored, err := b.append(ctx, job, fact, true)
	if err != nil {
		return Job{}, fmt.Errorf("recordstarted: %w", err)
	}
	return stored, nil
}

// Record appends one step sentence. The previous facts stay in the stream.
// A fact about the whole run goes through RecordRun.
func (b *Business) Record(ctx context.Context, job Job, cause Cause) (Job, error) {
	if cause == nil {
		return Job{}, fmt.Errorf("fact is required")
	}
	fact := Fact{
		ID:    uuid.New(),
		JobID: job.ID,
		Cause: cause,
	}
	stored, err := b.append(ctx, job, fact, false)
	if err != nil {
		return Job{}, fmt.Errorf("record: %w", err)
	}
	return stored, nil
}

// RecordProgress appends one progress update. The job row stays as it was.
func (b *Business) RecordProgress(ctx context.Context, job Job, stage Stage, fraction *float64, headline *string) error {
	if stage.String() == "" {
		return fmt.Errorf("stage is required")
	}
	progress := Progress{
		ID:       uuid.New(),
		JobID:    job.ID,
		Stage:    stage,
		Fraction: fraction,
		Headline: headline,
	}
	if _, err := b.storer.AppendProgress(ctx, job, progress); err != nil {
		return fmt.Errorf("progress: %w", err)
	}
	return nil
}

// runFactNamespace is the UUID v5 namespace for the one terminal fact of a run.
// It stays fixed so a retry writes the same message id.
var runFactNamespace = uuid.MustParse("6f0e1d2c-3b4a-5968-8796-a5b4c3d2e1f0")

// RunFactID is the message id of the run-level fact for a job.
func RunFactID(jobID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(runFactNamespace, jobID[:])
}

// RecordRun records one ending sentence. A second ending leaves the first fact in place.
func (b *Business) RecordRun(ctx context.Context, job Job, cause Cause) (Job, error) {
	cause = normalizeEnding(cause)
	if !factIsEnding(Fact{Cause: cause}) {
		return Job{}, fmt.Errorf("ending fact is required")
	}
	fact := Fact{
		ID:    RunFactID(job.ID),
		JobID: job.ID,
		Cause: cause,
	}
	stored, err := b.append(ctx, job, fact, true)
	if err != nil {
		return Job{}, fmt.Errorf("recordrun: %w", err)
	}
	return stored, nil
}

// append writes the fact and the folded row in one transaction.
func (b *Business) append(ctx context.Context, job Job, fact Fact, stable bool) (Job, error) {
	if b.storer.InTx() {
		return b.appendLocked(ctx, job, fact, stable)
	}

	tx, err := b.storer.Begin(ctx)
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	txBus, err := b.NewWithTx(tx)
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	inner, ok := txBus.(*Business)
	if !ok {
		return Job{}, fmt.Errorf("append: transaction bus")
	}
	stored, err := inner.appendLocked(ctx, job, fact, stable)
	if err != nil {
		return Job{}, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	return stored, nil
}

func (b *Business) appendLocked(ctx context.Context, job Job, fact Fact, stable bool) (Job, error) {
	existing, err := b.readStream(ctx, job.ID)
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	if factIsEnding(fact) && hasEndingFact(existing) {
		return b.saveProjection(ctx, job.ID, existing)
	}

	if stable {
		_, err = b.storer.AppendStableFact(ctx, job, fact)
	} else {
		_, err = b.storer.AppendFact(ctx, job, fact)
	}
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	facts, err := b.readStream(ctx, job.ID)
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	return b.saveProjection(ctx, job.ID, facts)
}

func (b *Business) saveProjection(ctx context.Context, jobID uuid.UUID, facts []Fact) (Job, error) {
	if !hasAcceptedFact(facts) {
		return Job{}, fmt.Errorf("append: accepted fact is required")
	}
	folded := Fold(jobID, facts)
	_, err := b.storer.QueryByID(ctx, jobID)
	switch {
	case err == nil:
		err = b.storer.Update(ctx, folded)
	case errors.Is(err, ErrNotFound):
		err = b.storer.Create(ctx, folded)
	}
	if err != nil {
		return Job{}, fmt.Errorf("append: %w", err)
	}
	folded.Facts = facts
	return folded, nil
}

func (b *Business) readStream(ctx context.Context, jobID uuid.UUID) ([]Fact, error) {
	var all []Fact
	after := int64(-1)
	for {
		batch, err := b.storer.StreamFacts(ctx, jobID, after, 1000)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < 1000 {
			return all, nil
		}
		after = batch[len(batch)-1].Position
	}
}

// StreamFacts returns facts of one job after the given stream position.
func (b *Business) StreamFacts(ctx context.Context, jobID uuid.UUID, after int64, limit int) ([]Fact, error) {
	facts, err := b.storer.StreamFacts(ctx, jobID, after, limit)
	if err != nil {
		return nil, fmt.Errorf("query: jobID[%s]: %w", jobID, err)
	}
	return facts, nil
}

// StreamLog returns business events and progress updates after the given stream position.
func (b *Business) StreamLog(ctx context.Context, jobID uuid.UUID, after int64, limit int) (Log, error) {
	log, err := b.storer.StreamLog(ctx, jobID, after, limit)
	if err != nil {
		return Log{}, fmt.Errorf("query: jobID[%s]: %w", jobID, err)
	}
	return log, nil
}

// ProjectFacts returns facts of one project after the given global position.
func (b *Business) ProjectFacts(ctx context.Context, filter QueryFilter, after int64, limit int) ([]Fact, error) {
	if filter.ProjectID == nil {
		return nil, errors.New("project id is required")
	}
	facts, err := b.storer.ProjectFacts(ctx, filter, after, limit)
	if err != nil {
		return nil, fmt.Errorf("query: projectID[%s]: %w", *filter.ProjectID, err)
	}
	return facts, nil
}

// ProjectLog returns business events and progress updates after the given global position.
func (b *Business) ProjectLog(ctx context.Context, filter QueryFilter, after int64, limit int) (Log, error) {
	if filter.ProjectID == nil {
		return Log{}, errors.New("project id is required")
	}
	log, err := b.storer.ProjectLog(ctx, filter, after, limit)
	if err != nil {
		return Log{}, fmt.Errorf("query: projectID[%s]: %w", *filter.ProjectID, err)
	}
	return log, nil
}

// ProjectCursor returns the newest global position for a project.
func (b *Business) ProjectCursor(ctx context.Context, filter QueryFilter) (int64, error) {
	if filter.ProjectID == nil {
		return 0, errors.New("project id is required")
	}
	cursor, err := b.storer.ProjectCursor(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("query: projectID[%s]: %w", *filter.ProjectID, err)
	}
	return cursor, nil
}

// AttachHeadFacts sets the latest fact of each step on the jobs.
func (b *Business) AttachHeadFacts(ctx context.Context, jobs []Job) ([]Job, error) {
	if len(jobs) == 0 {
		return jobs, nil
	}
	ids := make([]uuid.UUID, len(jobs))
	for i, job := range jobs {
		ids[i] = job.ID
	}
	facts, err := b.storer.HeadFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	byJob := make(map[uuid.UUID][]Fact, len(jobs))
	for _, fact := range facts {
		byJob[fact.JobID] = append(byJob[fact.JobID], fact)
	}
	for i := range jobs {
		jobs[i].Facts = byJob[jobs[i].ID]
		slices.SortFunc(jobs[i].Facts, func(a, b Fact) int {
			return cmp.Compare(a.Position, b.Position)
		})
	}
	updates, err := b.storer.HeadProgress(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	byProgress := make(map[uuid.UUID][]Progress, len(jobs))
	for _, update := range updates {
		byProgress[update.JobID] = append(byProgress[update.JobID], update)
	}
	for i := range jobs {
		jobs[i].Progress = byProgress[jobs[i].ID]
		slices.SortFunc(jobs[i].Progress, func(a, b Progress) int {
			return cmp.Compare(a.Position, b.Position)
		})
	}
	return jobs, nil
}
