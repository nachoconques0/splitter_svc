// Package bill is the Bill domain's storage: hand-written SQL, no ORM.
package bill

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
)

// Repository reads and writes Bills.
type Repository struct {
	db *sql.DB
}

// New builds the repository over the given database.
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Both *sql.DB and *sql.Tx provide this, so the read after a replace can run
// inside that replace's transaction.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// FindByID returns a Bill and the Share Set it owns.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (bill.Bill, error) {
	return findByID(ctx, r.db, id)
}

// findByID reads the Bill and its Share Set in one statement: read separately,
// the two could straddle a replace. LEFT JOIN so a Bill with no Shares still
// returns its own row rather than looking like a missing Bill.
func findByID(ctx context.Context, q queryer, id uuid.UUID) (bill.Bill, error) {
	const query = `
SELECT b.id,
       b.description,
       b.total,
       b.currency,
       b.version,
       s.person_id,
       p.name,
       s.percentage
  FROM bills b
  LEFT JOIN shares s ON s.bill_id = b.id
  LEFT JOIN people p ON p.id = s.person_id
 WHERE b.id = $1
 -- By name for the page, then by id: names are not unique.
 ORDER BY p.name, s.person_id`

	rows, err := q.QueryContext(ctx, query, id)
	if err != nil {
		return bill.Bill{}, fmt.Errorf("reading bill %s: %w", id, err)
	}
	defer rows.Close()

	var (
		found bill.Bill
		seen  bool
	)
	for rows.Next() {
		var (
			personID   uuid.NullUUID
			personName sql.NullString
			percentage sql.NullInt32
		)
		if err := rows.Scan(
			&found.ID,
			&found.Description,
			&found.Total.Minor,
			&found.Total.Currency,
			&found.Version,
			&personID,
			&personName,
			&percentage,
		); err != nil {
			return bill.Bill{}, fmt.Errorf("reading bill %s: %w", id, err)
		}
		seen = true

		// Nulls across the join mean no Shares, not one nameless Share.
		if !personID.Valid {
			continue
		}
		found.Shares = append(found.Shares, bill.Share{
			PersonID:   personID.UUID,
			PersonName: personName.String,
			Percentage: bill.Percentage(percentage.Int32),
		})
	}
	if err := rows.Err(); err != nil {
		return bill.Bill{}, fmt.Errorf("reading bill %s: %w", id, err)
	}

	if !seen {
		return bill.Bill{}, bill.ErrNotFound
	}
	return found, nil
}

// ExistingPeople returns the subset of ids that name a Person. It is the same
// check the foreign key would make, early enough to refuse an unknown Person
// before any transaction is opened.
func (r *Repository) ExistingPeople(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	const query = `SELECT id FROM people WHERE id = ANY($1)`

	asStrings := make([]string, 0, len(ids))
	for _, id := range ids {
		asStrings = append(asStrings, id.String())
	}

	rows, err := r.db.QueryContext(ctx, query, pq.Array(asStrings))
	if err != nil {
		return nil, fmt.Errorf("reading people: %w", err)
	}
	defer rows.Close()

	var existing []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("reading people: %w", err)
		}
		existing = append(existing, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("reading people: %w", err)
	}

	return existing, nil
}

// notAdvanced says why the guarded update affected nothing. The write has
// already decided not to land; this only picks which answer the client gets.
func notAdvanced(ctx context.Context, tx *sql.Tx, id uuid.UUID) error {
	const query = `SELECT version FROM bills WHERE id = $1`

	var current int64
	switch err := tx.QueryRowContext(ctx, query, id).Scan(&current); {
	case errors.Is(err, sql.ErrNoRows):
		return bill.ErrNotFound
	case err != nil:
		return fmt.Errorf("reading the version of bill %s: %w", id, err)
	default:
		return bill.ErrModified
	}
}

// ReplaceShareSet swaps the whole Share Set, provided the Bill is still at the
// version the client read. The delete and the inserts are one transaction: a
// Bill whose old set is gone and whose new one is half written accounts for
// nothing.
func (r *Repository) ReplaceShareSet(ctx context.Context, id uuid.UUID, expectedVersion int64, shares bill.ShareSet) (bill.Bill, error) {
	// The whole of the concurrency control, and the only statement that moves a
	// Bill's version. It affects a row only if the Bill is there and still at the
	// version the client read, so two saves racing on one Bill cannot both win.
	const advanceVersion = `UPDATE bills SET version = version + 1 WHERE id = $1 AND version = $2`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return bill.Bill{}, fmt.Errorf("replacing the share set of bill %s: %w", id, err)
	}
	// Rollback after a commit is a no-op, so this covers every path out.
	defer tx.Rollback() //nolint:errcheck // the meaningful error is the one returned below

	advanced, err := tx.ExecContext(ctx, advanceVersion, id, expectedVersion)
	if err != nil {
		return bill.Bill{}, fmt.Errorf("replacing the share set of bill %s: %w", id, err)
	}
	switch affected, err := advanced.RowsAffected(); {
	case err != nil:
		return bill.Bill{}, fmt.Errorf("replacing the share set of bill %s: %w", id, err)
	case affected == 0:
		return bill.Bill{}, notAdvanced(ctx, tx, id)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM shares WHERE bill_id = $1`, id); err != nil {
		return bill.Bill{}, fmt.Errorf("clearing the share set of bill %s: %w", id, err)
	}

	const insertShare = `INSERT INTO shares (bill_id, person_id, percentage) VALUES ($1, $2, $3)`
	for _, share := range shares {
		if _, err := tx.ExecContext(ctx, insertShare, id, share.PersonID, share.Percentage); err != nil {
			return bill.Bill{}, fmt.Errorf("adding a share to bill %s: %w", id, err)
		}
	}

	// Inside the transaction, so the answer carries what was just written along
	// with the Person names the request did not send.
	written, err := findByID(ctx, tx, id)
	if err != nil {
		return bill.Bill{}, err
	}

	if err := tx.Commit(); err != nil {
		return bill.Bill{}, fmt.Errorf("replacing the share set of bill %s: %w", id, err)
	}

	return written, nil
}
