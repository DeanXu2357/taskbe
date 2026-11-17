package service

import (
	"context"

	"github.com/DeanXu2357/taskbe/internal/task"
)

type Store interface {
	List(ctx context.Context, afterID int64, limit int) ([]task.Task, error)
	Get(ctx context.Context, id int64) (task.Task, error)
	Create(ctx context.Context, t task.Task) (task.Task, error)
	// Update replaces Name, Status and UpdatedAt of the task with t.ID and keeps its CreatedAt.
	Update(ctx context.Context, t task.Task) (task.Task, error)
	Delete(ctx context.Context, id int64) error
}
