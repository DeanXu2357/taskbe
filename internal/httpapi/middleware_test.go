package httpapi_test

import (
	"bytes"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
	"github.com/DeanXu2357/taskbe/internal/logx"
)

func TestRecoverPanics(t *testing.T) {
	h := httpapi.RecoverPanics(discardLogger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := do(h, http.MethodGet, "/tasks", "")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if code := decodeError(t, rec).Error.Code; code != "internal_error" {
		t.Errorf("code = %q, want internal_error", code)
	}
}

func TestLogRequests(t *testing.T) {
	var buf bytes.Buffer
	h := httpapi.LogRequests(logx.New(log.New(&buf, "", 0)), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	do(h, http.MethodGet, "/tasks", "")

	for _, want := range []string{"GET", "/tasks", "418"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("log %q does not contain %q", buf.String(), want)
		}
	}
}
