package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
)

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func do(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{}, discardLogger)
	cases := []struct{ method, target, allow string }{
		{http.MethodPut, "/tasks", "GET, POST"},
		{http.MethodDelete, "/tasks", "GET, POST"},
		{http.MethodPatch, "/tasks", "GET, POST"},
		{http.MethodPost, "/tasks/1", "GET, PUT, DELETE"},
		{http.MethodPatch, "/tasks/1", "GET, PUT, DELETE"},
	}
	for _, tc := range cases {
		rec := do(mux, tc.method, tc.target, "")
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want 405", tc.method, tc.target, rec.Code)
			continue
		}
		if got := rec.Header().Get("Allow"); got != tc.allow {
			t.Errorf("%s %s: Allow = %q, want %q", tc.method, tc.target, got, tc.allow)
		}
		if code := decodeError(t, rec).Error.Code; code != "method_not_allowed" {
			t.Errorf("%s %s: code = %q, want method_not_allowed", tc.method, tc.target, code)
		}
	}
}

func TestInvalidTaskPaths(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{}, discardLogger)
	paths := []string{
		"/tasks/",
		"/tasks/abc",
		"/tasks/0",
		"/tasks/-1",
		"/tasks/01",
		"/tasks/+1",
		"/tasks/1/",
		"/tasks/1/x",
		"/tasks/99999999999999999999",
	}
	for _, p := range paths {
		rec := do(mux, http.MethodGet, p, "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404", p, rec.Code)
			continue
		}
		if code := decodeError(t, rec).Error.Code; code != "not_found" {
			t.Errorf("GET %s: code = %q, want not_found", p, code)
		}
	}
}

func TestUnknownPath(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{}, discardLogger)
	for _, p := range []string{"/", "/foo"} {
		rec := do(mux, http.MethodGet, p, "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404", p, rec.Code)
			continue
		}
		if code := decodeError(t, rec).Error.Code; code != "not_found" {
			t.Errorf("GET %s: code = %q, want not_found", p, code)
		}
	}
}
