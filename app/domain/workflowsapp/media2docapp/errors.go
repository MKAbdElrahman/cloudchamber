package media2docapp

import (
	"context"
	"errors"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
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

// invalidInput is a permanent failure for an input the step cannot read.
func invalidInput(ctx context.Context, err error) error {
	return permanent(ctx, errTypeInvalidInput, err.Error(), err)
}

// sourceFailure decides whether a failure to use the recording can be retried.
// A recording the run may not convert, or whose bytes are gone, will not get
// better on another attempt. Anything else may be the store having a bad moment.
func sourceFailure(ctx context.Context, err error) error {
	for _, rule := range []error{
		media2docbus.ErrSourceNotFound,
		media2docbus.ErrSourceNotInProject,
		media2docbus.ErrSourceNotReady,
		media2docbus.ErrSourceMissing,
	} {
		if errors.Is(err, rule) {
			return permanent(ctx, errTypeSourceNotFound, rule.Error(), err)
		}
	}
	return err
}
