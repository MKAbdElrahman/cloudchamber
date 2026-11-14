package sim

import (
	"context"
	"math"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

// stubRun is how long a simulated step stays running so the page can follow it.
const stubRun = 20 * time.Second

// factSample is the progress a step has measured. Has is false while the amount is unknown.
type factSample struct {
	Fraction float64
	Has      bool
	Message  string
}

// emit is how every worker records a running step.
// It writes one running fact immediately, then once a second while work runs:
// a Temporal heartbeat, and another fact when the sentence or the shown percent changes.
// The percent moves forward by at most five points in that second.
// A later progress write is best-effort and does not fail the work.
func (a *Activities) emit(ctx context.Context, jobID, label, message string, stage pipelinejobbus.Stage, sample func(shown float64) factSample, work func(context.Context) error) error {
	job, err := a.recordProgressJob(a.traced(ctx), jobID, stage, nil, headline(message))
	if err != nil {
		return err
	}

	errc := make(chan error, 1)
	go func() {
		errc <- work(ctx)
	}()

	pulse := time.NewTicker(time.Second)
	defer pulse.Stop()
	var beats int
	lastShown := -1.0
	lastPercent := -1
	lastMessage := message
	for {
		select {
		case err := <-errc:
			return err
		case <-pulse.C:
			beats++
			if beats%10 == 0 && activity.IsActivity(ctx) {
				activity.RecordHeartbeat(ctx, label)
				activity.GetLogger(ctx).Info("activity heartbeat", "step", label, "jobID", jobID, "percent", lastPercent)
			}
			current := sample(lastShown)
			if current.Message == "" {
				current.Message = message
			}
			var fraction float64
			var hasFraction bool
			if current.Has {
				fraction = advanceProgress(lastShown, lastShown >= 0, current.Fraction)
				lastShown = fraction
				hasFraction = true
			}
			percent := -1
			if hasFraction {
				percent = min(int(math.Round(fraction*100)), 100)
			}
			if percent == lastPercent && current.Message == lastMessage {
				continue
			}
			lastPercent = percent
			lastMessage = current.Message
			var stored *float64
			if hasFraction {
				value := fraction
				stored = &value
			}
			a.recordLaterProgress(ctx, job, stage, stored, current.Message)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// stub runs a simulated step for stubRun, emitting a steady percent through emit.
func (a *Activities) stub(ctx context.Context, jobID, label, message string, stage pipelinejobbus.Stage) error {
	start := time.Now()
	return a.emit(ctx, jobID, label, message, stage, func(float64) factSample {
		return factSample{Fraction: stubFraction(time.Since(start), stubRun), Has: true, Message: message}
	}, func(ctx context.Context) error {
		timer := time.NewTimer(stubRun)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	})
}

func stubFraction(elapsed, total time.Duration) float64 {
	done := 1.0
	if total > 0 {
		done = elapsed.Seconds() / total.Seconds()
	}
	if done < 0 {
		return 0
	}
	if done > 1 {
		return 1
	}
	return done
}

func headline(message string) *string {
	if message == "" {
		return nil
	}
	return &message
}
