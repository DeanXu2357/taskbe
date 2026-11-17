package httpapi_test

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
	"github.com/DeanXu2357/taskbe/internal/logx"
	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

var t0 = time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)

var discardLogger = logx.New(log.New(io.Discard, "", 0))

func assertBody(t *testing.T, got, want string) {
	t.Helper()
	if got := strings.TrimSpace(got); got != want {
		t.Errorf("body:\n got %s\nwant %s", got, want)
	}
}

func TestCreateTask(t *testing.T) {
	var got service.CreateInput
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		create: func(_ context.Context, in service.CreateInput) (task.Task, error) {
			got = in
			return task.Task{ID: 1, Name: "買早餐", Status: task.StatusIncomplete, Version: 1, CreatedAt: t0, UpdatedAt: t0}, nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodPost, "/tasks", `{"name":"買早餐"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/tasks/1" {
		t.Errorf("Location = %q, want /tasks/1", loc)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	assertBody(t, rec.Body.String(),
		`{"data":{"id":1,"name":"買早餐","status":0,"version":1,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:00Z"}}`)
	if got.Name == nil || *got.Name != "買早餐" || got.Status != nil {
		t.Errorf("service got Name=%v Status=%v, want 買早餐 and nil", got.Name, got.Status)
	}
}

func TestCreateTaskDecodesInput(t *testing.T) {
	one := task.StatusCompleted
	name := "a"
	cases := []struct {
		body       string
		wantName   *string
		wantStatus *task.Status
	}{
		{`{"name":"a","status":1}`, &name, &one},
		{`{"name":"a","status":null}`, &name, nil},
		{`{"name":"a","foo":1}`, &name, nil},
		{`null`, nil, nil},
	}
	for _, tc := range cases {
		var got service.CreateInput
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			create: func(_ context.Context, in service.CreateInput) (task.Task, error) {
				got = in
				return task.Task{ID: 1}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodPost, "/tasks", tc.body)

		if rec.Code != http.StatusCreated {
			t.Errorf("%s: status = %d, want 201", tc.body, rec.Code)
			continue
		}
		if !equalPtr(got.Name, tc.wantName) || !equalPtr(got.Status, tc.wantStatus) {
			t.Errorf("%s: service got Name=%v Status=%v", tc.body, got.Name, got.Status)
		}
	}
}

func TestCreateTaskRejectsBody(t *testing.T) {
	cases := []struct {
		body       string
		wantStatus int
		wantCode   string
	}{
		{``, http.StatusBadRequest, "invalid_json"},
		{`{`, http.StatusBadRequest, "invalid_json"},
		{`[]`, http.StatusBadRequest, "invalid_json"},
		{`{"name":"a"} {"name":"b"}`, http.StatusBadRequest, "invalid_json"},
		{`{"name":123}`, http.StatusBadRequest, "invalid_json"},
		{`{"name":"a","status":"1"}`, http.StatusBadRequest, "invalid_json"},
		{`{"name":"a","status":1.5}`, http.StatusBadRequest, "invalid_json"},
		{`{"name":"a","status":true}`, http.StatusBadRequest, "invalid_json"},
		{`{"name":"` + strings.Repeat("a", 1<<20) + `"}`, http.StatusRequestEntityTooLarge, "payload_too_large"},
	}
	for _, tc := range cases {
		called := false
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			create: func(context.Context, service.CreateInput) (task.Task, error) {
				called = true
				return task.Task{}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodPost, "/tasks", tc.body)

		label := tc.body
		if len(label) > 40 {
			label = label[:40] + "..."
		}
		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", label, rec.Code, tc.wantStatus)
			continue
		}
		body := decodeError(t, rec)
		if body.Error.Code != tc.wantCode {
			t.Errorf("%s: code = %q, want %q", label, body.Error.Code, tc.wantCode)
		}
		msg := body.Error.Message
		if msg == "" || strings.Contains(msg, "Go struct") || strings.Contains(msg, "task.") || strings.Contains(msg, "service.") {
			t.Errorf("%s: message %q is empty or leaks Go type names", label, msg)
		}
		if called {
			t.Errorf("%s: service was called", label)
		}
	}
}

func TestCreateTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNameRequired, http.StatusBadRequest, "validation_failed"},
		{task.ErrNameEmpty, http.StatusBadRequest, "validation_failed"},
		{task.ErrNameTooLong, http.StatusBadRequest, "validation_failed"},
		{task.ErrInvalidStatus, http.StatusBadRequest, "validation_failed"},
		{task.ErrNotFound, http.StatusInternalServerError, "internal_error"},
		{fmt.Errorf("%w: boom", task.ErrUnexpected), http.StatusInternalServerError, "internal_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			mux := http.NewServeMux()
			httpapi.Register(mux, &fakeService{
				create: func(context.Context, service.CreateInput) (task.Task, error) {
					return task.Task{}, err
				},
			}, discardLogger)

			rec := do(mux, http.MethodPost, "/tasks", `{"name":"a"}`)

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestListTasksServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrInvalidLimit, http.StatusBadRequest, "validation_failed"},
		{task.ErrInvalidCursor, http.StatusBadRequest, "validation_failed"},
		{fmt.Errorf("%w: boom", task.ErrUnexpected), http.StatusInternalServerError, "internal_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			mux := http.NewServeMux()
			httpapi.Register(mux, &fakeService{
				list: func(context.Context, service.ListParams) (service.Page, error) {
					return service.Page{}, err
				},
			}, discardLogger)

			rec := do(mux, http.MethodGet, "/tasks", "")

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func assertServiceError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d", rec.Code, wantStatus)
	}
	if code := decodeError(t, rec).Error.Code; code != wantCode {
		t.Errorf("code = %q, want %q", code, wantCode)
	}
}

func TestListTasks(t *testing.T) {
	var got service.ListParams
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		list: func(_ context.Context, p service.ListParams) (service.Page, error) {
			got = p
			return service.Page{
				Tasks: []task.Task{
					{ID: 1, Name: "買早餐", Status: task.StatusIncomplete, Version: 1, CreatedAt: t0, UpdatedAt: t0},
					{ID: 2, Name: "買午餐", Status: task.StatusCompleted, Version: 2, CreatedAt: t0, UpdatedAt: t0.Add(time.Hour)},
				},
				HasMore: true,
			}, nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodGet, "/tasks", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got != (service.ListParams{AfterID: 0, Limit: 20}) {
		t.Errorf("service got %+v, want {AfterID:0 Limit:20}", got)
	}
	assertBody(t, rec.Body.String(),
		`{"data":[`+
			`{"id":1,"name":"買早餐","status":0,"version":1,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:00Z"},`+
			`{"id":2,"name":"買午餐","status":1,"version":2,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T09:00:00Z"}`+
			`],"next_cursor":"eyJhZnRlcl9pZCI6Mn0"}`)
}

func TestListLastPage(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		list: func(context.Context, service.ListParams) (service.Page, error) {
			return service.Page{Tasks: nil, HasMore: false}, nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodGet, "/tasks", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	assertBody(t, rec.Body.String(), `{"data":[],"next_cursor":null}`)
}

func TestListParsesQuery(t *testing.T) {
	cases := []struct {
		target string
		want   service.ListParams
	}{
		{"/tasks?limit=5&cursor=eyJhZnRlcl9pZCI6Mn0", service.ListParams{AfterID: 2, Limit: 5}},
		{"/tasks?limit=101", service.ListParams{AfterID: 0, Limit: 101}},
	}
	for _, tc := range cases {
		var got service.ListParams
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			list: func(_ context.Context, p service.ListParams) (service.Page, error) {
				got = p
				return service.Page{}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodGet, tc.target, "")

		if rec.Code != http.StatusOK {
			t.Errorf("%s: status = %d, want 200", tc.target, rec.Code)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: service got %+v, want %+v", tc.target, got, tc.want)
		}
	}
}

func TestListRejectsQuery(t *testing.T) {
	b64 := base64.RawURLEncoding.EncodeToString
	queries := []string{
		"limit=abc",
		"cursor=!!!",
		"cursor=" + b64([]byte("not json")),
		"cursor=" + b64([]byte(`{"after_id":-1}`)),
		"cursor=" + b64([]byte(`{"after_id":0}`)),
	}
	for _, q := range queries {
		called := false
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			list: func(context.Context, service.ListParams) (service.Page, error) {
				called = true
				return service.Page{}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodGet, "/tasks?"+q, "")

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", q, rec.Code)
			continue
		}
		if code := decodeError(t, rec).Error.Code; code != "validation_failed" {
			t.Errorf("%s: code = %q, want validation_failed", q, code)
		}
		if called {
			t.Errorf("%s: service was called", q)
		}
	}
}

func equalPtr[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
