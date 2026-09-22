// Package person holds the Person domain's service layer.
package person

import (
	"context"

	"github.com/nachoconques0/splitter_svc/backend/internal/entity/person"
)

//go:generate go tool mockgen -source=service.go -destination=mocks/service.go -package=mocks

// No Update and no Delete: a Share refers to a Person, so deleting one would
// leave a Bill pointing at nobody.
type Repository interface {
	Create(ctx context.Context, name string) (person.Person, error)
	List(ctx context.Context) ([]person.Person, error)
}

// Service holds the Person rules.
type Service struct {
	repository Repository
}

// New builds the service over the given storage.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}

// The name is judged here rather than left to the database's check, so the
// refusal is something a client can act on.
func (s *Service) Create(ctx context.Context, name string) (person.Person, error) {
	parsed, err := person.ParseName(name)
	if err != nil {
		return person.Person{}, err
	}
	return s.repository.Create(ctx, parsed)
}

// List returns everyone, in the order storage gives them.
func (s *Service) List(ctx context.Context) ([]person.Person, error) {
	return s.repository.List(ctx)
}
