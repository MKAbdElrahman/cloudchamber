// Package runworker is what a worker process adds so that its activities
// show up as the steps of a run. A workflow author writes plain Temporal code.
// The only call an activity may add is Progress, to say how far it has got.
package runworker

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/stepstate"
	"github.com/MKAbdElrahman/cloudchamber/foundation/logger"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
)

// StepInterceptor reports each activity of a run as a step: working when an
// attempt starts, and how it ended when the attempt returns. The step key is
// the activity type name. An activity of a workflow that is not a run is left alone.
//
// It also heartbeats for the activity. The engine can only tell a running
// activity to pause or to stop in the reply to a heartbeat, so an activity that
// never heartbeats cannot be stopped. With the interceptor's heartbeat, every
// step's context is canceled soon after a suspend or a cancel, whether or not
// its author thought about it. See WithoutHeartbeat for what that costs.
type StepInterceptor struct {
	interceptor.WorkerInterceptorBase
	log   *logger.Logger
	steps runbus.StepReporter
	pulse time.Duration
}

// defaultPulse is how often the interceptor heartbeats for an activity.
const defaultPulse = time.Second

// NewStepInterceptor constructs the interceptor. Pass it in worker.Options.Interceptors.
func NewStepInterceptor(log *logger.Logger, steps runbus.StepReporter) *StepInterceptor {
	return &StepInterceptor{log: log, steps: steps, pulse: defaultPulse}
}

// WithoutHeartbeat returns an interceptor that reports steps and leaves
// heartbeating to each activity.
//
// The interceptor's heartbeat comes from a timer, not from the work, so it
// keeps arriving when the work has hung. A worker whose activities must be
// declared dead by a missed heartbeat should use this and heartbeat from the work.
func (s *StepInterceptor) WithoutHeartbeat() *StepInterceptor {
	return &StepInterceptor{log: s.log, steps: s.steps}
}

// InterceptActivity wraps every activity the worker runs.
func (s *StepInterceptor) InterceptActivity(_ context.Context, next interceptor.ActivityInboundInterceptor) interceptor.ActivityInboundInterceptor {
	inbound := &stepInbound{log: s.log, steps: s.steps, pulse: s.pulse}
	inbound.Next = next
	return inbound
}

type stepInbound struct {
	interceptor.ActivityInboundInterceptorBase
	log   *logger.Logger
	steps runbus.StepReporter
	pulse time.Duration
	beats *beatOutbound
}

// Init wraps the outbound side so the interceptor sees the activity's own heartbeats.
func (s *stepInbound) Init(outbound interceptor.ActivityOutboundInterceptor) error {
	s.beats = &beatOutbound{}
	s.beats.Next = outbound
	return s.Next.Init(s.beats)
}

// ExecuteActivity reports the step around the activity. A report that fails is
// logged and does not fail the work: the step record follows the work, never the reverse.
func (s *stepInbound) ExecuteActivity(ctx context.Context, in *interceptor.ExecuteActivityInput) (any, error) {
	info := activity.GetInfo(ctx)
	runID, ok := runbus.RunIDFromWorkflow(info.WorkflowExecution.ID)
	if !ok || info.IsLocalActivity {
		return s.Next.ExecuteActivity(ctx, in)
	}

	step := &stepProgress{log: s.log, steps: s.steps, runID: runID, key: info.ActivityType.Name}
	step.write(ctx, runbus.StepReport{Key: step.key, State: stepstate.Working})

	if s.pulse > 0 && s.beats != nil {
		stop := s.beats.keepBeating(ctx, s.pulse)
		defer stop()
	}

	result, err := s.Next.ExecuteActivity(context.WithValue(ctx, progressKey{}, step), in)

	report := outcome(ctx, info, err)
	report.Key = step.key
	if report.State != stepstate.Done {
		// A step that stopped keeps showing how far it got.
		report.Fraction, report.Measured = step.reached()
	}
	// The activity's context may already be canceled. The report still has to land.
	step.write(context.WithoutCancel(ctx), report)

	return result, err
}

// beatOutbound remembers the activity's own heartbeats and adds the interceptor's.
type beatOutbound struct {
	interceptor.ActivityOutboundInterceptorBase

	mu      sync.Mutex
	details []any
	own     bool
	last    time.Time
}

// RecordHeartbeat passes the activity's heartbeat on and remembers what it recorded.
func (b *beatOutbound) RecordHeartbeat(ctx context.Context, details ...any) {
	b.mu.Lock()
	b.details, b.own, b.last = details, true, time.Now()
	b.mu.Unlock()

	b.Next.RecordHeartbeat(ctx, details...)
}

// keepBeating heartbeats every pulse until stop is called or ctx ends.
//
// A heartbeat replaces the details the engine holds, and an activity keeps
// its place in those details. So the interceptor sends again what the
// activity last recorded. Until the activity has recorded anything, an attempt
// that started with details from an earlier attempt is left alone: sending
// nothing would erase the place it is about to continue from.
func (b *beatOutbound) keepBeating(ctx context.Context, pulse time.Duration) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup

	wg.Go(func() {
		ticker := time.NewTicker(pulse)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			b.mu.Lock()
			details, own, recent := b.details, b.own, time.Since(b.last) < pulse
			b.mu.Unlock()

			switch {
			case recent:
				// The activity is heartbeating on its own.
			case !own && b.Next.HasHeartbeatDetails(ctx):
				// Its place from an earlier attempt is still the only record of it.
			default:
				b.Next.RecordHeartbeat(ctx, details...)
			}
		}
	})

	return func() {
		close(done)
		wg.Wait()
	}
}

// outcome names the state an attempt left its step in.
// A failure that the engine will retry leaves the step working.
func outcome(ctx context.Context, info activity.Info, err error) runbus.StepReport {
	switch {
	case err == nil:
		return runbus.StepReport{State: stepstate.Done}
	case errors.Is(context.Cause(ctx), activity.ErrActivityPaused):
		return runbus.StepReport{State: stepstate.Paused}
	case errors.Is(ctx.Err(), context.Canceled), temporal.IsCanceledError(err):
		return runbus.StepReport{State: stepstate.Canceled}
	case final(info, err):
		return runbus.StepReport{State: stepstate.Failed, Message: failureText(err)}
	default:
		return runbus.StepReport{State: stepstate.Working, Message: "Trying again. " + failureText(err)}
	}
}

// final reports whether the engine will not retry this failure.
func final(info activity.Info, err error) bool {
	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok && appErr.NonRetryable() {
		return true
	}
	policy := info.RetryPolicy
	return policy != nil && policy.MaximumAttempts > 0 && info.Attempt >= policy.MaximumAttempts
}

func failureText(err error) string {
	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok {
		if msg := strings.TrimSpace(appErr.Message()); msg != "" {
			return msg
		}
	}
	return strings.TrimSpace(err.Error())
}

// =============================================================================

type progressKey struct{}

// progressEvery is the least time between two stored progress reports that
// differ only in how far the step has got.
const progressEvery = time.Second

// stepProgress is the live progress of one activity attempt.
type stepProgress struct {
	log   *logger.Logger
	steps runbus.StepReporter
	runID uuid.UUID
	key   string

	mu       sync.Mutex
	message  string
	fraction float64
	measured bool
	percent  int
	stored   time.Time
}

// Progress says how far the step the activity is running has got, and in a
// few words what it is doing. Pass a fraction from 0 to 1, or a negative number
// when the amount is not known.
//
// Call it as often as is convenient. The step record is overwritten in place,
// at most once a second unless the message changes, so a long step adds no rows.
// Outside a run's activity it does nothing.
func Progress(ctx context.Context, fraction float64, message string) {
	step, ok := ctx.Value(progressKey{}).(*stepProgress)
	if !ok {
		return
	}
	step.report(ctx, fraction, message)
}

func (p *stepProgress) report(ctx context.Context, fraction float64, message string) {
	measured := fraction >= 0
	fraction = min(max(fraction, 0), 1)
	percent := int(math.Round(fraction * 100))

	p.mu.Lock()
	changedMessage := message != p.message
	changedAmount := measured != p.measured || percent != p.percent
	due := time.Since(p.stored) >= progressEvery
	p.message, p.fraction, p.measured, p.percent = message, fraction, measured, percent
	store := changedMessage || (changedAmount && due)
	if store {
		p.stored = time.Now()
	}
	p.mu.Unlock()

	if !store || ctx.Err() != nil {
		return
	}
	p.write(ctx, runbus.StepReport{Key: p.key, State: stepstate.Working, Message: message, Fraction: fraction, Measured: measured})
}

func (p *stepProgress) reached() (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fraction, p.measured
}

func (p *stepProgress) write(ctx context.Context, report runbus.StepReport) {
	if err := p.steps.ReportStep(ctx, p.runID, report); err != nil {
		p.log.Error(ctx, "runworker.step", "run_id", p.runID, "step", p.key, "err", err)
	}
}
