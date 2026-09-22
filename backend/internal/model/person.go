package model

// CreatePersonRequest is the body of POST /people.
type CreatePersonRequest struct {
	Name string `json:"name"`
}

// Person is a Person as they cross the wire.
type Person struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// PeopleResponse is what GET /people answers with.
type PeopleResponse struct {
	People []Person `json:"people"`
}
