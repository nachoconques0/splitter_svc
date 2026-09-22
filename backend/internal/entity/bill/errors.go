package bill

import (
	"strings"

	"github.com/google/uuid"
)

// Types rather than sentinels because each has to quote something back. Matched
// with errors.As.

type SumError struct {
	Sum Percentage
}

func (e SumError) Error() string {
	return "shares sum to " + e.Sum.Decimal() + ", not " + Whole.Decimal()
}

// DuplicatePersonError names the Person who appeared more than once.
type DuplicatePersonError struct {
	PersonID uuid.UUID
}

func (e DuplicatePersonError) Error() string {
	return "person " + e.PersonID.String() + " appears more than once"
}

// Every Person that does not exist, not just the first.
type UnknownPersonError struct {
	PersonIDs []uuid.UUID
}

func (e UnknownPersonError) Error() string {
	ids := make([]string, 0, len(e.PersonIDs))
	for _, id := range e.PersonIDs {
		ids = append(ids, id.String())
	}
	return "no person with id " + strings.Join(ids, ", ")
}

// A negative one can hide inside a set that still sums to 100.00.
type PercentageOutOfRangeError struct {
	Percentage Percentage
}

func (e PercentageOutOfRangeError) Error() string {
	return "percentage " + e.Percentage.Decimal() + " is outside 0.00 to " + Whole.Decimal()
}
