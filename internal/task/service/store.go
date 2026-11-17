package service

import (
	"context"
	"errors"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
)

// ErrRecordNotFound is the error a Store must return from Get, Update and
// Delete when no task has the given id. The Service translates it into
// task.ErrNotFound; any other Store error is treated as a storage failure.
var ErrRecordNotFound = errors.New("record not found")

// ErrVersionMismatch is the error a Store must return from Update when the
// stored version differs from expectedVersion. The Service translates it
// into task.ErrVersionConflict.
var ErrVersionMismatch = errors.New("version mismatch")

type TaskUpdate struct {
	Name      string
	Status    task.Status
	UpdatedAt time.Time
}

type Store interface {
	List(ctx context.Context, afterID int64, limit int) ([]task.Task, error)
	Get(ctx context.Context, id int64) (task.Task, error)
	Create(ctx context.Context, t task.Task) (task.Task, error)
	// Update writes data to the task with id and sets its version to
	// expectedVersion+1, but only if its stored version equals expectedVersion;
	// the check and the write must be atomic.
	Update(ctx context.Context, id, expectedVersion int64, data TaskUpdate) (task.Task, error)
	Delete(ctx context.Context, id int64) error
}
