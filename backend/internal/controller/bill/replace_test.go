package bill_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	"github.com/nachoconques0/splitter_svc/backend/internal/service/bill/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

// What a client that had just read the seeded Bill would send back.
const seededVersion = 1

// Builds a replace payload from person id / percentage pairs, so the tests read
// as the Share Sets they are rather than as JSON.
func body(pairs ...string) *strings.Reader {
	return versionedBody(seededVersion, pairs...)
}

func versionedBody(version int64, pairs ...string) *strings.Reader {
	shares := make([]string, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		shares = append(shares, `{"person_id":"`+pairs[i]+`","percentage":"`+pairs[i+1]+`"}`)
	}
	return strings.NewReader(fmt.Sprintf(`{"version":%d,"shares":[%s]}`, version, strings.Join(shares, ",")))
}

// allPeopleExist stubs the reference check every valid Share Set makes.
func allPeopleExist(repository *mocks.MockRepository) {
	repository.EXPECT().ExistingPeople(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, ids []uuid.UUID) ([]uuid.UUID, error) { return ids, nil }).AnyTimes()
}

// So a stored Share can be answered with the name the request never sent.
var names = map[uuid.UUID]string{
	adaID:   "Ada Lovelace",
	graceID: "Grace Hopper",
	alanID:  "Alan Turing",
}

// The repository answers with the Share Set it was handed, so what reached
// storage is visible in the response rather than asserted on the mock.
func storesWhatItIsGiven(repository *mocks.MockRepository) {
	allPeopleExist(repository)
	repository.EXPECT().ReplaceShareSet(gomock.Any(), billID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ any, _ uuid.UUID, _ int64, shares billentity.ShareSet) (billentity.Bill, error) {
			stored := seededBill()
			stored.Shares = nil
			for _, share := range shares {
				share.PersonName = names[share.PersonID]
				stored.Shares = append(stored.Shares, share)
			}
			return stored, nil
		})
}

// The mandated test: a Share Set that does not total exactly 100.00 is refused,
// and the refusal quotes what it did total.
func TestAShareSetThatDoesNotSumTo100IsRejected(t *testing.T) {
	// No replace expectation: reaching storage fails the test, which is how "a
	// rejected request never opens a transaction" is shown at this seam.
	router := newRouter(t, allPeopleExist)

	recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
		body(adaID.String(), "33.33", graceID.String(), "33.33", alanID.String(), "32.84"))

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusUnprocessableEntity, decoded)
	}
	testutil.AssertFields(t, decoded, map[string]any{
		"code":       float64(http.StatusUnprocessableEntity),
		"error_code": "shares_must_sum_to_100",
	})

	message, _ := decoded["message"].(string)
	if !strings.Contains(message, "99.50") {
		t.Errorf("message = %q, want it to quote the sum that was received (99.50)", message)
	}
	if !strings.Contains(message, "100.00") {
		t.Errorf("message = %q, want it to name the total a share set must reach", message)
	}
}

func TestAShareSetIsRejectedWhenItBreaksARule(t *testing.T) {
	tests := []struct {
		name          string
		setup         func(*mocks.MockRepository)
		body          *strings.Reader
		wantErrorCode string
		wantInMessage string
	}{
		{
			name:          "an empty share set sums to nothing",
			setup:         allPeopleExist,
			body:          body(),
			wantErrorCode: "shares_must_sum_to_100",
			wantInMessage: "0.00",
		},
		{
			name:          "the percentages total more than the whole bill",
			setup:         allPeopleExist,
			body:          body(adaID.String(), "60.00", graceID.String(), "60.00"),
			wantErrorCode: "shares_must_sum_to_100",
			wantInMessage: "120.00",
		},
		{
			name:          "the same person appears twice",
			setup:         allPeopleExist,
			body:          body(adaID.String(), "50.00", adaID.String(), "50.00"),
			wantErrorCode: "duplicate_person",
			wantInMessage: "",
		},
		{
			name: "a person who does not exist",
			setup: func(repository *mocks.MockRepository) {
				// Only Ada comes back, so Grace is unknown.
				repository.EXPECT().ExistingPeople(gomock.Any(), gomock.Any()).
					Return([]uuid.UUID{adaID}, nil).AnyTimes()
			},
			body:          body(adaID.String(), "50.00", graceID.String(), "50.00"),
			wantErrorCode: "unknown_person",
			wantInMessage: "",
		},
		{
			// Sums to exactly 100.00, so the sum rule alone would let it through.
			name:          "a negative percentage hiding in a set that still sums to 100",
			setup:         allPeopleExist,
			body:          body(adaID.String(), "100.00", graceID.String(), "10.00", alanID.String(), "-10.00"),
			wantErrorCode: "invalid_percentage",
			wantInMessage: "-10.00",
		},
		{
			name:          "a single share claiming more than the whole bill",
			setup:         allPeopleExist,
			body:          body(adaID.String(), "110.00"),
			wantErrorCode: "invalid_percentage",
			wantInMessage: "110.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, decoded := testutil.Do(t, newRouter(t, tt.setup), http.MethodPut, shareSetPath(billID), tt.body)

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusUnprocessableEntity, decoded)
			}
			testutil.AssertFields(t, decoded, map[string]any{"error_code": tt.wantErrorCode})

			if tt.wantInMessage != "" {
				message, _ := decoded["message"].(string)
				if !strings.Contains(message, tt.wantInMessage) {
					t.Errorf("message = %q, want it to quote %q", message, tt.wantInMessage)
				}
			}
		})
	}
}

// 400 means the request could not be read; 422 means it was read and refused.
func TestMalformedInputIsDistinguishableFromARuleBeingBroken(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "not JSON at all", body: `{`},
		{name: "shares is not an array", body: `{"shares":"33.33"}`},
		{name: "a percentage that is not a number", body: `{"shares":[{"person_id":"` + adaID.String() + `","percentage":"thirty"}]}`},
		{name: "a percentage carrying more places than a percentage has", body: `{"shares":[{"person_id":"` + adaID.String() + `","percentage":"33.333"}]}`},
		{name: "a person id that is not an identifier", body: `{"shares":[{"person_id":"nobody","percentage":"100.00"}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No expectations: nothing malformed should reach storage at all.
			recorder, decoded := testutil.Do(t, newRouter(t, nil), http.MethodPut, shareSetPath(billID), strings.NewReader(tt.body))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusBadRequest, decoded)
			}
			testutil.AssertFields(t, decoded, map[string]any{"error_code": "invalid_request_body"})
		})
	}
}

// The same envelope GET answers with, so a client need not read the Bill back.
func TestAValidShareSetIsStoredAndAnsweredWithTheBill(t *testing.T) {
	router := newRouter(t, storesWhatItIsGiven)

	recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
		body(adaID.String(), "33.33", graceID.String(), "33.33", alanID.String(), "33.34"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
	}

	testutil.AssertFieldSet(t, decoded, "bill", "shares")
	testutil.AssertFieldSet(t, testutil.Object(t, decoded, "bill"), "id", "description", "total", "currency", "version")

	// The whole set, with the Person names the request did not send.
	wantShares := []map[string]any{
		{"person_id": adaID.String(), "name": "Ada Lovelace", "percentage": "33.33"},
		{"person_id": graceID.String(), "name": "Grace Hopper", "percentage": "33.33"},
		{"person_id": alanID.String(), "name": "Alan Turing", "percentage": "33.34"},
	}
	shares := testutil.Array(t, decoded, "shares")
	if len(shares) != len(wantShares) {
		t.Fatalf("answered with %d shares, want %d", len(shares), len(wantShares))
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

// 0% records that someone was there without charging them.
func TestAShareOfNothingIsAccepted(t *testing.T) {
	recorder, decoded := testutil.Do(t, newRouter(t, storesWhatItIsGiven), http.MethodPut, shareSetPath(billID),
		body(adaID.String(), "100.00", graceID.String(), "0.00"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
	}

	// Kept, not dropped: dropping it would quietly remove them.
	shares := testutil.Array(t, decoded, "shares")
	if len(shares) != 2 {
		t.Fatalf("answered with %d shares, want the 0%% share to be kept", len(shares))
	}
	testutil.AssertFields(t, shares[1].(map[string]any), map[string]any{
		"person_id":  graceID.String(),
		"percentage": "0.00",
	})
}

func TestReplacingTheShareSetOfABillThatIsNotThere(t *testing.T) {
	router := newRouter(t, func(repository *mocks.MockRepository) {
		allPeopleExist(repository)
		repository.EXPECT().ReplaceShareSet(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return(billentity.Bill{}, billentity.ErrNotFound)
	})

	recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
		body(adaID.String(), "100.00"))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusNotFound, decoded)
	}
	testutil.AssertFields(t, decoded, map[string]any{"error_code": "bill_not_found"})
}
