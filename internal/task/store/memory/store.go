package memory

import (
	"context"
	"sort"
	"sync"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

type Store struct {
	mu     sync.RWMutex
	tasks  map[int64]task.Task
	ids    []int64
	lastID int64
}

func New() *Store {
	return &Store{tasks: make(map[int64]task.Task)}
}

func (s *Store) List(ctx context.Context, afterID int64, limit int) ([]task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	start := sort.Search(len(s.ids), func(i int) bool { return s.ids[i] > afterID })
	end := len(s.ids)
	if limit < end-start {
		end = start + limit
	}
	out := make([]task.Task, 0, end-start)
	for _, id := range s.ids[start:end] {
		out = append(out, s.tasks[id])
	}
	return out, nil
}

func (s *Store) Get(ctx context.Context, id int64) (task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	if !ok {
		return task.Task{}, service.ErrRecordNotFound
	}
	return t, nil
}

func (s *Store) Create(ctx context.Context, t task.Task) (task.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastID++
	t.ID = s.lastID
	s.tasks[t.ID] = t
	s.ids = append(s.ids, t.ID)
	return t, nil
}

func (s *Store) Update(ctx context.Context, id, expectedVersion int64, data service.TaskUpdate) (task.Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[id]
	if !ok {
		return task.Task{}, service.ErrRecordNotFound
	}
	if t.Version != expectedVersion {
		return task.Task{}, service.ErrVersionMismatch
	}
	t.Name = data.Name
	t.Status = data.Status
	t.UpdatedAt = data.UpdatedAt
	t.Version = expectedVersion + 1
	s.tasks[id] = t
	return t, nil
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return service.ErrRecordNotFound
	}
	delete(s.tasks, id)
	i := sort.Search(len(s.ids), func(i int) bool { return s.ids[i] >= id })
	s.ids = append(s.ids[:i], s.ids[i+1:]...)
	return nil
}
