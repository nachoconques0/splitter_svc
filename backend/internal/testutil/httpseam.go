// Only _test packages import this, so it is never linked into the server.
package testutil

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// Do sends a request through the router and decodes the JSON object it answers with.
func Do(t testing.TB, router http.Handler, method, target string, body io.Reader) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()

	request := httptest.NewRequest(method, target, body)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	raw, err := io.ReadAll(recorder.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("response body is not a JSON object: %v (body %q)", err, raw)
	}
	return recorder, decoded
}

// AssertFieldSet compares the whole field set, so the wire contract cannot grow
// one unnoticed.
func AssertFieldSet(t testing.TB, got map[string]any, want ...string) {
	t.Helper()

	gotKeys, wantKeys := slices.Sorted(maps.Keys(got)), slices.Sorted(slices.Values(want))
	if !slices.Equal(gotKeys, wantKeys) {
		t.Errorf("field set = %v, want %v", gotKeys, wantKeys)
	}
}

// AssertFields checks the values a test names.
func AssertFields(t testing.TB, got map[string]any, want map[string]any) {
	t.Helper()

	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("field %q = %#v, want %#v", key, got[key], wantValue)
		}
	}
}

// Object reads a nested JSON object out of a decoded body.
func Object(t testing.TB, body map[string]any, field string) map[string]any {
	t.Helper()

	object, ok := body[field].(map[string]any)
	if !ok {
		t.Fatalf("%s = %#v, want an object", field, body[field])
	}
	return object
}

// Array reads a nested JSON array out of a decoded body.
func Array(t testing.TB, body map[string]any, field string) []any {
	t.Helper()

	array, ok := body[field].([]any)
	if !ok {
		t.Fatalf("%s = %#v, want an array", field, body[field])
	}
	return array
}
