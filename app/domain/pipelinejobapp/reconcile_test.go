package pipelinejobapp

import (
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/pipelinejobbus"
)

func TestActivityInFlightSkipsStoredJob(t *testing.T) {
	a := &app{}
	stored := pipelinejobbus.Job{
		WorkflowID: "cloudchamber-job",
		Status:     pipelinejobbus.MustParseStatus(pipelinejobbus.StatusCanceled),
	}

	activity, err := a.activityInFlight(t.Context(), stored)
	if err != nil {
		t.Fatal(err)
	}
	if activity != "" {
		t.Fatalf("activity = %q", activity)
	}

	open := pipelinejobbus.Job{Status: pipelinejobbus.MustParseStatus(pipelinejobbus.StatusRunning)}
	activity, err = a.activityInFlight(t.Context(), open)
	if err != nil {
		t.Fatal(err)
	}
	if activity != "" {
		t.Fatalf("activity = %q", activity)
	}
}
