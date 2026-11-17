package httpapi_test

import (
	"context"
	"errors"

	"github.com/DeanXu2357/taskbe/internal/task"
)

var errUnexpectedCall = errors.New("unexpected call")

type fakeService struct {
	list    func(ctx context.Context, p task.ListParams) (task.Page, error)
	get     func(ctx context.Context, id int64) (task.Task, error)
	create  func(ctx context.Context, in task.CreateInput) (task.Task, error)
	replace func(ctx context.Context, id, expectedVersion int64, in task.ReplaceInput) (task.Task, error)
	delete  func(ctx context.Context, id int64) error
}

func (f *fakeService) List(ctx context.Context, p task.ListParams) (task.Page, error) {
	if f.list == nil {
		return task.Page{}, errUnexpectedCall
	}
	return f.list(ctx, p)
}

func (f *fakeService) Get(ctx context.Context, id int64) (task.Task, error) {
	if f.get == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.get(ctx, id)
}

func (f *fakeService) Create(ctx context.Context, in task.CreateInput) (task.Task, error) {
	if f.create == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.create(ctx, in)
}

func (f *fakeService) Replace(ctx context.Context, id, expectedVersion int64, in task.ReplaceInput) (task.Task, error) {
	if f.replace == nil {
		return task.Task{}, errUnexpectedCall
	}
	return f.replace(ctx, id, expectedVersion, in)
}

func (f *fakeService) Delete(ctx context.Context, id int64) error {
	if f.delete == nil {
		return errUnexpectedCall
	}
	return f.delete(ctx, id)
}
