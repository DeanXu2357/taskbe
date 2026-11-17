package httpapi_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

func TestGetTask(t *testing.T) {
	var gotID int64
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		get: func(_ context.Context, id int64) (task.Task, error) {
			gotID = id
			return task.Task{ID: 1, Name: "買早餐", Status: task.StatusCompleted, Version: 1, CreatedAt: t0, UpdatedAt: t0}, nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodGet, "/tasks/1", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotID != 1 {
		t.Errorf("service got id %d, want 1", gotID)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	assertBody(t, rec.Body.String(),
		`{"data":{"id":1,"name":"買早餐","status":1,"version":1,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:00Z"}}`)
}

func TestGetTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNotFound, http.StatusNotFound, "not_found"},
		{fmt.Errorf("%w: boom", task.ErrUnexpected), http.StatusInternalServerError, "internal_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			mux := http.NewServeMux()
			httpapi.Register(mux, &fakeService{
				get: func(context.Context, int64) (task.Task, error) {
					return task.Task{}, err
				},
			}, discardLogger)

			rec := do(mux, http.MethodGet, "/tasks/1", "")

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestReplaceTask(t *testing.T) {
	var gotID, gotVersion int64
	var got service.ReplaceInput
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		replace: func(_ context.Context, id, expectedVersion int64, in service.ReplaceInput) (task.Task, error) {
			gotID, gotVersion, got = id, expectedVersion, in
			return task.Task{ID: 1, Name: "買午餐", Status: task.StatusCompleted, Version: 4, CreatedAt: t0, UpdatedAt: t0.Add(time.Second)}, nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodPut, "/tasks/1", `{"name":"買午餐","status":1,"version":3}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	want := service.ReplaceInput{Name: "買午餐", Status: task.StatusCompleted}
	if gotID != 1 || gotVersion != 3 || got != want {
		t.Errorf("service got id %d, version %d, input %+v; want 1, 3, %+v", gotID, gotVersion, got, want)
	}
	assertBody(t, rec.Body.String(),
		`{"data":{"id":1,"name":"買午餐","status":1,"version":4,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:01Z"}}`)
}

func TestReplaceTaskBodyID(t *testing.T) {
	cases := []struct {
		body       string
		wantStatus int
		wantCalled bool
	}{
		{`{"id":1,"name":"a","status":1,"version":1}`, http.StatusOK, true},
		{`{"id":2,"name":"a","status":1,"version":1}`, http.StatusBadRequest, false},
		{`{"name":"a","status":1,"version":1,"created_at":"1999-01-01T00:00:00Z"}`, http.StatusOK, true},
	}
	for _, tc := range cases {
		called := false
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			replace: func(context.Context, int64, int64, service.ReplaceInput) (task.Task, error) {
				called = true
				return task.Task{ID: 1, Name: "a", Status: task.StatusCompleted, CreatedAt: t0, UpdatedAt: t0}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodPut, "/tasks/1", tc.body)

		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.body, rec.Code, tc.wantStatus)
			continue
		}
		if called != tc.wantCalled {
			t.Errorf("%s: service called = %v, want %v", tc.body, called, tc.wantCalled)
		}
		if tc.wantStatus == http.StatusBadRequest {
			if code := decodeError(t, rec).Error.Code; code != "validation_failed" {
				t.Errorf("%s: code = %q, want validation_failed", tc.body, code)
			}
		}
	}
}

func TestReplaceTaskRequiresFields(t *testing.T) {
	bodies := []string{
		`{"status":1,"version":1}`,
		`{"name":null,"status":1,"version":1}`,
		`{"name":"a","version":1}`,
		`{"name":"a","status":null,"version":1}`,
		`{"name":"a","status":1}`,
		`{"name":"a","status":1,"version":null}`,
	}
	for _, body := range bodies {
		called := false
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			replace: func(context.Context, int64, int64, service.ReplaceInput) (task.Task, error) {
				called = true
				return task.Task{}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodPut, "/tasks/1", body)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, rec.Code)
			continue
		}
		if code := decodeError(t, rec).Error.Code; code != "validation_failed" {
			t.Errorf("%s: code = %q, want validation_failed", body, code)
		}
		if called {
			t.Errorf("%s: service was called", body)
		}
	}
}

func TestReplaceTaskRejectsBody(t *testing.T) {
	for _, body := range []string{`{`, `null`} {
		called := false
		mux := http.NewServeMux()
		httpapi.Register(mux, &fakeService{
			replace: func(context.Context, int64, int64, service.ReplaceInput) (task.Task, error) {
				called = true
				return task.Task{}, nil
			},
		}, discardLogger)

		rec := do(mux, http.MethodPut, "/tasks/1", body)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", body, rec.Code)
			continue
		}
		if code := decodeError(t, rec).Error.Code; code != "invalid_json" {
			t.Errorf("%s: code = %q, want invalid_json", body, code)
		}
		if called {
			t.Errorf("%s: service was called", body)
		}
	}
}

func TestReplaceTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNameEmpty, http.StatusBadRequest, "validation_failed"},
		{task.ErrNameTooLong, http.StatusBadRequest, "validation_failed"},
		{task.ErrInvalidStatus, http.StatusBadRequest, "validation_failed"},
		{task.ErrVersionConflict, http.StatusConflict, "conflict"},
		{task.ErrNotFound, http.StatusNotFound, "not_found"},
		{fmt.Errorf("%w: boom", task.ErrUnexpected), http.StatusInternalServerError, "internal_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			mux := http.NewServeMux()
			httpapi.Register(mux, &fakeService{
				replace: func(context.Context, int64, int64, service.ReplaceInput) (task.Task, error) {
					return task.Task{}, err
				},
			}, discardLogger)

			rec := do(mux, http.MethodPut, "/tasks/1", `{"name":"a","status":1,"version":1}`)

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestDeleteTask(t *testing.T) {
	var gotID int64
	mux := http.NewServeMux()
	httpapi.Register(mux, &fakeService{
		delete: func(_ context.Context, id int64) error {
			gotID = id
			return nil
		},
	}, discardLogger)

	rec := do(mux, http.MethodDelete, "/tasks/1", "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotID != 1 {
		t.Errorf("service got id %d, want 1", gotID)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestDeleteTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNotFound, http.StatusNotFound, "not_found"},
		{fmt.Errorf("%w: boom", task.ErrUnexpected), http.StatusInternalServerError, "internal_error"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			mux := http.NewServeMux()
			httpapi.Register(mux, &fakeService{
				delete: func(context.Context, int64) error {
					return err
				},
			}, discardLogger)

			rec := do(mux, http.MethodDelete, "/tasks/1", "")

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}
