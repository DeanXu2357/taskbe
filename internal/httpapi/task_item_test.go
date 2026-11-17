package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

func TestGetTask(t *testing.T) {
	var gotID int64
	h := newAPI(&fakeService{
		get: func(_ context.Context, id int64) (task.Task, error) {
			gotID = id
			return task.Task{ID: 1, Name: "買早餐", Status: task.StatusCompleted, CreatedAt: t0, UpdatedAt: t0}, nil
		},
	})

	rec := do(h, http.MethodGet, "/tasks/1", "")

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
		`{"data":{"id":1,"name":"買早餐","status":1,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:00Z"}}`)
}

func TestGetTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNotFound, http.StatusNotFound, "not_found"},
		{task.ErrNotImplemented, http.StatusNotImplemented, "not_implemented"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			h := newAPI(&fakeService{
				get: func(context.Context, int64) (task.Task, error) {
					return task.Task{}, err
				},
			})

			rec := do(h, http.MethodGet, "/tasks/1", "")

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestReplaceTask(t *testing.T) {
	var gotID int64
	var got service.ReplaceInput
	h := newAPI(&fakeService{
		replace: func(_ context.Context, id int64, in service.ReplaceInput) (task.Task, error) {
			gotID, got = id, in
			return task.Task{ID: 1, Name: "買午餐", Status: task.StatusCompleted, CreatedAt: t0, UpdatedAt: t0.Add(time.Second)}, nil
		},
	})

	rec := do(h, http.MethodPut, "/tasks/1", `{"name":"買午餐","status":1}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotID != 1 {
		t.Errorf("service got id %d, want 1", gotID)
	}
	name, one := "買午餐", task.StatusCompleted
	if !equalStringPtr(got.Name, &name) || !equalStatusPtr(got.Status, &one) {
		t.Errorf("service got Name=%v Status=%v, want 買午餐 and 1", got.Name, got.Status)
	}
	assertBody(t, rec.Body.String(),
		`{"data":{"id":1,"name":"買午餐","status":1,"created_at":"2026-11-17T08:00:00Z","updated_at":"2026-11-17T08:00:01Z"}}`)
}

func TestReplaceTaskBodyID(t *testing.T) {
	cases := []struct {
		body       string
		wantStatus int
		wantCalled bool
	}{
		{`{"id":1,"name":"a","status":1}`, http.StatusOK, true},
		{`{"id":2,"name":"a","status":1}`, http.StatusBadRequest, false},
		{`{"name":"a","status":1,"created_at":"1999-01-01T00:00:00Z"}`, http.StatusOK, true},
	}
	for _, tc := range cases {
		called := false
		h := newAPI(&fakeService{
			replace: func(context.Context, int64, service.ReplaceInput) (task.Task, error) {
				called = true
				return task.Task{ID: 1, Name: "a", Status: task.StatusCompleted, CreatedAt: t0, UpdatedAt: t0}, nil
			},
		})

		rec := do(h, http.MethodPut, "/tasks/1", tc.body)

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

func TestReplaceTaskDecodesInput(t *testing.T) {
	name, one := "a", task.StatusCompleted
	cases := []struct {
		body       string
		wantName   *string
		wantStatus *task.Status
	}{
		{`{"name":"a"}`, &name, nil},
		{`{"status":1}`, nil, &one},
	}
	for _, tc := range cases {
		var got service.ReplaceInput
		h := newAPI(&fakeService{
			replace: func(_ context.Context, _ int64, in service.ReplaceInput) (task.Task, error) {
				got = in
				return task.Task{ID: 1}, nil
			},
		})

		do(h, http.MethodPut, "/tasks/1", tc.body)

		if !equalStringPtr(got.Name, tc.wantName) || !equalStatusPtr(got.Status, tc.wantStatus) {
			t.Errorf("%s: service got Name=%v Status=%v", tc.body, got.Name, got.Status)
		}
	}
}

func TestReplaceTaskRejectsBody(t *testing.T) {
	called := false
	h := newAPI(&fakeService{
		replace: func(context.Context, int64, service.ReplaceInput) (task.Task, error) {
			called = true
			return task.Task{}, nil
		},
	})

	rec := do(h, http.MethodPut, "/tasks/1", "{")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if code := decodeError(t, rec).Error.Code; code != "invalid_json" {
		t.Errorf("code = %q, want invalid_json", code)
	}
	if called {
		t.Error("service was called")
	}
}

func TestReplaceTaskServiceErrors(t *testing.T) {
	cases := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{task.ErrNameRequired, http.StatusBadRequest, "validation_failed"},
		{task.ErrNameEmpty, http.StatusBadRequest, "validation_failed"},
		{task.ErrNameTooLong, http.StatusBadRequest, "validation_failed"},
		{task.ErrStatusRequired, http.StatusBadRequest, "validation_failed"},
		{task.ErrInvalidStatus, http.StatusBadRequest, "validation_failed"},
		{task.ErrNotFound, http.StatusNotFound, "not_found"},
		{task.ErrNotImplemented, http.StatusNotImplemented, "not_implemented"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			h := newAPI(&fakeService{
				replace: func(context.Context, int64, service.ReplaceInput) (task.Task, error) {
					return task.Task{}, err
				},
			})

			rec := do(h, http.MethodPut, "/tasks/1", `{"name":"a","status":1}`)

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}

func TestDeleteTask(t *testing.T) {
	var gotID int64
	h := newAPI(&fakeService{
		delete: func(_ context.Context, id int64) error {
			gotID = id
			return nil
		},
	})

	rec := do(h, http.MethodDelete, "/tasks/1", "")

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
		{task.ErrNotImplemented, http.StatusNotImplemented, "not_implemented"},
		{errors.New("boom"), http.StatusInternalServerError, "internal_error"},
	}
	for _, tc := range cases {
		err := tc.err
		t.Run(err.Error(), func(t *testing.T) {
			h := newAPI(&fakeService{
				delete: func(context.Context, int64) error {
					return err
				},
			})

			rec := do(h, http.MethodDelete, "/tasks/1", "")

			assertServiceError(t, rec, tc.wantStatus, tc.wantCode)
		})
	}
}
