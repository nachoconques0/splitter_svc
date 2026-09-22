package model

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Decimal is a number as it arrived: the digits that were sent, kept as text.
// Takes a quoted "33.33" or a bare 33.33, and neither becomes a float — the bare
// form decodes through json.Number, which is the token's own text.
type Decimal struct {
	raw string
}

func (d *Decimal) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}

	if len(data) > 0 && data[0] == '"' {
		var quoted string
		if err := json.Unmarshal(data, &quoted); err != nil {
			return fmt.Errorf("reading a decimal: %w", err)
		}
		d.raw = quoted
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("reading a decimal: %w", err)
	}
	d.raw = number.String()
	return nil
}

func (d Decimal) String() string { return d.raw }
