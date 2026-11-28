// Package workflowname represents the name of a workflow a run can be started from.
package workflowname

import (
	"fmt"
	"regexp"
)

// WorkflowName represents the name of a registered workflow. The platform
// does not know what a workflow does.
type WorkflowName struct {
	value string
}

// String returns the value of the workflow name.
func (w WorkflowName) String() string {
	return w.value
}

// Equal provides support for the go-cmp package and testing.
func (w WorkflowName) Equal(w2 WorkflowName) bool {
	return w.value == w2.value
}

// MarshalText provides support for logging and any marshal needs.
func (w WorkflowName) MarshalText() ([]byte, error) {
	return []byte(w.value), nil
}

// =============================================================================

var workflowNameRegEx = regexp.MustCompile("^[a-z][a-z0-9-]{0,62}$")

// Parse parses the string value and returns a workflow name if the value
// complies with the rules for one: lower-case letters, digits and hyphens,
// starting with a letter.
func Parse(value string) (WorkflowName, error) {
	if !workflowNameRegEx.MatchString(value) {
		return WorkflowName{}, fmt.Errorf("invalid workflow name %q", value)
	}

	return WorkflowName{value}, nil
}

// MustParse parses the string value and returns a workflow name if the value
// complies with the rules for one. If an error occurs the function panics.
func MustParse(value string) WorkflowName {
	name, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return name
}
