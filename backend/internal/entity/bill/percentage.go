package bill

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Whole is what a Share Set's Percentages must add up to: 100.00, held as 10000.
// The sum rule compares against it exactly; there is no tolerance value anywhere.
const Whole = Percentage(100 * 100)

// Not a Percentage at all, as opposed to one that is well formed and disallowed.
// The two get different status codes.
var ErrMalformedPercentage = errors.New("malformed percentage")

// ParsePercentage reads a decimal string into hundredths of a percent, digit by
// digit rather than through ParseFloat. 65.68 + 9.62 + 24.70 is a valid Share
// Set that adds up to 100.00000000000001 as float64.
func ParsePercentage(text string) (Percentage, error) {
	digits := strings.TrimSpace(text)
	if digits == "" {
		return 0, ErrMalformedPercentage
	}

	negative := false
	if rest, found := strings.CutPrefix(digits, "-"); found {
		negative, digits = true, rest
	}

	whole, fraction, hasPoint := strings.Cut(digits, ".")
	if !allDigits(whole) {
		return 0, ErrMalformedPercentage
	}
	if hasPoint {
		// More than two places is a value this type cannot hold, and dropping the
		// extra digits would change what was asked for.
		if !allDigits(fraction) || len(fraction) > decimalPlaces {
			return 0, ErrMalformedPercentage
		}
	}

	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrMalformedPercentage
	}
	// "33.3" means 33.30, not 33.03.
	hundredths, err := strconv.ParseInt(fraction+strings.Repeat("0", decimalPlaces-len(fraction)), 10, 64)
	if err != nil {
		return 0, ErrMalformedPercentage
	}

	// Checked before the conversion, so a number too large to be a Percentage is
	// malformed rather than wrapping into a plausible one.
	if units > (math.MaxInt64-hundredths)/100 {
		return 0, ErrMalformedPercentage
	}
	value := units*100 + hundredths
	if negative {
		value = -value
	}
	if value > math.MaxInt32 || value < math.MinInt32 {
		return 0, ErrMalformedPercentage
	}

	return Percentage(value), nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
