package http_test

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"

	splitterhttp "github.com/nachoconques0/splitter_svc/backend/internal/http"
	"github.com/nachoconques0/splitter_svc/backend/internal/http/mocks"
	"github.com/nachoconques0/splitter_svc/backend/internal/testutil"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// The mocked routes carry no expectations, so reaching them fails the test.
func newRouter(t *testing.T, setup func(*mocks.MockPinger)) http.Handler {
	t.Helper()

	controller := gomock.NewController(t)
	pinger := mocks.NewMockPinger(controller)
	if setup != nil {
		setup(pinger)
	}

	return splitterhttp.NewRouter(slog.New(slog.DiscardHandler), pinger, mocks.NewMockBillController(controller), mocks.NewMockPersonController(controller))
}

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*mocks.MockPinger)
		wantStatus int
		wantBody   map[string]any
	}{
		{
			name: "reports up when the database answers",
			setup: func(pinger *mocks.MockPinger) {
				pinger.EXPECT().PingContext(gomock.Any()).Return(nil).AnyTimes()
			},
			wantStatus: http.StatusOK,
			wantBody: map[string]any{
				"status":   "ok",
				"database": "ok",
			},
		},
		{
			// Asserting the body exactly is what keeps the driver's error off the wire.
			name: "reports unavailable when the database does not answer",
			setup: func(pinger *mocks.MockPinger) {
				pinger.EXPECT().PingContext(gomock.Any()).Return(errors.New("dial tcp 172.18.0.2:5432: connect: connection refused")).AnyTimes()
			},
			wantStatus: http.StatusServiceUnavailable,
			wantBody: map[string]any{
				"code":       float64(http.StatusServiceUnavailable),
				"error_code": "database_unavailable",
				"message":    "the database is not reachable",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, body := testutil.Do(t, newRouter(t, tt.setup), http.MethodGet, "/health", nil)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body %v)", recorder.Code, tt.wantStatus, body)
			}
			testutil.AssertFieldSet(t, body, keysOf(tt.wantBody)...)
			testutil.AssertFields(t, body, tt.wantBody)
		})
	}
}

func TestUnknownRouteUsesTheErrorEnvelope(t *testing.T) {
	tests := []struct {
		name          string
		method        string
		target        string
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "an unknown path",
			method:        http.MethodGet,
			target:        "/nope",
			wantStatus:    http.StatusNotFound,
			wantErrorCode: "not_found",
		},
		{
			name:          "a known path with the wrong method",
			method:        http.MethodPost,
			target:        "/health",
			wantStatus:    http.StatusMethodNotAllowed,
			wantErrorCode: "method_not_allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder, body := testutil.Do(t, newRouter(t, nil), tt.method, tt.target, nil)

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			testutil.AssertFieldSet(t, body, "code", "error_code", "message")
			testutil.AssertFields(t, body, map[string]any{
				"code":       float64(tt.wantStatus),
				"error_code": tt.wantErrorCode,
			})
			// Prose, so asserted to exist rather than to read a particular way.
			if message, ok := body["message"].(string); !ok || message == "" {
				t.Errorf("error envelope carries no message: %#v", body["message"])
			}
		})
	}
}

// The one path that reaches the client without a handler choosing the response.
// The route is added to the real engine, so the middleware is the installed one.
func TestAPanicStillWearsTheErrorEnvelope(t *testing.T) {
	router := newRouter(t, nil).(*gin.Engine)
	router.GET("/boom", func(*gin.Context) {
		panic("a handler did something unforgivable")
	})

	recorder, body := testutil.Do(t, router, http.MethodGet, "/boom", nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	testutil.AssertFieldSet(t, body, "code", "error_code", "message")
	testutil.AssertFields(t, body, map[string]any{
		"code":       float64(http.StatusInternalServerError),
		"error_code": "internal_error",
		"message":    "something went wrong",
	})
	if strings.Contains(recorder.Body.String(), "unforgivable") {
		t.Errorf("the panic value reached the client: %s", recorder.Body.String())
	}
}

func keysOf(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
