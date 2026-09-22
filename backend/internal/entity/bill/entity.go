// Package bill holds the Bill aggregate and the integer types it is made of.
package bill

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("bill not found")
	ErrModified = errors.New("bill modified since it was read")
)

// Two minor units per major unit holds for every currency this app accepts; one
// with a different exponent would need a table here.
const decimalPlaces = 2

// Percentage is hundredths of a percent: 100.00% is 10000.
type Percentage int32

func (p Percentage) Decimal() string {
	return decimal(int64(p))
}

// Money is an amount in a currency's minor units: 8500 EUR is 85.00.
type Money struct {
	Minor    int64
	Currency string
}

func (m Money) Decimal() string {
	return decimal(m.Minor)
}

// decimal places the point into an integer holding decimalPlaces implied digits.
// It moves digits rather than dividing: dividing loses the sign when the whole
// part is zero, and negating to take an absolute value overflows at MinInt64.
// decimal places the point into an integer carrying decimalPlaces implied digits.
func decimal(value int64) string {
	digits := strconv.FormatInt(value, 10)

	sign := ""
	if rest, negative := strings.CutPrefix(digits, "-"); negative {
		sign, digits = "-", rest
	}
	for len(digits) <= decimalPlaces {
		digits = "0" + digits
	}

	split := len(digits) - decimalPlaces
	return sign + digits[:split] + "." + digits[split:]
}

type Share struct {
	PersonID   uuid.UUID
	PersonName string
	Percentage Percentage
}

type ShareSet []Share

// Validate checks everything the Share Set can be judged on by itself. Whether
// the people on it exist needs the database, so that stays in the service.
func (set ShareSet) Validate() error {
	seen := make(map[uuid.UUID]struct{}, len(set))
	sum := Percentage(0)

	for _, share := range set {
		// Per Share rather than left to the sum: 110.00 and -10.00 still total
		// exactly 100.00.
		if share.Percentage < 0 || share.Percentage > Whole {
			return PercentageOutOfRangeError{Percentage: share.Percentage}
		}
		if _, duplicate := seen[share.PersonID]; duplicate {
			return DuplicatePersonError{PersonID: share.PersonID}
		}
		seen[share.PersonID] = struct{}{}

		sum += share.Percentage
	}

	// Exact integer comparison, no tolerance.
	if sum != Whole {
		return SumError{Sum: sum}
	}

	return nil
}

type Bill struct {
	ID          uuid.UUID
	Description string
	Total       Money
	Version     int64
	Shares      ShareSet
}
