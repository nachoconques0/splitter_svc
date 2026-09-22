package bill

import (
	"cmp"
	"slices"
)

// AllocatedShare is a Share and the Amount its Percentage comes to.
type AllocatedShare struct {
	Share
	Amount Money
}

// Allocate turns the Share Set and Total into an Amount for each Share.
//
// Rounding each Share on its own loses money: three Shares of 33.33/33.33/33.34
// against a 10.00 Total round to 3.33 each and account for 9.99. Every Share is
// floored instead, and the remainder handed out a unit at a time to the largest
// fractional parts, so the Amounts sum to the Total exactly. Ties go to the
// earlier Share, so the same Bill allocates the same way every time.
func (b Bill) Allocate() []AllocatedShare {
	allocated := make([]AllocatedShare, len(b.Shares))

	type fraction struct {
		position int
		size     int64
	}
	fractions := make([]fraction, len(b.Shares))

	var floored, claimed int64
	for i, share := range b.Shares {
		amount, remainder := share.Percentage.Of(b.Total.Minor)

		allocated[i] = AllocatedShare{
			Share:  share,
			Amount: Money{Minor: amount, Currency: b.Total.Currency},
		}
		fractions[i] = fraction{position: i, size: remainder}
		floored += amount
		claimed += int64(share.Percentage)
	}

	// A Share Set that does not account for the whole Bill has no Total for its
	// Amounts to add up to, so there is no remainder to hand out. The page
	// allocates one of these while it is still being typed.
	if claimed != int64(Whole) {
		return allocated
	}

	// Always smaller than the number of Shares: the fractions sum to Whole times
	// this, and none of them reaches Whole.
	remainder := b.Total.Minor - floored

	slices.SortStableFunc(fractions, func(a, b fraction) int {
		return cmp.Compare(b.size, a.size)
	})
	for i := int64(0); i < remainder; i++ {
		allocated[fractions[i].position].Amount.Minor++
	}

	return allocated
}

// Of applies the Percentage to an amount of money, answering the whole minor
// units it comes to and what was rounded away, scaled by Whole.
//
// Of applies the Percentage to an amount of money, answering the whole units it
// comes to and what was rounded away. The multiplication is split around the
// division because total * p overflows int64 above about nine trillion euros.
func (p Percentage) Of(total int64) (amount, remainder int64) {
	units, part := total/int64(Whole), total%int64(Whole)

	scaled := part * int64(p)
	return units*int64(p) + scaled/int64(Whole), scaled % int64(Whole)
}
