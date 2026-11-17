package httpapi

import (
	"context"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

type TaskService interface {
	List(ctx context.Context, p service.ListParams) (service.Page, error)
	Get(ctx context.Context, id int64) (task.Task, error)
	Create(ctx context.Context, in service.CreateInput) (task.Task, error)
	Replace(ctx context.Context, id, expectedVersion int64, in service.ReplaceInput) (task.Task, error)
	Delete(ctx context.Context, id int64) error
}
