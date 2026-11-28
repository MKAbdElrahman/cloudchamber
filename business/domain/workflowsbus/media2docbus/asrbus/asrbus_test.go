package asrbus_test

import (
	"encoding/json"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/asrbus"
	"github.com/google/uuid"
)

func TestTranscribePublishesTheTranscript(t *testing.T) {
	files := media2docbus.NewTestFiles()
	src := media2docbus.Source{FileID: uuid.New(), OwnerID: uuid.New(), ProjectID: uuid.New(), Name: "week-03.mp4"}
	runID := uuid.New()

	fileID, err := asrbus.NewBusiness(files).Transcribe(t.Context(), runID, src)
	if err != nil {
		t.Fatalf("transcribe: %v", err)
	}

	out, ok := files.Outputs[fileID]
	if !ok || fileID != media2docbus.OutputFileID(runID, media2docbus.KindTranscript) {
		t.Fatalf("output id = %s, stored = %v", fileID, ok)
	}
	if out.Kind != media2docbus.KindTranscript || out.Name != "week-03.transcript.json" || out.ContentType != "application/json" {
		t.Errorf("output = %+v", out)
	}
	if out.OwnerID != src.OwnerID || out.ProjectID != src.ProjectID || !json.Valid(out.Body) {
		t.Errorf("output belongs to %s/%s, valid json = %v", out.OwnerID, out.ProjectID, json.Valid(out.Body))
	}
}
