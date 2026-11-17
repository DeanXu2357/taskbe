package memory

import (
	"context"

	"github.com/DeanXu2357/taskbe/internal/task"
)

type Store struct{}

func New() *Store {
	return &Store{}
}

func (s *Store) List(ctx context.Context, afterID int64, limit int) ([]task.Task, error) {
	return nil, task.ErrNotImplemented
}

func (s *Store) Get(ctx context.Context, id int64) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Store) Create(ctx context.Context, t task.Task) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Store) Update(ctx context.Context, t task.Task) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	return task.ErrNotImplemented
}
