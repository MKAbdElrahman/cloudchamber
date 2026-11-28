// Package phase represents the observed state of a run.
package phase

import "fmt"

// The set of phases a run can be observed in.
var (
	Pending    = newPhase("pending")
	Running    = newPhase("running")
	Paused     = newPhase("paused")
	Succeeded  = newPhase("succeeded")
	Failed     = newPhase("failed")
	Canceled   = newPhase("canceled")
	Terminated = newPhase("terminated")
)

// =============================================================================

// Set of known phases.
var phases = make(map[string]Phase)

// Phase represents the observed state of a run. Only the controller writes it.
type Phase struct {
	value string
}

func newPhase(phase string) Phase {
	p := Phase{phase}
	phases[phase] = p
	return p
}

// String returns the name of the phase.
func (p Phase) String() string {
	return p.value
}

// Equal provides support for the go-cmp package and testing.
func (p Phase) Equal(p2 Phase) bool {
	return p.value == p2.value
}

// MarshalText provides support for logging and any marshal needs.
func (p Phase) MarshalText() ([]byte, error) {
	return []byte(p.value), nil
}

// Terminal reports whether the execution behind this phase has closed.
func (p Phase) Terminal() bool {
	switch p {
	case Succeeded, Failed, Canceled, Terminated:
		return true
	default:
		return false
	}
}

// =============================================================================

// Parse parses the string value and returns a phase if one exists.
func Parse(value string) (Phase, error) {
	phase, exists := phases[value]
	if !exists {
		return Phase{}, fmt.Errorf("invalid phase %q", value)
	}

	return phase, nil
}

// MustParse parses the string value and returns a phase if one exists. If
// an error occurs the function panics.
func MustParse(value string) Phase {
	phase, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return phase
}
