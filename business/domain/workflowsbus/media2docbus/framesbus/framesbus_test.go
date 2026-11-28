package framesbus_test

import (
	"bytes"
	"image/png"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/framesbus"
	"github.com/google/uuid"
)

func TestHasPicture(t *testing.T) {
	for contentType, want := range map[string]bool{"video/mp4": true, "audio/mpeg": false, " Audio/WAV ": false, "": true, "application/octet-stream": true} {
		if got := framesbus.HasPicture(contentType); got != want {
			t.Errorf("HasPicture(%q) = %v, want %v", contentType, got, want)
		}
	}
}

func TestExtractKeyFramesPublishesAFrame(t *testing.T) {
	files := media2docbus.NewTestFiles()
	src := media2docbus.Source{FileID: uuid.New(), OwnerID: uuid.New(), ProjectID: uuid.New(), Name: "week-03.mp4"}
	runID := uuid.New()

	fileID, err := framesbus.NewBusiness(files).ExtractKeyFrames(t.Context(), runID, src)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	out := files.Outputs[fileID]
	if fileID != media2docbus.OutputFileID(runID, media2docbus.KindFrames) || out.Kind != media2docbus.KindFrames || out.Name != "week-03-frame.png" {
		t.Errorf("id = %s output = %+v", fileID, out)
	}
	if _, err := png.Decode(bytes.NewReader(out.Body)); err != nil {
		t.Errorf("the frame is not a PNG: %v", err)
	}
}
