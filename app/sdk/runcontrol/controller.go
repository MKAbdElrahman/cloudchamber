// Package runcontrol moves each run toward what its spec asks for.
//
// The controller is level-triggered. For one run it reads the spec, asks the
// engine what is true, lets runbus.Decide pick one action, records the status,
// takes the action, and looks again. A nudge only says which run to look at.
// It carries no state, so a nudge that is lost, repeated or late does no harm.
// A periodic resync looks at every run that is not settled, so a missed nudge
// or a crash is repaired on the next pass.
//
// The shape follows a Kubernetes controller: a keyed work queue that never
// gives one run to two workers, a retry with backoff for a reconcile that
// fails, an immediate look for a change a person asked for, and a batched look
// for a change a worker reported.
package runcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
)

// Engine is the durable execution engine the controller drives.
type Engine interface {
	// Observe reports the latest execution of the run. Found is false when there is none.
	Observe(ctx context.Context, run runbus.Run) (runbus.Execution, error)
	// Apply takes one action on the run's execution.
	Apply(ctx context.Context, run runbus.Run, action runbus.Action) error
}

// Input is what every workflow receives.
// Input.Input is the workflow's own input, passed through as the caller sent it.
type Input struct {
	RunID     string          `json:"runId"`
	Workflow  string          `json:"workflow"`
	OwnerID   string          `json:"ownerId"`
	ProjectID string          `json:"projectId"`
	Input     json.RawMessage `json:"input"`
}

// Config holds what a controller needs.
type Config struct {
	Log    *logger.Logger
	Runs   runbus.Observer
	Engine Engine
	// Grace is how long a canceled run may take to stop before it is terminated.
	Grace time.Duration
	// Resync is how often every unsettled run is looked at again. It is a
	// safety net: changes normally arrive as nudges.
	Resync time.Duration
	// Workers is how many runs are reconciled at once.
	Workers int
}

// Controller reconciles runs.
type Controller struct {
	log     *logger.Logger
	runs    runbus.Observer
	engine  Engine
	grace   time.Duration
	resync  time.Duration
	workers int
	queue   *queue
}

const (
	// maxPasses bounds the actions taken for one run in one reconcile.
	// Suspend, for example, takes two: the execution, then its activities.
	maxPasses = 6
	// resyncBatch is how many unsettled runs one resync looks at.
	resyncBatch = 500
	// batchPeriod is how long a change reported by a worker waits before the
	// run is looked at, so several reports close together cost one look.
	// A change a person asked for is not delayed.
	batchPeriod = time.Second
	// transitionPoll is how soon a run in transition is looked at again when
	// no report arrives to say it has moved.
	transitionPoll = 2 * time.Second
)

// New constructs a controller.
func New(cfg Config) *Controller {
	if cfg.Grace <= 0 {
		cfg.Grace = time.Minute
	}
	if cfg.Resync <= 0 {
		cfg.Resync = 30 * time.Second
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 4
	}
	return &Controller{
		log:     cfg.Log,
		runs:    cfg.Runs,
		engine:  cfg.Engine,
		grace:   cfg.Grace,
		resync:  cfg.Resync,
		workers: cfg.Workers,
		queue:   newQueue(),
	}
}

// Nudge asks the controller to look at a run now. Use it for a change a
// person asked for. It never blocks, and a run nudged many times is looked at once.
func (c *Controller) Nudge(runID uuid.UUID) {
	c.queue.Add(runID)
}

// NudgeBatched asks the controller to look at a run shortly. Use it for a
// change a worker reported, so reports that arrive close together cost one look.
func (c *Controller) NudgeBatched(runID uuid.UUID) {
	c.queue.AddAfter(runID, batchPeriod)
}

// Resync queues every run that is not settled. Run calls it on a timer.
// Call it also after nudges may have been missed, such as when the connection
// that carries them was lost.
func (c *Controller) Resync(ctx context.Context) {
	runs, err := c.runs.Unsettled(ctx, resyncBatch)
	if err != nil {
		if ctx.Err() == nil {
			c.log.Error(ctx, "runcontrol.resync", "err", err)
		}
		return
	}
	for _, run := range runs {
		c.queue.Add(run.ID)
	}
}

// Run reconciles until ctx is canceled. Several workers share the queue, and
// the queue never hands one run to two of them at once. Run may be called
// again after it returns, as happens when this process loses the lead and
// takes it back. Runs nudged in between are kept.
func (c *Controller) Run(ctx context.Context) {
	var wg sync.WaitGroup
	for range c.workers {
		wg.Go(func() { c.work(ctx) })
	}
	c.Resync(ctx)
	ticker := time.NewTicker(c.resync)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return
		case <-ticker.C:
			c.Resync(ctx)
		}
	}
}

// work takes runs off the queue until it closes. A reconcile that fails is
// tried again after a wait that grows with each failure in a row.
func (c *Controller) work(ctx context.Context) {
	for {
		runID, ok := c.queue.Get(ctx)
		if !ok {
			return
		}

		err := c.Reconcile(ctx, runID)
		switch {
		case err == nil:
			c.queue.Forget(runID)
			c.followUp(ctx, runID)
		case ctx.Err() == nil:
			c.log.Error(ctx, "runcontrol.reconcile", "run_id", runID, "err", err)
			c.queue.Retry(runID)
		}

		c.queue.Done(runID)
	}
}

// followUp schedules the next look at a run that is in transition. A worker's
// report normally brings the run back sooner. This covers a step that reports
// nothing, and the moment a cancel's grace period ends.
func (c *Controller) followUp(ctx context.Context, runID uuid.UUID) {
	run, err := c.runs.QueryByID(ctx, runID)
	if err != nil || run.Converged() {
		return
	}

	wait := transitionPoll
	if at := run.Status.CancelRequestedAt; run.Spec.Cancel && !at.IsZero() {
		if left := time.Until(at.Add(c.grace)); left > 0 {
			wait = min(wait*5, left+100*time.Millisecond)
		}
	}
	c.queue.AddAfter(runID, wait)
}

// Reconcile moves one run as far as it can go now. It is safe to call at any
// time and any number of times.
func (c *Controller) Reconcile(ctx context.Context, runID uuid.UUID) error {
	for range maxPasses {
		run, err := c.runs.QueryByID(ctx, runID)
		if err != nil {
			if errors.Is(err, runbus.ErrNotFound) {
				return nil
			}
			return fmt.Errorf("query: %w", err)
		}

		exec, err := c.engine.Observe(ctx, run)
		if err != nil {
			return fmt.Errorf("observe: %w", err)
		}

		decision := runbus.Decide(run, exec, time.Now().UTC(), c.grace)

		if !decision.Status.Same(run.Status) {
			run, err = c.runs.RecordStatus(ctx, run, decision.Status)
			if err != nil {
				return fmt.Errorf("record status: %w", err)
			}
		}

		if decision.Action.Kind == runbus.ActionNone {
			return nil
		}

		// A restart from the beginning forgets the steps of the attempt that closed.
		if decision.Action.Kind == runbus.ActionStart && run.Status.Attempt > 0 {
			if err := c.runs.ClearSteps(ctx, run); err != nil {
				return fmt.Errorf("clear steps: %w", err)
			}
		}

		if err := c.engine.Apply(ctx, run, decision.Action); err != nil {
			return fmt.Errorf("apply action %d: %w", decision.Action.Kind, err)
		}
	}

	return nil
}
