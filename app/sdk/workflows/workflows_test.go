package workflows

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry(t *testing.T) {
	refused := &InputError{Field: "size", Message: "size must be positive"}
	r := NewRegistry(
		Workflow{Name: "b-flow", WorkflowType: "B", TaskQueue: "b", Validate: func(context.Context, Request) error { return refused }},
		Workflow{Name: "a-flow", WorkflowType: "A", TaskQueue: "a"},
	)

	if !r.Has("a-flow") || r.Has("c-flow") {
		t.Error("has")
	}
	if list := r.List(); len(list) != 2 || list[0].Name != "a-flow" || list[1].Name != "b-flow" {
		t.Errorf("list = %+v", list)
	}
	if defs := r.Definitions(); defs["b-flow"].WorkflowType != "B" || defs["a-flow"].TaskQueue != "a" {
		t.Errorf("definitions = %+v", defs)
	}

	if err := r.Validate(t.Context(), "a-flow", Request{}); err != nil {
		t.Errorf("a workflow with no check refused a request: %v", err)
	}
	if err := r.Validate(t.Context(), "b-flow", Request{}); !errors.Is(err, refused) {
		t.Errorf("err = %v", err)
	}

	err := r.Validate(t.Context(), "c-flow", Request{})
	if inputErr, ok := errors.AsType[*InputError](err); !ok || inputErr.Field != "workflow" {
		t.Errorf("unknown workflow err = %v", err)
	}
}

func TestRegistryRefusesAMistake(t *testing.T) {
	for name, list := range map[string][]Workflow{
		"no workflow type": {{Name: "a", TaskQueue: "q"}},
		"no queue":         {{Name: "a", WorkflowType: "W"}},
		"no name":          {{WorkflowType: "W", TaskQueue: "q"}},
		"twice":            {{Name: "a", WorkflowType: "W", TaskQueue: "q"}, {Name: "a", WorkflowType: "W", TaskQueue: "q"}},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("the registry accepted it")
				}
			}()
			NewRegistry(list...)
		})
	}
}
