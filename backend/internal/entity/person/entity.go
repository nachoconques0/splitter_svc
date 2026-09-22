// Package person holds the Person aggregate.
package person

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

var ErrInvalidName = errors.New("a person needs a name")

// No rename and no delete: Shares refer to a Person, and the page edits
// percentages rather than names.
type Person struct {
	ID   uuid.UUID
	Name string
}

// ParseName trims first and judges after: a name of spaces is no name, and
// trimming stops a stray space making one person look like two.
func ParseName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrInvalidName
	}
	return trimmed, nil
}
