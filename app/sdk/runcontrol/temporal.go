package runcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MKAbdElrahman/cloudchamber/business/domain/runbus"
	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// Definition says how the engine runs one workflow.
type Definition struct {
	// WorkflowType is the workflow type name registered with the engine.
	WorkflowType string
	// TaskQueue is the queue the workflow's worker polls.
	TaskQueue string
}

// identity names the controller in the engine's history.
const identity = "cloudchamber-run-controller"

// Temporal drives runs on a Temporal namespace.
type Temporal struct {
	client    client.Client
	namespace string
	workflows map[string]Definition
}

// NewTemporal constructs the Temporal engine. workflows maps a workflow name to its definition.
func NewTemporal(c client.Client, namespace string, workflows map[string]Definition) *Temporal {
	if namespace == "" {
		namespace = client.DefaultNamespace
	}
	return &Temporal{client: c, namespace: namespace, workflows: workflows}
}

// Observe reports the latest execution of the run.
func (t *Temporal) Observe(ctx context.Context, run runbus.Run) (runbus.Execution, error) {
	workflowID := runbus.WorkflowID(run.ID)

	described, err := t.client.DescribeWorkflow(ctx, workflowID, "")
	if err != nil {
		if _, ok := errors.AsType[*serviceerror.NotFound](err); ok {
			return runbus.Execution{}, nil
		}
		return runbus.Execution{}, fmt.Errorf("describe: %w", err)
	}
	// The raw response carries what the typed description leaves out: whether a
	// cancel was requested, and each pending activity's pause state.
	desc := described.RawResponse

	info := desc.GetWorkflowExecutionInfo()
	exec := runbus.Execution{
		Found:           true,
		ID:              info.GetExecution().GetRunId(),
		CancelRequested: desc.GetWorkflowExtendedInfo().GetCancelRequested(),
	}

	switch info.GetStatus() {
	case enumspb.WORKFLOW_EXECUTION_STATUS_PAUSED:
		exec.Paused = true
	case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		exec.State = runbus.ExecutionCompleted
	case enumspb.WORKFLOW_EXECUTION_STATUS_FAILED:
		exec.State = runbus.ExecutionFailed
	case enumspb.WORKFLOW_EXECUTION_STATUS_TIMED_OUT:
		exec.State = runbus.ExecutionTimedOut
	case enumspb.WORKFLOW_EXECUTION_STATUS_CANCELED:
		exec.State = runbus.ExecutionCanceled
	case enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		exec.State = runbus.ExecutionTerminated
	}

	if exec.State == runbus.ExecutionOpen {
		exec.Activities = toActivities(desc.GetPendingActivities())
		return exec, nil
	}

	// The database keeps microseconds. Truncating here keeps a recorded status equal to the next observation.
	exec.DateClosed = info.GetCloseTime().AsTime().UTC().Truncate(time.Microsecond)

	closing, err := t.closeEvent(ctx, workflowID, exec.ID)
	if err != nil {
		return runbus.Execution{}, err
	}
	exec.Result, exec.Failure = outcome(closing)

	return exec, nil
}

// toActivities converts the pending activities of an open execution.
func toActivities(pending []*workflowpb.PendingActivityInfo) []runbus.Activity {
	activities := make([]runbus.Activity, len(pending))
	for i, act := range pending {
		state := act.GetState()
		activities[i] = runbus.Activity{
			ID: act.GetActivityId(),
			Running: state == enumspb.PENDING_ACTIVITY_STATE_STARTED ||
				state == enumspb.PENDING_ACTIVITY_STATE_PAUSE_REQUESTED ||
				state == enumspb.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED,
			Paused: act.GetPaused() ||
				state == enumspb.PENDING_ACTIVITY_STATE_PAUSED ||
				state == enumspb.PENDING_ACTIVITY_STATE_PAUSE_REQUESTED,
		}
	}
	return activities
}

// closeEvent returns the event that closed an execution.
func (t *Temporal) closeEvent(ctx context.Context, workflowID, runID string) (*historypb.HistoryEvent, error) {
	iter := t.client.GetWorkflowHistory(ctx, workflowID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT)
	var last *historypb.HistoryEvent
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return nil, fmt.Errorf("close event: %w", err)
		}
		last = event
	}
	return last, nil
}

// outcome reads the result or the failure text from a close event.
func outcome(event *historypb.HistoryEvent) (json.RawMessage, string) {
	switch event.GetEventType() {
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED:
		payloads := event.GetWorkflowExecutionCompletedEventAttributes().GetResult().GetPayloads()
		if len(payloads) == 0 || !json.Valid(payloads[0].GetData()) {
			return nil, ""
		}
		return json.RawMessage(payloads[0].GetData()), ""
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED:
		return nil, failureText(event.GetWorkflowExecutionFailedEventAttributes().GetFailure())
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TIMED_OUT:
		return nil, "the run took longer than its time limit"
	case enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_TERMINATED:
		return nil, event.GetWorkflowExecutionTerminatedEventAttributes().GetReason()
	default:
		return nil, ""
	}
}

// failureText picks the text a person should read from a workflow failure.
// The workflow failure wraps the activity failure, which wraps the error the
// step returned. The step's own message is the first application failure in that chain.
func failureText(failure *failurepb.Failure) string {
	text := failure.GetMessage()
	for f := failure; f != nil; f = f.GetCause() {
		if f.GetApplicationFailureInfo() != nil {
			return f.GetMessage()
		}
		text = f.GetMessage()
	}
	return text
}

// Apply takes one action on the run's execution.
func (t *Temporal) Apply(ctx context.Context, run runbus.Run, action runbus.Action) error {
	workflowID := runbus.WorkflowID(run.ID)
	service := t.client.WorkflowService()
	execution := &commonpb.WorkflowExecution{WorkflowId: workflowID}

	switch action.Kind {
	case runbus.ActionStart:
		return t.start(ctx, run)

	case runbus.ActionReset:
		return t.reset(ctx, run)

	case runbus.ActionPause:
		_, err := service.PauseWorkflowExecution(ctx, &workflowservice.PauseWorkflowExecutionRequest{
			Namespace:  t.namespace,
			WorkflowId: workflowID,
			Identity:   identity,
			Reason:     "suspended",
			RequestId:  uuid.NewString(),
		})
		return err

	case runbus.ActionUnpause:
		_, err := service.UnpauseWorkflowExecution(ctx, &workflowservice.UnpauseWorkflowExecutionRequest{
			Namespace:  t.namespace,
			WorkflowId: workflowID,
			Identity:   identity,
			Reason:     "resumed",
			RequestId:  uuid.NewString(),
		})
		return err

	case runbus.ActionPauseActivities:
		for _, id := range action.Activities {
			_, err := service.PauseActivity(ctx, &workflowservice.PauseActivityRequest{
				Namespace: t.namespace,
				Execution: execution,
				Identity:  identity,
				Activity:  &workflowservice.PauseActivityRequest_Id{Id: id},
				Reason:    "suspended",
			})
			if err != nil && !gone(err) {
				return fmt.Errorf("pause activity %s: %w", id, err)
			}
		}
		return nil

	case runbus.ActionUnpauseActivities:
		for _, id := range action.Activities {
			_, err := service.UnpauseActivity(ctx, &workflowservice.UnpauseActivityRequest{
				Namespace: t.namespace,
				Execution: execution,
				Identity:  identity,
				Activity:  &workflowservice.UnpauseActivityRequest_Id{Id: id},
			})
			if err != nil && !gone(err) {
				return fmt.Errorf("unpause activity %s: %w", id, err)
			}
		}
		return nil

	case runbus.ActionCancel:
		return t.client.CancelWorkflow(ctx, workflowID, "")

	case runbus.ActionTerminate:
		return t.client.TerminateWorkflow(ctx, workflowID, "", "the run did not stop within the cancel grace period")

	default:
		return fmt.Errorf("unknown action %d", action.Kind)
	}
}

// gone reports whether an activity finished between the observation and the action.
// The next observation sees the run as it now is.
func gone(err error) bool {
	if _, ok := errors.AsType[*serviceerror.NotFound](err); ok {
		return true
	}
	_, ok := errors.AsType[*serviceerror.FailedPrecondition](err)
	return ok
}

// start begins a new execution for the run. An execution that is already open is left as it is.
func (t *Temporal) start(ctx context.Context, run runbus.Run) error {
	def, ok := t.workflows[run.Workflow.String()]
	if !ok {
		return fmt.Errorf("workflow %q is not registered", run.Workflow)
	}

	opts := client.StartWorkflowOptions{
		ID:                       runbus.WorkflowID(run.ID),
		TaskQueue:                def.TaskQueue,
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowIDReusePolicy:    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}
	input := Input{
		RunID:     run.ID.String(),
		Workflow:  run.Workflow.String(),
		OwnerID:   run.OwnerID.String(),
		ProjectID: run.ProjectID.String(),
		Input:     run.Spec.Input,
	}

	if _, err := t.client.ExecuteWorkflow(ctx, opts, def.WorkflowType, input); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	return nil
}

// reset continues the closed execution from the first step that did not finish.
// History before that step is kept, so finished steps do not run again.
// When history has no such step, the run starts again from the beginning.
func (t *Temporal) reset(ctx context.Context, run runbus.Run) error {
	workflowID := runbus.WorkflowID(run.ID)
	runID := run.Status.ExecutionID

	point, err := t.resetPoint(ctx, workflowID, runID)
	if err != nil {
		return err
	}
	if point == 0 {
		return t.start(ctx, run)
	}

	_, err = t.client.ResetWorkflowExecution(ctx, &workflowservice.ResetWorkflowExecutionRequest{
		Namespace:                 t.namespace,
		WorkflowExecution:         &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: runID},
		Reason:                    "retry from the failed step",
		WorkflowTaskFinishEventId: point,
		RequestId:                 uuid.NewString(),
		Identity:                  identity,
	})
	if err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	return nil
}

// resetPoint finds the workflow task that scheduled the earliest activity that
// never completed. Resetting there schedules that activity again. Zero means
// every scheduled activity completed, or none was scheduled.
func (t *Temporal) resetPoint(ctx context.Context, workflowID, runID string) (int64, error) {
	// unfinished maps a scheduled activity's event id to the workflow task that scheduled it.
	unfinished := make(map[int64]int64)

	iter := t.client.GetWorkflowHistory(ctx, workflowID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		if err != nil {
			return 0, fmt.Errorf("history: %w", err)
		}
		switch event.GetEventType() {
		case enumspb.EVENT_TYPE_ACTIVITY_TASK_SCHEDULED:
			unfinished[event.GetEventId()] = event.GetActivityTaskScheduledEventAttributes().GetWorkflowTaskCompletedEventId()
		case enumspb.EVENT_TYPE_ACTIVITY_TASK_COMPLETED:
			delete(unfinished, event.GetActivityTaskCompletedEventAttributes().GetScheduledEventId())
		}
	}

	var point int64
	for _, task := range unfinished {
		if point == 0 || task < point {
			point = task
		}
	}
	return point, nil
}
