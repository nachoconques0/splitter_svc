package bill_test

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	"github.com/nachoconques0/splitter_svc/backend/internal/service/bill/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

// The same cases, with the same expected Amounts, run against the TypeScript
// Allocation in frontend/src/utils/allocation.spec.ts. The rule is written twice
// on purpose, so keep the two tables in step.
var allocationCases = []struct {
	name        string
	totalMinor  int64
	percentages []billentity.Percentage
	wantAmounts []string
}{
	{
		name:        "shares that divide the total evenly leave nothing over",
		totalMinor:  10000,
		percentages: []billentity.Percentage{2500, 2500, 2500, 2500},
		wantAmounts: []string{"25.00", "25.00", "25.00", "25.00"},
	},
	{
		// Rounding each Share alone gives 3.33 three times, or 9.99 of 10.00.
		name:        "a remainder is handed to the largest fractional part",
		totalMinor:  1000,
		percentages: []billentity.Percentage{3333, 3333, 3334},
		wantAmounts: []string{"3.33", "3.33", "3.34"},
	},
	{
		// Both Shares want the same cent. The first one gets it, every time, for
		// the same input — the alternative is a Bill that reads differently on
		// each request.
		name:        "a tie on the fractional parts goes to the earlier share",
		totalMinor:  3,
		percentages: []billentity.Percentage{5000, 5000},
		wantAmounts: []string{"0.02", "0.01"},
	},
	{
		name:        "a single share at 100% takes the whole total",
		totalMinor:  8500,
		percentages: []billentity.Percentage{10000},
		wantAmounts: []string{"85.00"},
	},
	{
		// A Share of nothing is owed nothing, even while a Remainder is being
		// handed out beside it: its fractional part is zero, and a zero can never
		// be among the largest.
		name:        "a share at 0% is never handed a cent",
		totalMinor:  10,
		percentages: []billentity.Percentage{0, 3333, 3333, 3334},
		wantAmounts: []string{"0.00", "0.03", "0.03", "0.04"},
	},
	{
		name:        "the seeded bill",
		totalMinor:  8500,
		percentages: []billentity.Percentage{3333, 3333, 3334},
		wantAmounts: []string{"28.33", "28.33", "28.34"},
	},
}

// billWith builds a Bill carrying exactly these Percentages against this Total.
func billWith(totalMinor int64, percentages []billentity.Percentage) billentity.Bill {
	built := seededBill()
	built.Total = billentity.Money{Minor: totalMinor, Currency: "EUR"}
	built.Shares = nil
	for i, percentage := range percentages {
		built.Shares = append(built.Shares, billentity.Share{
			PersonID:   uuid.MustParse(fmt.Sprintf("a0000000-0000-4000-8000-%012d", i+1)),
			PersonName: fmt.Sprintf("Person %c", 'A'+i),
			Percentage: percentage,
		})
	}
	return built
}

func TestEveryShareIsAnsweredWithTheAmountItComesTo(t *testing.T) {
	for _, tt := range allocationCases {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, returning(billWith(tt.totalMinor, tt.percentages), nil))

			recorder, decoded := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
			}

			shares := testutil.Array(t, decoded, "shares")
			if len(shares) != len(tt.wantAmounts) {
				t.Fatalf("answered with %d shares, want %d", len(shares), len(tt.wantAmounts))
			}

			for i, want := range tt.wantAmounts {
				share, ok := shares[i].(map[string]any)
				if !ok {
					t.Fatalf("shares[%d] = %#v, want an object", i, shares[i])
				}
				// The Amount crosses the wire as a decimal string, like every other
				// number this API speaks.
				if got := share["amount"]; got != want {
					t.Errorf("shares[%d].amount = %#v, want %q", i, got, want)
				}
			}
		})
	}
}

// Asserted separately from the expected Amounts, so a table entry with a typo in
// it still cannot pass.
func TestTheAmountsAlwaysSumToTheTotal(t *testing.T) {
	for _, tt := range allocationCases {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, returning(billWith(tt.totalMinor, tt.percentages), nil))

			_, decoded := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

			var summed int64
			for _, share := range testutil.Array(t, decoded, "shares") {
				amount, _ := share.(map[string]any)["amount"].(string)
				summed += minorUnits(t, amount)
			}

			if summed != tt.totalMinor {
				t.Errorf("the amounts sum to %d minor units, want the total of %d", summed, tt.totalMinor)
			}
		})
	}
}

func TestSharesOnEqualPercentagesDifferByAtMostACent(t *testing.T) {
	// A Total that divides evenly by none of these, so the Remainder is genuinely
	// in play for every Share.
	router := newRouter(t, returning(billWith(10007, []billentity.Percentage{3333, 3333, 3334}), nil))

	_, decoded := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

	shares := testutil.Array(t, decoded, "shares")
	first := minorUnits(t, shares[0].(map[string]any)["amount"].(string))
	second := minorUnits(t, shares[1].(map[string]any)["amount"].(string))

	if difference := first - second; difference > 1 || difference < -1 {
		t.Errorf("two shares on 33.33%% came to %d and %d minor units, a difference of %d", first, second, difference)
	}
}

// Reads an Amount back into minor units, so the test adds integers rather than
// the floats it is asserting the absence of.
func minorUnits(t *testing.T, amount string) int64 {
	t.Helper()

	whole, fraction, found := strings.Cut(amount, ".")
	if !found || len(fraction) != 2 {
		t.Fatalf("amount %q is not a decimal string carrying two places", amount)
	}

	minor, err := strconv.ParseInt(whole+fraction, 10, 64)
	if err != nil {
		t.Fatalf("amount %q is not a decimal string: %v", amount, err)
	}
	return minor
}

// The property the table's cases are examples of, across Share Sets nobody would
// think to write down. The per-Share bound matters as much as the sum: a rule
// that handed the whole Total to the first Share would satisfy the sum alone.
// The seed is fixed, so a failure names an input that can be reproduced.
func TestTheAmountsSumToTheTotalForAnyShareSet(t *testing.T) {
	random := rand.New(rand.NewPCG(1, 2))

	// One router for every case, so this stays at the seam the rest of the suite
	// uses.
	var current billentity.Bill
	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().FindByID(gomock.Any(), gomock.Any()).
			DoAndReturn(func(any, uuid.UUID) (billentity.Bill, error) { return current, nil }).AnyTimes()
	})

	for range 500 {
		totalMinor := random.Int64N(1_000_000)
		percentages := randomShareSet(random)
		current = billWith(totalMinor, percentages)

		_, decoded := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

		var summed int64
		for i, share := range testutil.Array(t, decoded, "shares") {
			amount := minorUnits(t, share.(map[string]any)["amount"].(string))
			summed += amount

			// What this Share was owed before the Remainder moved anything.
			owed, _ := percentages[i].Of(totalMinor)
			if amount != owed && amount != owed+1 {
				t.Fatalf("share %d came to %d minor units, want %d or %d, for total %d and percentages %v",
					i, amount, owed, owed+1, totalMinor, percentages)
			}
		}

		if summed != totalMinor {
			t.Fatalf("amounts sum to %d, want the total of %d, for percentages %v", summed, totalMinor, percentages)
		}
	}
}

// Percentages totalling exactly Whole, by cutting the whole at random points.
func randomShareSet(random *rand.Rand) []billentity.Percentage {
	cuts := make([]billentity.Percentage, random.IntN(8)+1)
	for i := range cuts {
		cuts[i] = billentity.Percentage(random.IntN(int(billentity.Whole) + 1))
	}
	slices.Sort(cuts)

	percentages := make([]billentity.Percentage, 0, len(cuts))
	previous := billentity.Percentage(0)
	for _, cut := range cuts {
		percentages = append(percentages, cut-previous)
		previous = cut
	}
	return append(percentages, billentity.Whole-previous)
}
