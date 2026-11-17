package memory_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
	"github.com/DeanXu2357/taskbe/internal/task/store/memory"
)

var (
	t0 = time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	t1 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)
)

func ids(tasks []task.Task) []int64 {
	out := []int64{}
	for _, t := range tasks {
		out = append(out, t.ID)
	}
	return out
}

func TestCreateAssignsSequentialIDs(t *testing.T) {
	s := memory.New()
	ctx := context.Background()

	for want := int64(1); want <= 3; want++ {
		got, err := s.Create(ctx, task.Task{ID: 99, Name: "a", Version: 1, CreatedAt: t0, UpdatedAt: t0})
		if err != nil {
			t.Fatalf("Create err = %v", err)
		}
		if got.ID != want {
			t.Errorf("ID = %d, want %d", got.ID, want)
		}
	}
}

func TestIDsAreNotReused(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	s.Create(ctx, task.Task{Name: "a"})
	s.Create(ctx, task.Task{Name: "b"})
	if err := s.Delete(ctx, 2); err != nil {
		t.Fatalf("Delete err = %v", err)
	}

	got, err := s.Create(ctx, task.Task{Name: "c"})

	if err != nil {
		t.Fatalf("Create err = %v", err)
	}
	if got.ID != 3 {
		t.Errorf("ID = %d, want 3", got.ID)
	}
}

func TestGet(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	created, _ := s.Create(ctx, task.Task{Name: "a", Status: task.StatusCompleted, Version: 1, CreatedAt: t0, UpdatedAt: t0})

	got, err := s.Get(ctx, created.ID)

	if err != nil {
		t.Fatalf("Get err = %v", err)
	}
	if got != created {
		t.Errorf("Get = %+v, want %+v", got, created)
	}
	if _, err := s.Get(ctx, 42); !errors.Is(err, service.ErrRecordNotFound) {
		t.Errorf("Get(42) err = %v, want ErrRecordNotFound", err)
	}
}

func TestUpdateKeepsCreatedAt(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	s.Create(ctx, task.Task{Name: "a", Version: 1, CreatedAt: t0, UpdatedAt: t0})

	got, err := s.Update(ctx, 1, 1, service.TaskUpdate{Name: "b", Status: task.StatusCompleted, UpdatedAt: t1})

	if err != nil {
		t.Fatalf("Update err = %v", err)
	}
	want := task.Task{ID: 1, Name: "b", Status: task.StatusCompleted, Version: 2, CreatedAt: t0, UpdatedAt: t1}
	if got != want {
		t.Errorf("Update = %+v, want %+v", got, want)
	}
	if stored, _ := s.Get(ctx, 1); stored != want {
		t.Errorf("stored = %+v, want %+v", stored, want)
	}
	if _, err := s.Update(ctx, 42, 1, service.TaskUpdate{Name: "b"}); !errors.Is(err, service.ErrRecordNotFound) {
		t.Errorf("Update(42) err = %v, want ErrRecordNotFound", err)
	}
}

func TestUpdateChecksVersion(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	s.Create(ctx, task.Task{Name: "a", Version: 1, CreatedAt: t0, UpdatedAt: t0})
	s.Update(ctx, 1, 1, service.TaskUpdate{Name: "b", UpdatedAt: t1})
	before, _ := s.Get(ctx, 1)

	_, err := s.Update(ctx, 1, 1, service.TaskUpdate{Name: "c", Status: task.StatusCompleted, UpdatedAt: t1})

	if !errors.Is(err, service.ErrVersionMismatch) {
		t.Errorf("err = %v, want ErrVersionMismatch", err)
	}
	if after, _ := s.Get(ctx, 1); after != before {
		t.Errorf("stored = %+v, want unchanged %+v", after, before)
	}
}

func TestDelete(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	s.Create(ctx, task.Task{Name: "a"})

	if err := s.Delete(ctx, 1); err != nil {
		t.Fatalf("Delete err = %v", err)
	}
	if _, err := s.Get(ctx, 1); !errors.Is(err, service.ErrRecordNotFound) {
		t.Errorf("Get after delete err = %v, want ErrRecordNotFound", err)
	}
	if err := s.Delete(ctx, 1); !errors.Is(err, service.ErrRecordNotFound) {
		t.Errorf("second Delete err = %v, want ErrRecordNotFound", err)
	}
}

func TestListKeyset(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		s.Create(ctx, task.Task{Name: "a"})
	}

	cases := []struct {
		afterID int64
		limit   int
		want    []int64
	}{
		{0, 2, []int64{1, 2}},
		{2, 2, []int64{3, 4}},
		{4, 2, []int64{5}},
		{5, 2, []int64{}},
	}
	for _, tc := range cases {
		got, err := s.List(ctx, tc.afterID, tc.limit)
		if err != nil {
			t.Fatalf("List(%d, %d) err = %v", tc.afterID, tc.limit, err)
		}
		if !reflect.DeepEqual(ids(got), tc.want) {
			t.Errorf("List(%d, %d) = %v, want %v", tc.afterID, tc.limit, ids(got), tc.want)
		}
	}

	if err := s.Delete(ctx, 3); err != nil {
		t.Fatalf("Delete err = %v", err)
	}
	for _, afterID := range []int64{2, 3} {
		got, _ := s.List(ctx, afterID, 2)
		if want := []int64{4, 5}; !reflect.DeepEqual(ids(got), want) {
			t.Errorf("after delete, List(%d, 2) = %v, want %v", afterID, ids(got), want)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := memory.New()
	ctx := context.Background()
	done := make(chan struct{})
	var readers sync.WaitGroup
	readers.Add(1)
	go func() {
		defer readers.Done()
		for {
			select {
			case <-done:
				return
			default:
				s.List(ctx, 0, 10)
				s.Get(ctx, 1)
			}
		}
	}()

	var writers sync.WaitGroup
	for i := 0; i < 100; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			s.Create(ctx, task.Task{Name: "a"})
		}()
	}
	writers.Wait()
	close(done)
	readers.Wait()

	got, err := s.List(ctx, 0, 1000)
	if err != nil {
		t.Fatalf("List err = %v", err)
	}
	want := make([]int64, 100)
	for i := range want {
		want[i] = int64(i + 1)
	}
	if !reflect.DeepEqual(ids(got), want) {
		t.Errorf("ids = %v, want 1..100", ids(got))
	}
}
