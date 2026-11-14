package sim

import (
	"context"
	"errors"
	"fmt"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/objectstore"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeInvalidInput   = "InvalidInput"
	errTypeSourceNotFound = "SourceNotFound"
)

// permanent records a failure the activity must not retry. The non-retryable
// flag sits on this error. Wrapping it in fmt.Errorf would drop the flag.
func permanent(ctx context.Context, errType, message string, cause error) error {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("error.type", errType),
		attribute.Bool("pipeline.non_retryable", true),
	)
	noteAttempt(ctx, span)
	if activity.IsActivity(ctx) {
		activity.GetLogger(ctx).Info("permanent activity error", "type", errType, "message", message)
	}
	return temporal.NewNonRetryableApplicationError(message, errType, cause)
}

// contextDone stops the activity before a write when the context is already canceled.
// Returning the context error lets the SDK report the activity as canceled.
func contextDone(ctx context.Context, point string) error {
	err := ctx.Err()
	if err == nil {
		return nil
	}
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("pipeline.stopped", point))
	noteAttempt(ctx, span)
	if activity.IsActivity(ctx) {
		activity.GetLogger(ctx).Info("activity stopped before write", "point", point)
	}
	return err
}

func noteAttempt(ctx context.Context, span trace.Span) {
	if !activity.IsActivity(ctx) {
		return
	}
	span.SetAttributes(attribute.Int("temporal.attempt", int(activity.GetInfo(ctx).Attempt)))
}

func invalidInput(ctx context.Context, message string, cause error) error {
	return permanent(ctx, errTypeInvalidInput, message, cause)
}

func loadSourceFailure(ctx context.Context, err error) error {
	if errors.Is(err, artifactbus.ErrNotFound) {
		return permanent(ctx, errTypeSourceNotFound, "load source", err)
	}
	return fmt.Errorf("load source: %w", err)
}

func headSourceFailure(ctx context.Context, err error) error {
	if objectstore.IsMissing(err) {
		return permanent(ctx, errTypeSourceNotFound, "head source", err)
	}
	return fmt.Errorf("head source: %w", err)
}

func readSourceFailure(ctx context.Context, err error) error {
	if objectstore.IsMissing(err) {
		return permanent(ctx, errTypeSourceNotFound, "read source", err)
	}
	return fmt.Errorf("read source: %w", err)
}
