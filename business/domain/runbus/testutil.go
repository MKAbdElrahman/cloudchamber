package runbus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

// TestGenerateNewRuns is a helper method for testing.
func TestGenerateNewRuns(n int, ownerID uuid.UUID, projectID uuid.UUID) []NewRun {
	newRuns := make([]NewRun, n)

	idx := rand.Intn(10000)
	for i := range n {
		idx++

		newRuns[i] = NewRun{
			ID:        uuid.New(),
			Workflow:  workflowname.MustParse("sample"),
			OwnerID:   ownerID,
			ProjectID: projectID,
			Input:     json.RawMessage(fmt.Sprintf(`{"units": %d}`, idx)),
		}
	}

	return newRuns
}

// TestGenerateSeedRuns is a helper method for testing.
func TestGenerateSeedRuns(ctx context.Context, n int, api ExtBusiness, ownerID uuid.UUID, projectID uuid.UUID) ([]Run, error) {
	newRuns := TestGenerateNewRuns(n, ownerID, projectID)

	runs := make([]Run, len(newRuns))
	for i, nr := range newRuns {
		run, err := api.Create(ctx, nr)
		if err != nil {
			return nil, fmt.Errorf("seeding run: idx: %d : %w", i, err)
		}

		runs[i] = run
	}

	return runs, nil
}
