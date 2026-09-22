package person_test

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	personcontroller "github.com/nachoconques0/splitter_svc/backend/internal/controller/person"
	personentity "github.com/nachoconques0/splitter_svc/backend/internal/entity/person"
	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	httpmocks "github.com/nachoconques0/splitter_svc/backend/internal/http/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/model"
	personservice "github.com/nachoconques0/splitter_svc/backend/internal/service/person"
	"github.com/nachoconques0/splitter_svc/backend/internal/service/person/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

var (
	adaID       = uuid.MustParse("a0000000-0000-4000-8000-000000000001")
	katherineID = uuid.MustParse("a0000000-0000-4000-8000-000000000004")
)

// The real router and service, mocking only the repository. The Bill routes
// carry no expectations, so reaching them fails the test.
func newRouter(t *testing.T, setup func(*mocks.MockRepository)) http.Handler {
	t.Helper()

	controller := gomock.NewController(t)
	repository := mocks.NewMockRepository(controller)
	if setup != nil {
		setup(repository)
	}

	logger := slog.New(slog.DiscardHandler)
	return splitterhttp.NewRouter(
		logger,
		httpmocks.NewMockPinger(controller),
		httpmocks.NewMockBillController(controller),
		personcontroller.New(logger, personservice.New(repository)),
	)
}

// Encoded properly, so a name carrying a tab stays a test about trimming.
func createBody(t *testing.T, name string) *strings.Reader {
	t.Helper()

	encoded, err := json.Marshal(model.CreatePersonRequest{Name: name})
	if err != nil {
		t.Fatalf("encoding the request body: %v", err)
	}
	return strings.NewReader(string(encoded))
}

func TestAPersonIsCreatedByName(t *testing.T) {
	created := personentity.Person{ID: katherineID, Name: "Katherine Johnson"}

	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().Create(gomock.Any(), "Katherine Johnson").Return(created, nil)
	})

	recorder, decoded := testutil.Do(t, router, http.MethodPost, "/people", createBody(t, "Katherine Johnson"))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusCreated, decoded)
	}
	testutil.AssertFieldSet(t, decoded, "id", "name")
	testutil.AssertFields(t, decoded, map[string]any{
		"id":   katherineID.String(),
		"name": "Katherine Johnson",
	})
}

// Stated as the name storage is asked for, so the response shows what was stored
// without the test reaching past the seam.
func TestANameIsTrimmedBeforeItIsStored(t *testing.T) {
	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().Create(gomock.Any(), "Katherine Johnson").
			DoAndReturn(func(_ any, name string) (personentity.Person, error) {
				return personentity.Person{ID: katherineID, Name: name}, nil
			})
	})

	recorder, decoded := testutil.Do(t, router, http.MethodPost, "/people", createBody(t, "  Katherine Johnson\t"))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusCreated, decoded)
	}
	testutil.AssertFields(t, decoded, map[string]any{"name": "Katherine Johnson"})
}

func TestANameThatIsNotAName(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "empty", body: `{"name":""}`},
		{name: "spaces", body: `{"name":"   "}`},
		{name: "a tab and a newline", body: `{"name":"\t\n"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No expectations: a name that is not a name never reaches storage.
			recorder, decoded := testutil.Do(t, newRouter(t, nil), http.MethodPost, "/people", strings.NewReader(tt.body))

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusUnprocessableEntity, decoded)
			}
			testutil.AssertFieldSet(t, decoded, "code", "error_code", "message", "detail")
			testutil.AssertFields(t, decoded, map[string]any{
				"code":       float64(http.StatusUnprocessableEntity),
				"error_code": "invalid_name",
				// Which field is at fault, so a client acts on the code and the
				// detail and never has to read the message.
				"detail": "name",
			})
		})
	}
}

func TestACreateBodyThatCouldNotBeRead(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "not JSON at all", body: `{`},
		{name: "a name that is not text", body: `{"name":42}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, decoded := testutil.Do(t, newRouter(t, nil), http.MethodPost, "/people", strings.NewReader(tt.body))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusBadRequest, decoded)
			}
			testutil.AssertFields(t, decoded, map[string]any{"error_code": "invalid_request_body"})
		})
	}
}

func TestThePeopleWhoExistCanBeListed(t *testing.T) {
	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().List(gomock.Any()).Return([]personentity.Person{
			{ID: adaID, Name: "Ada Lovelace"},
			{ID: katherineID, Name: "Katherine Johnson"},
		}, nil)
	})

	recorder, decoded := testutil.Do(t, router, http.MethodGet, "/people", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body %v)", recorder.Code, http.StatusOK, decoded)
	}
	testutil.AssertFieldSet(t, decoded, "people")

	people := testutil.Array(t, decoded, "people")
	if len(people) != 2 {
		t.Fatalf("answered with %d people, want 2", len(people))
	}
	for i, want := range []map[string]any{
		{"id": adaID.String(), "name": "Ada Lovelace"},
		{"id": katherineID.String(), "name": "Katherine Johnson"},
	} {
		entry, ok := people[i].(map[string]any)
		if !ok {
			t.Fatalf("people[%d] = %#v, want an object", i, people[i])
		}
		testutil.AssertFieldSet(t, entry, "id", "name")
		testutil.AssertFields(t, entry, want)
	}
}

func TestListingPeopleWhenThereAreNone(t *testing.T) {
	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().List(gomock.Any()).Return(nil, nil)
	})

	_, decoded := testutil.Do(t, router, http.MethodGet, "/people", nil)

	if people := testutil.Array(t, decoded, "people"); len(people) != 0 {
		t.Errorf("people = %#v, want an empty array", people)
	}
}

func TestAnUnexpectedRepositoryFailureDoesNotLeak(t *testing.T) {
	router := newRouter(t, func(repository *mocks.MockRepository) {
		repository.EXPECT().List(gomock.Any()).Return(nil, errors.New(`pq: relation "people" does not exist`))
	})

	recorder, decoded := testutil.Do(t, router, http.MethodGet, "/people", nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	testutil.AssertFields(t, decoded, map[string]any{"error_code": "internal_error"})
	if leaked := recorder.Body.String(); strings.Contains(leaked, "pq:") || strings.Contains(leaked, "relation") {
		t.Errorf("the driver error reached the client: %s", leaked)
	}
}
