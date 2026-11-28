package renderbus_test

import (
	"strings"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus"
	"github.com/MKAbdElrahman/cloudchamber/business/domain/workflowsbus/media2docbus/renderbus"
	"github.com/google/uuid"
)

func TestComposePublishesTheDocument(t *testing.T) {
	files := media2docbus.NewTestFiles()
	src := media2docbus.Source{FileID: uuid.New(), OwnerID: uuid.New(), ProjectID: uuid.New(), Name: "week-03.mp4"}
	bus := renderbus.NewBusiness(files)
	runID := uuid.New()

	fileID, err := bus.Compose(t.Context(), runID, src, true)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	out := files.Outputs[fileID]
	if fileID != media2docbus.OutputFileID(runID, media2docbus.KindDocument) || out.Name != "week-03.md" || out.ContentType != "text/markdown" {
		t.Errorf("id = %s output = %+v", fileID, out)
	}
	if !strings.Contains(string(out.Body), "Frame at") {
		t.Error("a document with frames does not mention one")
	}

	plain, err := bus.Compose(t.Context(), uuid.New(), src, false)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if strings.Contains(string(files.Outputs[plain].Body), "Frame at") {
		t.Error("a document with no frames mentions one")
	}
}
