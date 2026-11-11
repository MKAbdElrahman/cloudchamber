package sim

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/artifactbus"
)

func TestPermanentErrorIsNonRetryable(t *testing.T) {
	err := permanent(context.Background(), errTypeInvalidInput, "file id", errors.New("bad"))

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error type = %T", err)
	}
	if !appErr.NonRetryable() {
		t.Fatal("permanent error is retryable")
	}
	if appErr.Type() != errTypeInvalidInput {
		t.Fatalf("error type = %s", appErr.Type())
	}
}

func TestLoadSourceNotFoundIsPermanent(t *testing.T) {
	err := loadSourceFailure(context.Background(), artifactbus.ErrNotFound)

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) || !appErr.NonRetryable() || appErr.Type() != errTypeSourceNotFound {
		t.Fatalf("load source error = %v", err)
	}
}

func TestLoadSourceDBErrorStaysRetryable(t *testing.T) {
	err := loadSourceFailure(context.Background(), errors.New("connection reset"))

	if appErr, ok := errors.AsType[*temporal.ApplicationError](err); ok {
		t.Fatalf("transient load error became %s nonRetryable=%v", appErr.Type(), appErr.NonRetryable())
	}
}

func TestPublishStopsBeforeWriteWhenCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (&Activities{}).publish(ctx, "job", uuid.New(), uuid.Nil, artifactbus.KindAudio, "week-03.wav", "audio/wav", []byte("wav"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("publish error = %v", err)
	}
}
