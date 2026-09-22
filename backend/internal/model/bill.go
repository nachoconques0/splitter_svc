// Package model holds the DTOs that cross the wire. They are separate from the
// entities on purpose: an entity holds Percentages and money as integers, and
// every one of those becomes a decimal string here.
package model

// ShareSetResponse is what both share routes answer with.
type ShareSetResponse struct {
	Bill   Bill    `json:"bill"`
	Shares []Share `json:"shares"`
}

// Bill is a Bill as it crosses the wire.
type Bill struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Total       string `json:"total"`
	Currency    string `json:"currency"`
	Version     int64  `json:"version"`
}

// Share is one Share as it crosses the wire.
type Share struct {
	PersonID   string `json:"person_id"`
	Name       string `json:"name"`
	Percentage string `json:"percentage"`
	Amount     string `json:"amount"`
}

// ReplaceShareSetRequest is the body of PUT /bills/{id}/shares.
type ReplaceShareSetRequest struct {
	Version int64        `json:"version"`
	Shares  []ShareInput `json:"shares"`
}

// ShareInput is one Share as it arrives.
type ShareInput struct {
	PersonID   string  `json:"person_id"`
	Percentage Decimal `json:"percentage"`
}
