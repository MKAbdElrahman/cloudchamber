// Package title represents a project title.
package title

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxRunes = 80

// Title is a project title. A value only exists after Parse accepts it.
type Title struct {
	value string
}

// String returns the title.
func (t Title) String() string {
	return t.value
}

// Equal provides support for the go-cmp package and testing.
func (t Title) Equal(t2 Title) bool {
	return t.value == t2.value
}

// MarshalText provides support for logging and any marshal needs.
func (t Title) MarshalText() ([]byte, error) {
	return []byte(t.value), nil
}

// Parse trims the value and accepts one to eighty characters with no controls.
func Parse(value string) (Title, error) {
	value = strings.TrimSpace(value)
	n := utf8.RuneCountInString(value)
	if n < 1 || n > maxRunes {
		return Title{}, fmt.Errorf("invalid title %q", value)
	}

	for _, r := range value {
		if unicode.IsControl(r) {
			return Title{}, fmt.Errorf("invalid title %q", value)
		}
	}

	return Title{value: value}, nil
}

// MustParse parses the value and panics if it is not a title.
func MustParse(value string) Title {
	t, err := Parse(value)
	if err != nil {
		panic(err)
	}

	return t
}
