package sampleworkflow

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/MKAbdElrahman/cloudchamber/app/sdk/workflows"
)

func TestWorkflowChecksItsInput(t *testing.T) {
	def := Definition()

	tests := []struct {
		input string
		field string
	}{
		{input: `{}`},
		{input: `{"units": 40, "tickMillis": 250, "failForSeconds": 30}`},
		{input: `{"units": -1}`, field: "units"},
		{input: `{"units": 10001}`, field: "units"},
		{input: `{"tickMillis": 60001}`, field: "tickMillis"},
		{input: `{"failForSeconds": -5}`, field: "failForSeconds"},
		{input: `{"units": "many"}`, field: "input"},
	}
	for _, tt := range tests {
		err := def.Validate(t.Context(), workflows.Request{Input: json.RawMessage(tt.input)})
		switch {
		case tt.field == "":
			if err != nil {
				t.Errorf("%s refused: %v", tt.input, err)
			}
		default:
			inputErr, ok := errors.AsType[*workflows.InputError](err)
			if !ok || inputErr.Field != tt.field {
				t.Errorf("%s err = %v, want field %s", tt.input, err, tt.field)
			}
		}
	}
}

func TestWorkflowNamesItsSteps(t *testing.T) {
	def := Definition()
	if def.Name != Name || def.WorkflowType == "" || def.TaskQueue != TaskQueue || len(def.Steps) != 3 {
		t.Errorf("definition = %+v", def)
	}
	if !json.Valid(def.InputSchema) {
		t.Error("the input schema is not JSON")
	}
}
