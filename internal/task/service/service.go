package service

import (
	"context"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Service struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Service {
	return &Service{store: store, now: now}
}

type CreateInput struct {
	Name   *string
	Status *task.Status
}

type ReplaceInput struct {
	Name   *string
	Status *task.Status
}

type ListParams struct {
	AfterID int64
	Limit   int
}

type Page struct {
	Tasks   []task.Task
	HasMore bool
}

func (s *Service) List(ctx context.Context, p ListParams) (Page, error) {
	return Page{}, task.ErrNotImplemented
}

func (s *Service) Get(ctx context.Context, id int64) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Service) Create(ctx context.Context, in CreateInput) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Service) Replace(ctx context.Context, id int64, in ReplaceInput) (task.Task, error) {
	return task.Task{}, task.ErrNotImplemented
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return task.ErrNotImplemented
}
