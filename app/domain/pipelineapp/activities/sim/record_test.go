package sim

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.temporal.io/sdk/temporal"
)

func TestFinishRecordsAPermanentFailure(t *testing.T) {
	cause := temporal.NewNonRetryableApplicationError("load source", "SourceNotFound", nil)
	err := (&Activities{}).finish(context.Background(), "11111111-1111-1111-1111-111111111111", cause, sourceFailed, sourceCanceled)
	if err == nil || !strings.Contains(err.Error(), "pipeline jobs") {
		t.Fatalf("finish error = %v", err)
	}
}

func TestFinishLeavesARetryableFailure(t *testing.T) {
	cause := errors.New("temporary")
	err := (&Activities{}).finish(context.Background(), "job", cause, sourceFailed, sourceCanceled)
	if !errors.Is(err, cause) {
		t.Fatalf("finish error = %v", err)
	}
}

func TestFinishDoesNotReportCancelUntilTheJobIsStored(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := (&Activities{}).finish(ctx, "11111111-1111-1111-1111-111111111111", ctx.Err(), transcriptFailed, transcriptCanceled)
	if err == nil || !strings.Contains(err.Error(), "pipeline jobs") {
		t.Fatalf("finish error = %v", err)
	}
}
