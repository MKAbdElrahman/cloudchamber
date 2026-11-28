package mediabus_test

import (
	"errors"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/mediabus"
	"github.com/google/uuid"
)

func TestSourceRules(t *testing.T) {
	owner, project := uuid.New(), uuid.New()
	ready := media2docbus.Source{FileID: uuid.New(), OwnerID: owner, ProjectID: project, Name: "week-03.mp4", ContentType: "video/mp4", Ready: true}
	uploading := media2docbus.Source{FileID: uuid.New(), OwnerID: owner, ProjectID: project, Ready: false}
	noBytes := media2docbus.Source{FileID: uuid.New(), OwnerID: owner, ProjectID: project, Ready: true}
	unnamed := media2docbus.Source{FileID: uuid.New(), OwnerID: owner, ProjectID: project, Ready: true}

	files := media2docbus.NewTestFiles()
	files.AddSource(ready, []byte("bytes"))
	files.AddSource(uploading, []byte("bytes"))
	files.AddSource(noBytes, nil)
	files.AddSource(unnamed, []byte("bytes"))
	bus := mediabus.NewBusiness(files, "")
	ctx := t.Context()

	if got, err := bus.CheckSource(ctx, owner, project, ready.FileID); err != nil || got != ready {
		t.Errorf("check = %+v, %v", got, err)
	}
	if _, err := bus.CheckSource(ctx, owner, project, uuid.New()); !errors.Is(err, media2docbus.ErrSourceNotFound) {
		t.Errorf("unknown file: %v", err)
	}
	if _, err := bus.CheckSource(ctx, owner, uuid.New(), ready.FileID); !errors.Is(err, media2docbus.ErrSourceNotInProject) {
		t.Errorf("another project: %v", err)
	}
	if _, err := bus.CheckSource(ctx, owner, project, uploading.FileID); !errors.Is(err, media2docbus.ErrSourceNotReady) {
		t.Errorf("not ready: %v", err)
	}

	// A request is checked without looking at the bytes. A run that starts looks.
	if _, err := bus.CheckSource(ctx, owner, project, noBytes.FileID); err != nil {
		t.Errorf("check looked at the bytes: %v", err)
	}
	if _, err := bus.ResolveSource(ctx, owner, project, noBytes.FileID); !errors.Is(err, media2docbus.ErrSourceMissing) {
		t.Errorf("no bytes: %v", err)
	}

	got, err := bus.ResolveSource(ctx, owner, project, unnamed.FileID)
	if err != nil || got.Name != "recording" {
		t.Errorf("resolve unnamed = %+v, %v", got, err)
	}
}
