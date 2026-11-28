// Package resumefrom represents the point a new attempt of a run begins at.
package resumefrom

import "fmt"

// The set of points a new attempt can begin at.
var (
	Start   = newResumeFrom("start")
	Failure = newResumeFrom("failure")
)

// =============================================================================

// Set of known resume points.
var points = make(map[string]ResumeFrom)

// ResumeFrom represents where a new attempt of a run begins.
type ResumeFrom struct {
	value string
}

func newResumeFrom(point string) ResumeFrom {
	r := ResumeFrom{point}
	points[point] = r
	return r
}

// String returns the name of the resume point.
func (r ResumeFrom) String() string {
	return r.value
}

// Equal provides support for the go-cmp package and testing.
func (r ResumeFrom) Equal(r2 ResumeFrom) bool {
	return r.value == r2.value
}

// MarshalText provides support for logging and any marshal needs.
func (r ResumeFrom) MarshalText() ([]byte, error) {
	return []byte(r.value), nil
}

// =============================================================================

// Parse parses the string value and returns a resume point if one exists.
func Parse(value string) (ResumeFrom, error) {
	point, exists := points[value]
	if !exists {
		return ResumeFrom{}, fmt.Errorf("invalid resume point %q", value)
	}

	return point, nil
}

// MustParse parses the string value and returns a resume point if one exists.
// If an error occurs the function panics.
func MustParse(value string) ResumeFrom {
	point, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return point
}
