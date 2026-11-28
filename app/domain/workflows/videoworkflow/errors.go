package videoworkflow

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

const (
	errTypeInvalidInput   = "InvalidInput"
	errTypeSourceNotFound = "SourceNotFound"
)

// permanent is a failure the engine must not retry. The message is what the
// person who started the run reads. The non-retryable flag sits on this error,
// so wrapping it in fmt.Errorf would drop the flag.
func permanent(ctx context.Context, errType, message string, cause error) error {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("error.type", errType),
		attribute.Bool("run.non_retryable", true),
	)
	if activity.IsActivity(ctx) {
		span.SetAttributes(attribute.Int("temporal.attempt", int(activity.GetInfo(ctx).Attempt)))
		activity.GetLogger(ctx).Info("permanent activity error", "type", errType, "message", message)
	}
	return temporal.NewNonRetryableApplicationError(message, errType, cause)
}

func invalidInput(ctx context.Context, message string, cause error) error {
	return permanent(ctx, errTypeInvalidInput, message, cause)
}
