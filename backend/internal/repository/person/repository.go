// Package person is the Person domain's storage: hand-written SQL, no ORM.
package person

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nachoconques0/splitter_svc/backend/internal/entity/person"
)

// Repository reads and writes People.
type Repository struct {
	db *sql.DB
}

// New builds the repository over the given database.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create stores a Person under the given name.
func (r *Repository) Create(ctx context.Context, name string) (person.Person, error) {
	var created person.Person
	// RETURNING so creating a Person is one round trip.
	const query = `INSERT INTO people (name) VALUES ($1) RETURNING id, name`

	if err := r.db.QueryRowContext(ctx, query, name).Scan(&created.ID, &created.Name); err != nil {
		return person.Person{}, fmt.Errorf("creating person %q: %w", name, err)
	}
	return created, nil
}

// List returns everyone, ordered for the picker.
func (r *Repository) List(ctx context.Context) ([]person.Person, error) {
	// By name for the picker, then by id: names are not unique.
	const query = `SELECT id, name FROM people ORDER BY name, id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("reading people: %w", err)
	}
	defer rows.Close()

	var people []person.Person
	for rows.Next() {
		var found person.Person
		if err := rows.Scan(&found.ID, &found.Name); err != nil {
			return nil, fmt.Errorf("reading people: %w", err)
		}
		people = append(people, found)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading people: %w", err)
	}

	return people, nil
}
