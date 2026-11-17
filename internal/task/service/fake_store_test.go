package service_test

import (
	"context"
	"errors"

	"github.com/DeanXu2357/taskbe/internal/task"
	"github.com/DeanXu2357/taskbe/internal/task/service"
)

var errUnexpectedCall = errors.New("unexpected call")

type fakeStore struct {
	list   func(ctx context.Context, afterID int64, limit int) ([]task.Task, error)
	get    func(ctx context.Context, id int64) (task.Task, error)
	create func(ctx context.Context, t task.Task) (task.Task, error)
	update func(ctx context.Context, id, expectedVersion int64, data service.TaskUpdate) (task.Task, error)
	delete func(ctx context.Context, id int64) error
}

func (f *fakeStore) List(ctx context.Context, afterID int64, limit int) ([]task.Task, error) {
	if f.list == nil {
		return nil, errUnexpectedCall
	}
	return f.list(ctx, afterID, limit)
}

func (f *fakeStore) Get(ctx context.Context, id int64) (task.Task, error) {
	if f.get == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.get(ctx, id)
}

func (f *fakeStore) Create(ctx context.Context, t task.Task) (task.Task, error) {
	if f.create == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.create(ctx, t)
}

func (f *fakeStore) Update(ctx context.Context, id, expectedVersion int64, data service.TaskUpdate) (task.Task, error) {
	if f.update == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.update(ctx, id, expectedVersion, data)
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	if f.delete == nil {
		return errUnexpectedCall
	}
	return f.delete(ctx, id)
}
