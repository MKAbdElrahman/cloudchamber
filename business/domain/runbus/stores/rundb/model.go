package rundb

import (
	"cmp"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/MKAbdElrahman/cloudchamber/business/types/phase"
	"github.com/MKAbdElrahman/cloudchamber/business/types/resumefrom"
	"github.com/MKAbdElrahman/cloudchamber/business/types/stepstate"
	"github.com/MKAbdElrahman/cloudchamber/business/types/workflowname"
	"github.com/google/uuid"
)

type runDB struct {
	ID                 uuid.UUID      `db:"run_id"`
	Workflow           string         `db:"workflow"`
	OwnerID            uuid.UUID      `db:"owner_id"`
	ProjectID          uuid.UUID      `db:"project_id"`
	IdempotencyKeyHash sql.NullString `db:"idempotency_key_hash"`
	Generation         int64          `db:"generation"`
	Input              string         `db:"input"`
	Suspend            bool           `db:"suspend"`
	Cancel             bool           `db:"cancel"`
	Attempt            int            `db:"attempt"`
	ResumeFrom         string         `db:"resume_from"`
	Phase              string         `db:"phase"`
	ObservedGeneration int64          `db:"observed_generation"`
	ObservedAttempt    int            `db:"observed_attempt"`
	ExecutionID        string         `db:"execution_id"`
	Reason             string         `db:"reason"`
	Message            string         `db:"message"`
	Steps              string         `db:"steps"`
	Result             string         `db:"result"`
	CancelRequestedAt  sql.NullTime   `db:"cancel_requested_at"`
	DateClosed         sql.NullTime   `db:"date_closed"`
	DateCreated        time.Time      `db:"date_created"`
	DateUpdated        time.Time      `db:"date_updated"`
}

type stepDB struct {
	State       string    `json:"state"`
	Message     string    `json:"message"`
	Fraction    float64   `json:"fraction"`
	Measured    bool      `json:"measured"`
	DateStarted time.Time `json:"dateStarted,omitzero"`
	DateUpdated time.Time `json:"dateUpdated,omitzero"`
}

type eventDB struct {
	ID          int64         `db:"event_id"`
	RunID       uuid.UUID     `db:"run_id"`
	ActorID     uuid.NullUUID `db:"actor_id"`
	Type        string        `db:"type"`
	Message     string        `db:"message"`
	DateCreated time.Time     `db:"date_created"`
}

const jsonNull = "null"

// toDBRun converts a business run into its database representation.
// The steps column is not written from here. UpsertStep owns it.
func toDBRun(bus runbus.Run) runDB {
	return runDB{
		ID:                 bus.ID,
		Workflow:           bus.Workflow.String(),
		OwnerID:            bus.OwnerID,
		ProjectID:          bus.ProjectID,
		IdempotencyKeyHash: sql.NullString{String: bus.IdempotencyKeyHash, Valid: bus.IdempotencyKeyHash != ""},
		Generation:         bus.Generation,
		Input:              cmp.Or(string(bus.Spec.Input), "{}"),
		Suspend:            bus.Spec.Suspend,
		Cancel:             bus.Spec.Cancel,
		Attempt:            bus.Spec.Attempt,
		ResumeFrom:         bus.Spec.ResumeFrom.String(),
		Phase:              bus.Status.Phase.String(),
		ObservedGeneration: bus.Status.ObservedGeneration,
		ObservedAttempt:    bus.Status.Attempt,
		ExecutionID:        bus.Status.ExecutionID,
		Reason:             bus.Status.Reason,
		Message:            bus.Status.Message,
		Result:             cmp.Or(string(bus.Status.Result), jsonNull),
		CancelRequestedAt:  nullTime(bus.Status.CancelRequestedAt),
		DateClosed:         nullTime(bus.Status.DateClosed),
		DateCreated:        bus.DateCreated.UTC(),
		DateUpdated:        bus.DateUpdated.UTC(),
	}
}

// toBusRun converts a database row into a business run.
func toBusRun(db runDB) (runbus.Run, error) {
	name, err := workflowname.Parse(db.Workflow)
	if err != nil {
		return runbus.Run{}, fmt.Errorf("parse workflow: %w", err)
	}
	from, err := resumefrom.Parse(db.ResumeFrom)
	if err != nil {
		return runbus.Run{}, fmt.Errorf("parse resume from: %w", err)
	}
	phase, err := phase.Parse(db.Phase)
	if err != nil {
		return runbus.Run{}, fmt.Errorf("parse phase: %w", err)
	}
	steps, err := toBusSteps(db.Steps)
	if err != nil {
		return runbus.Run{}, err
	}

	var result json.RawMessage
	if db.Result != jsonNull {
		result = json.RawMessage(db.Result)
	}

	return runbus.Run{
		ID:                 db.ID,
		Workflow:           name,
		OwnerID:            db.OwnerID,
		ProjectID:          db.ProjectID,
		IdempotencyKeyHash: db.IdempotencyKeyHash.String,
		Generation:         db.Generation,
		Spec: runbus.Spec{
			Input:      json.RawMessage(db.Input),
			Suspend:    db.Suspend,
			Cancel:     db.Cancel,
			Attempt:    db.Attempt,
			ResumeFrom: from,
		},
		Status: runbus.Status{
			Phase:              phase,
			ObservedGeneration: db.ObservedGeneration,
			Attempt:            db.ObservedAttempt,
			ExecutionID:        db.ExecutionID,
			Reason:             db.Reason,
			Message:            db.Message,
			Steps:              steps,
			Result:             result,
			CancelRequestedAt:  localTime(db.CancelRequestedAt),
			DateClosed:         localTime(db.DateClosed),
		},
		DateCreated: db.DateCreated.In(time.Local),
		DateUpdated: db.DateUpdated.In(time.Local),
	}, nil
}

// toBusRuns converts database rows into business runs.
func toBusRuns(dbs []runDB) ([]runbus.Run, error) {
	bus := make([]runbus.Run, len(dbs))
	for i, db := range dbs {
		var err error
		bus[i], err = toBusRun(db)
		if err != nil {
			return nil, err
		}
	}
	return bus, nil
}

// toDBStep converts a business step into the JSON stored under its key.
// UpsertStep keeps the start time already stored for the key, so the first report sets it.
func toDBStep(bus runbus.Step) stepDB {
	return stepDB{
		State:       bus.State.String(),
		Message:     bus.Message,
		Fraction:    bus.Fraction,
		Measured:    bus.Measured,
		DateStarted: bus.DateStarted.UTC(),
		DateUpdated: bus.DateUpdated.UTC(),
	}
}

// toBusSteps converts the steps column into business steps, in the order they started.
func toBusSteps(column string) ([]runbus.Step, error) {
	var stored map[string]stepDB
	if err := json.Unmarshal([]byte(column), &stored); err != nil {
		return nil, fmt.Errorf("decode steps: %w", err)
	}

	steps := make([]runbus.Step, 0, len(stored))
	for key, db := range stored {
		state, err := stepstate.Parse(db.State)
		if err != nil {
			return nil, fmt.Errorf("parse step %q: %w", key, err)
		}
		steps = append(steps, runbus.Step{
			Key:         key,
			State:       state,
			Message:     db.Message,
			Fraction:    db.Fraction,
			Measured:    db.Measured,
			DateStarted: db.DateStarted.In(time.Local),
			DateUpdated: db.DateUpdated.In(time.Local),
		})
	}
	slices.SortFunc(steps, func(a, b runbus.Step) int {
		return cmp.Or(a.DateStarted.Compare(b.DateStarted), cmp.Compare(a.Key, b.Key))
	})

	return steps, nil
}

// toDBEvent converts a business event into its database representation.
func toDBEvent(bus runbus.Event) eventDB {
	return eventDB{
		RunID:       bus.RunID,
		ActorID:     uuid.NullUUID{UUID: bus.ActorID, Valid: bus.ActorID != uuid.Nil},
		Type:        bus.Type,
		Message:     bus.Message,
		DateCreated: bus.DateCreated.UTC(),
	}
}

// toBusEvents converts database rows into business events.
func toBusEvents(dbs []eventDB) []runbus.Event {
	bus := make([]runbus.Event, len(dbs))
	for i, db := range dbs {
		bus[i] = runbus.Event{
			ID:          db.ID,
			RunID:       db.RunID,
			ActorID:     db.ActorID.UUID,
			Type:        db.Type,
			Message:     db.Message,
			DateCreated: db.DateCreated.In(time.Local),
		}
	}
	return bus
}

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t.UTC(), Valid: !t.IsZero()}
}

// localTime converts a nullable column. NULL is the zero time.
func localTime(t sql.NullTime) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time.In(time.Local)
}
