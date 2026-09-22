package bill_test

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	billcontroller "github.com/nachoconques0/splitter_svc/backend/internal/controller/bill"
	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	httpmocks "github.com/nachoconques0/splitter_svc/backend/internal/http/mocks"
	billservice "github.com/nachoconques0/splitter_svc/backend/internal/service/bill"
	"github.com/nachoconques0/splitter_svc/backend/internal/service/bill/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// The ids the seed migration creates, so the fixture and the seeded Bill cannot
// drift into describing different things.
var (
	billID  = uuid.MustParse("b1110000-0000-4000-8000-000000000001")
	adaID   = uuid.MustParse("a0000000-0000-4000-8000-000000000001")
	graceID = uuid.MustParse("a0000000-0000-4000-8000-000000000002")
	alanID  = uuid.MustParse("a0000000-0000-4000-8000-000000000003")
)

// seededBill mirrors the Bill the seed migration creates: 85.00 EUR, three Shares
// of 33.33/33.33/33.34 totalling exactly 100.00.
func seededBill() billentity.Bill {
	return billentity.Bill{
		ID:          billID,
		Description: "Dinner at Trattoria da Enzo",
		Total:       billentity.Money{Minor: 8500, Currency: "EUR"},
		Version:     seededVersion,
		Shares: billentity.ShareSet{
			{PersonID: adaID, PersonName: "Ada Lovelace", Percentage: 3333},
			{PersonID: alanID, PersonName: "Alan Turing", Percentage: 3334},
			{PersonID: graceID, PersonName: "Grace Hopper", Percentage: 3333},
		},
	}
}

// The real router, controller and service, mocking only the repository — the one
// seam. Everything worth proving is a status, an error_code and a body.
func newRouter(t *testing.T, setup func(*mocks.MockRepository)) http.Handler {
	t.Helper()

	controller := gomock.NewController(t)
	repository := mocks.NewMockRepository(controller)
	if setup != nil {
		setup(repository)
	}

	logger := slog.New(slog.DiscardHandler)
	pinger := httpmocks.NewMockPinger(controller)

	// The people routes carry no expectations, so reaching them fails the test.
	return splitterhttp.NewRouter(logger, pinger, billcontroller.New(logger, billservice.New(repository)),
		httpmocks.NewMockPersonController(controller))
}

// returning stubs the repository to answer with one Bill.
func returning(found billentity.Bill, err error) func(*mocks.MockRepository) {
	return func(repository *mocks.MockRepository) {
		repository.EXPECT().FindByID(gomock.Any(), gomock.Any()).Return(found, err).AnyTimes()
	}
}

func shareSetPath(id uuid.UUID) string { return "/bills/" + id.String() + "/shares" }

// The exact field set at every level, so the contract cannot grow one unnoticed.
func TestReadingAShareSetReturnsTheBillAndItsShares(t *testing.T) {
	router := newRouter(t, returning(seededBill(), nil))

	recorder, body := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, body)
	}

	testutil.AssertFieldSet(t, body, "bill", "shares")

	billBody := testutil.Object(t, body, "bill")
	testutil.AssertFieldSet(t, billBody, "id", "description", "total", "currency", "version")
	testutil.AssertFields(t, billBody, map[string]any{
		"id":          billID.String(),
		"description": "Dinner at Trattoria da Enzo",
		// The Total crosses the wire as a decimal string, never as minor units.
		"total":    "85.00",
		"currency": "EUR",
		// The version a client must send back to save. JSON has one number type,
		// so a decoded body carries it as a float64 — it is a counter, not money.
		"version": float64(seededVersion),
	})

	shares := testutil.Array(t, body, "shares")
	if len(shares) != 3 {
		t.Fatalf("shares has %d entries, want 3", len(shares))
	}

	wantShares := []map[string]any{
		{"person_id": adaID.String(), "name": "Ada Lovelace", "percentage": "33.33"},
		{"person_id": alanID.String(), "name": "Alan Turing", "percentage": "33.34"},
		{"person_id": graceID.String(), "name": "Grace Hopper", "percentage": "33.33"},
	}
	for i, want := range wantShares {
		share, ok := shares[i].(map[string]any)
		if !ok {
			t.Fatalf("shares[%d] = %#v, want an object", i, shares[i])
		}
		testutil.AssertFieldSet(t, share, "person_id", "name", "percentage", "amount")
		testutil.AssertFields(t, share, want)
	}
}

func TestABillWithNoSharesAnswersWithAnEmptyArray(t *testing.T) {
	empty := seededBill()
	empty.Shares = nil

	_, body := testutil.Do(t, newRouter(t, returning(empty, nil)), http.MethodGet, shareSetPath(billID), nil)

	shares := testutil.Array(t, body, "shares")
	if len(shares) != 0 {
		t.Errorf("shares = %#v, want an empty array", shares)
	}
}

func TestPercentagesCrossTheWireAsDecimalStrings(t *testing.T) {
	tests := []struct {
		name       string
		percentage billentity.Percentage
		want       string
	}{
		{name: "a whole Share", percentage: 10000, want: "100.00"},
		{name: "two decimal places", percentage: 3333, want: "33.33"},
		{name: "a trailing zero is kept", percentage: 2500, want: "25.00"},
		{name: "a leading zero in the hundredths", percentage: 1005, want: "10.05"},
		{name: "under one percent", percentage: 7, want: "0.07"},
		{name: "a Share of nothing", percentage: 0, want: "0.00"},
		// Rejected by the API, but a formatter that dropped the sign would make the
		// rejection quote a number nobody sent.
		{name: "a negative under one percent keeps its sign", percentage: -7, want: "-0.07"},
		{name: "a negative Share keeps its sign", percentage: -3333, want: "-33.33"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			one := seededBill()
			one.Shares = billentity.ShareSet{{PersonID: adaID, PersonName: "Ada Lovelace", Percentage: tt.percentage}}

			_, body := testutil.Do(t, newRouter(t, returning(one, nil)), http.MethodGet, shareSetPath(billID), nil)

			share := testutil.Array(t, body, "shares")[0].(map[string]any)
			if got := share["percentage"]; got != tt.want {
				t.Errorf("percentage = %#v, want %q", got, tt.want)
			}
		})
	}
}

func TestTheTotalCrossesTheWireAsADecimalString(t *testing.T) {
	tests := []struct {
		name  string
		minor int64
		want  string
	}{
		{name: "a whole number of euros", minor: 8500, want: "85.00"},
		{name: "cents are kept", minor: 8501, want: "85.01"},
		{name: "under one euro", minor: 7, want: "0.07"},
		{name: "nothing at all", minor: 0, want: "0.00"},
		{name: "a large Total is not rounded", minor: 123456789012345, want: "1234567890123.45"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			one := seededBill()
			one.Total = billentity.Money{Minor: tt.minor, Currency: "EUR"}

			_, body := testutil.Do(t, newRouter(t, returning(one, nil)), http.MethodGet, shareSetPath(billID), nil)

			if got := testutil.Object(t, body, "bill")["total"]; got != tt.want {
				t.Errorf("total = %#v, want %q", got, tt.want)
			}
		})
	}
}

func TestReadingABillThatIsNotThere(t *testing.T) {
	missingID := uuid.MustParse("00000000-0000-4000-8000-00000000dead")

	tests := []struct {
		name          string
		target        string
		setup         func(*mocks.MockRepository)
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "a Bill that does not exist",
			target:        shareSetPath(missingID),
			setup:         returning(billentity.Bill{}, billentity.ErrNotFound),
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "bill_not_found",
		},
		{
			// No repository stub, so the mock fails the test if the request reaches
			// it: a malformed id is rejected before it can become a driver error the
			// client would see as a 500.
			name:          "an id that is not an identifier at all",
			target:        "/bills/not-a-uuid/shares",
			setup:         nil,
			wantStatus:    http.StatusBadRequest,
			wantErrorCode: "invalid_bill_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, body := testutil.Do(t, newRouter(t, tt.setup), http.MethodGet, tt.target, nil)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %v)", recorder.Code, tt.wantStatus, body)
			}
			testutil.AssertFieldSet(t, body, "code", "error_code", "message")
			testutil.AssertFields(t, body, map[string]any{
				"code":       float64(tt.wantStatus),
				"error_code": tt.wantErrorCode,
			})
		})
	}
}

func TestAnUnexpectedRepositoryFailureDoesNotLeak(t *testing.T) {
	router := newRouter(t, returning(billentity.Bill{}, errors.New(`pq: relation "shares" does not exist`)))

	recorder, body := testutil.Do(t, router, http.MethodGet, shareSetPath(billID), nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	testutil.AssertFields(t, body, map[string]any{"error_code": "internal_error"})
	if leaked := recorder.Body.String(); strings.Contains(leaked, "pq:") || strings.Contains(leaked, "relation") {
		t.Errorf("the driver error reached the client: %s", leaked)
	}
}
