package httpapi

import (
	"context"

	"github.com/DeanXu2357/taskbe/internal/task"
)

type TaskService interface {
	List(ctx context.Context, p task.ListParams) (task.Page, error)
	Get(ctx context.Context, id int64) (task.Task, error)
	Create(ctx context.Context, in task.CreateInput) (task.Task, error)
	Replace(ctx context.Context, id, expectedVersion int64, in task.ReplaceInput) (task.Task, error)
	Delete(ctx context.Context, id int64) error
}
