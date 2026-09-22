// Package bill holds the Bill domain's service layer.
package bill

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
)

//go:generate go tool mockgen -source=service.go -destination=mocks/service.go -package=mocks

// Repository is the storage this service needs.
type Repository interface {
	FindByID(ctx context.Context, id uuid.UUID) (bill.Bill, error)
	ExistingPeople(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)
	ReplaceShareSet(ctx context.Context, id uuid.UUID, expectedVersion int64, shares bill.ShareSet) (bill.Bill, error)
}

// Service holds the Bill rules that are not the database's to enforce.
type Service struct {
	repository Repository
}

// New builds the service over the given storage.
func New(repository Repository) *Service {
	return &Service{repository: repository}
}

// FindByID returns a Bill and the Share Set it owns.
func (s *Service) FindByID(ctx context.Context, id uuid.UUID) (bill.Bill, error) {
	return s.repository.FindByID(ctx, id)
}

// ReplaceShareSet checks every rule before storage is touched.
func (s *Service) ReplaceShareSet(ctx context.Context, id uuid.UUID, expectedVersion int64, shares bill.ShareSet) (bill.Bill, error) {
	if err := shares.Validate(); err != nil {
		return bill.Bill{}, err
	}

	if err := s.checkPeopleExist(ctx, shares); err != nil {
		return bill.Bill{}, err
	}

	// The version is not checked here: reading it first would leave a gap two
	// saves could both pass through, so the guard belongs in the write.
	return s.repository.ReplaceShareSet(ctx, id, expectedVersion, shares)
}

// checkPeopleExist refuses a Share Set naming someone who does not exist.
func (s *Service) checkPeopleExist(ctx context.Context, shares bill.ShareSet) error {
	ids := make([]uuid.UUID, 0, len(shares))
	for _, share := range shares {
		ids = append(ids, share.PersonID)
	}

	existing, err := s.repository.ExistingPeople(ctx, ids)
	if err != nil {
		return fmt.Errorf("checking the people on the share set: %w", err)
	}

	found := make(map[uuid.UUID]struct{}, len(existing))
	for _, id := range existing {
		found[id] = struct{}{}
	}

	var unknown []uuid.UUID
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		return bill.UnknownPersonError{PersonIDs: unknown}
	}

	return nil
}
