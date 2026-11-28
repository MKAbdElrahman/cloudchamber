// Package stepstate represents the last reported state of one step of a run.
package stepstate

import "fmt"

// The set of states a step can report.
var (
	Working  = newStepState("working")
	Done     = newStepState("done")
	Skipped  = newStepState("skipped")
	Failed   = newStepState("failed")
	Canceled = newStepState("canceled")
	Paused   = newStepState("paused")
)

// =============================================================================

// Set of known step states.
var states = make(map[string]StepState)

// StepState represents the last reported state of one step.
type StepState struct {
	value string
}

func newStepState(state string) StepState {
	s := StepState{state}
	states[state] = s
	return s
}

// String returns the name of the step state.
func (s StepState) String() string {
	return s.value
}

// Equal provides support for the go-cmp package and testing.
func (s StepState) Equal(s2 StepState) bool {
	return s.value == s2.value
}

// MarshalText provides support for logging and any marshal needs.
func (s StepState) MarshalText() ([]byte, error) {
	return []byte(s.value), nil
}

// =============================================================================

// Parse parses the string value and returns a step state if one exists.
func Parse(value string) (StepState, error) {
	state, exists := states[value]
	if !exists {
		return StepState{}, fmt.Errorf("invalid step state %q", value)
	}

	return state, nil
}

// MustParse parses the string value and returns a step state if one exists.
// If an error occurs the function panics.
func MustParse(value string) StepState {
	state, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return state
}
