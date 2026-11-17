package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

var (
	t0 = time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	t1 = t0.Add(time.Hour)
)

func ptr[T any](v T) *T { return &v }

func TestCreateNameRules(t *testing.T) {
	cases := []struct {
		in, want string
		wantErr  error
	}{
		{"買早餐", "買早餐", nil},
		{"  buy milk \t", "buy milk", nil},
		{"", "", task.ErrNameEmpty},
		{"   ", "", task.ErrNameEmpty},
		{strings.Repeat("字", 200), strings.Repeat("字", 200), nil},
		{strings.Repeat("字", 201), "", task.ErrNameTooLong},
		{strings.Repeat("a", 201), "", task.ErrNameTooLong},
	}
	for _, tc := range cases {
		var got task.Task
		called := false
		svc := service.New(&fakeStore{
			create: func(_ context.Context, t task.Task) (task.Task, error) {
				called, got = true, t
				return t, nil
			},
		}, func() time.Time { return t0 })

		_, err := svc.Create(context.Background(), service.CreateInput{Name: ptr(tc.in)})

		if err != tc.wantErr {
			t.Errorf("%q: err = %v, want %v", tc.in, err, tc.wantErr)
			continue
		}
		if tc.wantErr != nil {
			if called {
				t.Errorf("%q: store was called", tc.in)
			}
			continue
		}
		if got.Name != tc.want {
			t.Errorf("%q: store got name %q, want %q", tc.in, got.Name, tc.want)
		}
	}
}

func TestCreateDefaults(t *testing.T) {
	var got task.Task
	stored := task.Task{ID: 7, Name: "買早餐", CreatedAt: t0, UpdatedAt: t0}
	svc := service.New(&fakeStore{
		create: func(_ context.Context, t task.Task) (task.Task, error) {
			got = t
			return stored, nil
		},
	}, func() time.Time { return t0 })

	res, err := svc.Create(context.Background(), service.CreateInput{Name: ptr("  買早餐 ")})

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := task.Task{Name: "買早餐", Status: task.StatusIncomplete, CreatedAt: t0, UpdatedAt: t0}
	if got != want {
		t.Errorf("store got %+v, want %+v", got, want)
	}
	if res != stored {
		t.Errorf("result = %+v, want %+v", res, stored)
	}
}

func TestCreateValidation(t *testing.T) {
	cases := []struct {
		in      service.CreateInput
		wantErr error
	}{
		{service.CreateInput{}, task.ErrNameRequired},
		{service.CreateInput{Name: ptr("   ")}, task.ErrNameEmpty},
		{service.CreateInput{Name: ptr("a"), Status: ptr(task.Status(2))}, task.ErrInvalidStatus},
		{service.CreateInput{Name: ptr("a"), Status: ptr(task.Status(-1))}, task.ErrInvalidStatus},
		{service.CreateInput{Status: ptr(task.Status(2))}, task.ErrNameRequired},
	}
	for i, tc := range cases {
		svc := service.New(&fakeStore{}, func() time.Time { return t0 })

		_, err := svc.Create(context.Background(), tc.in)

		if err != tc.wantErr {
			t.Errorf("case %d: err = %v, want %v", i, err, tc.wantErr)
		}
	}
}

func TestReplace(t *testing.T) {
	var got task.Task
	stored := task.Task{ID: 1, Name: "b", Status: task.StatusCompleted, CreatedAt: t0, UpdatedAt: t1}
	svc := service.New(&fakeStore{
		update: func(_ context.Context, t task.Task) (task.Task, error) {
			got = t
			return stored, nil
		},
	}, func() time.Time { return t1 })

	res, err := svc.Replace(context.Background(), 1, service.ReplaceInput{Name: ptr("b"), Status: ptr(task.StatusCompleted)})

	if err != nil {
		t.Fatalf("err = %v", err)
	}
	want := task.Task{ID: 1, Name: "b", Status: task.StatusCompleted, UpdatedAt: t1}
	if got != want {
		t.Errorf("store got %+v, want %+v", got, want)
	}
	if res != stored {
		t.Errorf("result = %+v, want %+v", res, stored)
	}
}

func TestReplaceValidation(t *testing.T) {
	cases := []struct {
		in      service.ReplaceInput
		wantErr error
	}{
		{service.ReplaceInput{Status: ptr(task.StatusCompleted)}, task.ErrNameRequired},
		{service.ReplaceInput{Name: ptr("a")}, task.ErrStatusRequired},
		{service.ReplaceInput{Name: ptr(" "), Status: ptr(task.StatusCompleted)}, task.ErrNameEmpty},
		{service.ReplaceInput{Name: ptr("a"), Status: ptr(task.Status(2))}, task.ErrInvalidStatus},
	}
	for i, tc := range cases {
		svc := service.New(&fakeStore{}, func() time.Time { return t1 })

		_, err := svc.Replace(context.Background(), 1, tc.in)

		if err != tc.wantErr {
			t.Errorf("case %d: err = %v, want %v", i, err, tc.wantErr)
		}
	}
}

func TestStoreNotFoundIsTranslated(t *testing.T) {
	svc := service.New(&fakeStore{
		get: func(context.Context, int64) (task.Task, error) {
			return task.Task{}, service.ErrRecordNotFound
		},
		update: func(context.Context, task.Task) (task.Task, error) {
			return task.Task{}, service.ErrRecordNotFound
		},
		delete: func(context.Context, int64) error {
			return service.ErrRecordNotFound
		},
	}, func() time.Time { return t1 })
	ctx := context.Background()

	if _, err := svc.Get(ctx, 1); err != task.ErrNotFound {
		t.Errorf("Get err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Replace(ctx, 1, service.ReplaceInput{Name: ptr("a"), Status: ptr(task.StatusCompleted)}); err != task.ErrNotFound {
		t.Errorf("Replace err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, 1); err != task.ErrNotFound {
		t.Errorf("Delete err = %v, want ErrNotFound", err)
	}
}

func TestStoreFailuresAreWrapped(t *testing.T) {
	boom := errors.New("boom")
	svc := service.New(&fakeStore{
		list: func(context.Context, int64, int) ([]task.Task, error) {
			return nil, boom
		},
		get: func(context.Context, int64) (task.Task, error) {
			return task.Task{}, boom
		},
		create: func(context.Context, task.Task) (task.Task, error) {
			return task.Task{}, boom
		},
		update: func(context.Context, task.Task) (task.Task, error) {
			return task.Task{}, boom
		},
		delete: func(context.Context, int64) error {
			return boom
		},
	}, func() time.Time { return t1 })
	ctx := context.Background()
	valid := service.ReplaceInput{Name: ptr("a"), Status: ptr(task.StatusCompleted)}

	_, listErr := svc.List(ctx, service.ListParams{Limit: 20})
	_, getErr := svc.Get(ctx, 1)
	_, createErr := svc.Create(ctx, service.CreateInput{Name: ptr("a")})
	_, replaceErr := svc.Replace(ctx, 1, valid)
	deleteErr := svc.Delete(ctx, 1)

	for name, err := range map[string]error{"List": listErr, "Get": getErr, "Create": createErr, "Replace": replaceErr, "Delete": deleteErr} {
		if !errors.Is(err, task.ErrUnexpected) {
			t.Errorf("%s err = %v, want task.ErrUnexpected", name, err)
		}
	}
}

func TestGetAndDelete(t *testing.T) {
	stored := task.Task{ID: 3, Name: "a", CreatedAt: t0, UpdatedAt: t0}
	var gotGetID, gotDeleteID int64
	svc := service.New(&fakeStore{
		get: func(_ context.Context, id int64) (task.Task, error) {
			gotGetID = id
			return stored, nil
		},
		delete: func(_ context.Context, id int64) error {
			gotDeleteID = id
			return nil
		},
	}, func() time.Time { return t0 })
	ctx := context.Background()

	res, err := svc.Get(ctx, 3)
	if err != nil || res != stored || gotGetID != 3 {
		t.Errorf("Get = %+v, %v (store got id %d)", res, err, gotGetID)
	}
	if err := svc.Delete(ctx, 3); err != nil || gotDeleteID != 3 {
		t.Errorf("Delete = %v (store got id %d)", err, gotDeleteID)
	}
}

func TestListPaging(t *testing.T) {
	rows := func(n int) []task.Task {
		ts := make([]task.Task, n)
		for i := range ts {
			ts[i] = task.Task{ID: int64(i + 1)}
		}
		return ts
	}
	cases := []struct {
		stored      int
		wantLen     int
		wantHasMore bool
	}{
		{4, 3, true},
		{3, 3, false},
		{1, 1, false},
	}
	for _, tc := range cases {
		var gotAfter int64
		var gotLimit int
		stored := tc.stored
		svc := service.New(&fakeStore{
			list: func(_ context.Context, afterID int64, limit int) ([]task.Task, error) {
				gotAfter, gotLimit = afterID, limit
				return rows(stored), nil
			},
		}, func() time.Time { return t0 })

		page, err := svc.List(context.Background(), service.ListParams{AfterID: 0, Limit: 3})

		if err != nil {
			t.Fatalf("stored %d: err = %v", tc.stored, err)
		}
		if gotAfter != 0 || gotLimit != 4 {
			t.Errorf("stored %d: store got (%d, %d), want (0, 4)", tc.stored, gotAfter, gotLimit)
		}
		if len(page.Tasks) != tc.wantLen || page.HasMore != tc.wantHasMore {
			t.Errorf("stored %d: got %d tasks, HasMore %v; want %d, %v", tc.stored, len(page.Tasks), page.HasMore, tc.wantLen, tc.wantHasMore)
		}
	}
}

func TestListValidation(t *testing.T) {
	cases := []struct {
		p       service.ListParams
		wantErr error
	}{
		{service.ListParams{Limit: 0}, task.ErrInvalidLimit},
		{service.ListParams{Limit: -1}, task.ErrInvalidLimit},
		{service.ListParams{Limit: 101}, task.ErrInvalidLimit},
		{service.ListParams{AfterID: -1, Limit: 20}, task.ErrInvalidCursor},
		{service.ListParams{Limit: 100}, nil},
	}
	for _, tc := range cases {
		svc := service.New(&fakeStore{
			list: func(context.Context, int64, int) ([]task.Task, error) {
				return nil, nil
			},
		}, func() time.Time { return t0 })

		_, err := svc.List(context.Background(), tc.p)

		if err != tc.wantErr {
			t.Errorf("%+v: err = %v, want %v", tc.p, err, tc.wantErr)
		}
	}
}
