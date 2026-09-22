package bill_test

import (
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	billentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/bill"
	"github.com/nachoconques0/splitter_svc/backend/internal/service/bill/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

// Two people editing one Share Set would otherwise clobber each other silently.
func TestASaveCarryingAStaleVersionIsRefused(t *testing.T) {
	stored := seededBill()

	// Exactly the Share Set the Bill already holds, for the second case.
	identical := make([]string, 0, len(stored.Shares)*2)
	for _, share := range stored.Shares {
		identical = append(identical, share.PersonID.String(), share.Percentage.Decimal())
	}

	tests := []struct {
		name   string
		shares []string
	}{
		{
			name:   "carrying a different share set",
			shares: []string{adaID.String(), "100.00"},
		},
		{
			// The version alone decides it; payloads are never compared.
			name:   "carrying exactly what is already stored",
			shares: identical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := newRouter(t, func(repository *mocks.MockRepository) {
				allPeopleExist(repository)
				repository.EXPECT().ReplaceShareSet(gomock.Any(), billID, int64(seededVersion), gomock.Any()).
					Return(billentity.Bill{}, billentity.ErrModified)
			})

			recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
				versionedBody(seededVersion, tt.shares...))

			if recorder.Code != http.StatusConflict {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusConflict, decoded)
			}
			testutil.AssertFieldSet(t, decoded, "code", "error_code", "message")
			testutil.AssertFields(t, decoded, map[string]any{
				"code":       float64(http.StatusConflict),
				"error_code": "bill_modified",
			})
		})
	}
}

// Stated as an expectation on the call, so a service that substituted its own
// version would fail here.
func TestTheVersionTheClientSendsIsTheOneTheWriteIsGuardedBy(t *testing.T) {
	const held = 7

	router := newRouter(t, func(repository *mocks.MockRepository) {
		allPeopleExist(repository)
		repository.EXPECT().ReplaceShareSet(gomock.Any(), billID, int64(held), gomock.Any()).
			Return(seededBill(), nil)
	})

	recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
		versionedBody(held, adaID.String(), "100.00"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
	}
}

func TestASaveWithoutAVersionIsRefused(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "no version at all",
			body: `{"shares":[{"person_id":"` + adaID.String() + `","percentage":"100.00"}]}`,
		},
		{
			name: "a version that is not a number",
			body: `{"version":"one","shares":[{"person_id":"` + adaID.String() + `","percentage":"100.00"}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No expectations: the mock fails the test if anything reaches storage.
			recorder, decoded := testutil.Do(t, newRouter(t, nil), http.MethodPut, shareSetPath(billID), strings.NewReader(tt.body))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusBadRequest, decoded)
			}
			testutil.AssertFields(t, decoded, map[string]any{"error_code": "invalid_request_body"})
		})
	}
}

func TestASuccessfulSaveAnswersWithTheVersionToSaveAgainWith(t *testing.T) {
	advanced := seededBill()
	advanced.Version = seededVersion + 1

	router := newRouter(t, func(repository *mocks.MockRepository) {
		allPeopleExist(repository)
		repository.EXPECT().ReplaceShareSet(gomock.Any(), billID, int64(seededVersion), gomock.Any()).
			Return(advanced, nil)
	})

	recorder, decoded := testutil.Do(t, router, http.MethodPut, shareSetPath(billID),
		versionedBody(seededVersion, adaID.String(), "100.00"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
	}
	testutil.AssertFields(t, testutil.Object(t, decoded, "bill"), map[string]any{
		"version": float64(seededVersion + 1),
	})
}
