package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/DeanXu2357/taskbe/internal/task"
)

const DefaultPageSize = 20

const (
	maxPageSize   = 100
	maxNameLength = 200
)

type Service struct {
	store Store
	now   func() time.Time
}

func New(store Store, now func() time.Time) *Service {
	return &Service{store: store, now: now}
}

type CreateInput struct {
	Name   string
	Status task.Status
}

type ReplaceInput struct {
	Name   string
	Status task.Status
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
	if p.Limit < 1 || p.Limit > maxPageSize {
		return Page{}, fmt.Errorf("%w (must be between 1 and %d)", task.ErrInvalidLimit, maxPageSize)
	}
	if p.AfterID < 0 {
		return Page{}, task.ErrInvalidCursor
	}
	tasks, err := s.store.List(ctx, p.AfterID, p.Limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("%w: list tasks after %d: %v", task.ErrUnexpected, p.AfterID, err)
	}
	if len(tasks) > p.Limit {
		return Page{Tasks: tasks[:p.Limit], HasMore: true}, nil
	}
	return Page{Tasks: tasks}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (task.Task, error) {
	t, err := s.store.Get(ctx, id)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return task.Task{}, task.ErrNotFound
		}
		return task.Task{}, fmt.Errorf("%w: get task %d: %v", task.ErrUnexpected, id, err)
	}
	return t, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (task.Task, error) {
	name, err := normalizeName(in.Name)
	if err != nil {
		return task.Task{}, err
	}
	if !in.Status.Valid() {
		return task.Task{}, task.ErrInvalidStatus
	}
	now := s.now().UTC()
	t, err := s.store.Create(ctx, task.Task{Name: name, Status: in.Status, Version: 1, CreatedAt: now, UpdatedAt: now})
	if err != nil {
		return task.Task{}, fmt.Errorf("%w: create task: %v", task.ErrUnexpected, err)
	}
	return t, nil
}

func (s *Service) Replace(ctx context.Context, id, expectedVersion int64, in ReplaceInput) (task.Task, error) {
	name, err := normalizeName(in.Name)
	if err != nil {
		return task.Task{}, err
	}
	if !in.Status.Valid() {
		return task.Task{}, task.ErrInvalidStatus
	}
	t, err := s.store.Update(ctx, id, expectedVersion, TaskUpdate{Name: name, Status: in.Status, UpdatedAt: s.now().UTC()})
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return task.Task{}, task.ErrNotFound
		}
		if errors.Is(err, ErrVersionMismatch) {
			return task.Task{}, task.ErrVersionConflict
		}
		return task.Task{}, fmt.Errorf("%w: replace task %d: %v", task.ErrUnexpected, id, err)
	}
	return t, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if err := s.store.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return task.ErrNotFound
		}
		return fmt.Errorf("%w: delete task %d: %v", task.ErrUnexpected, id, err)
	}
	return nil
}

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", task.ErrNameEmpty
	}
	if utf8.RuneCountInString(name) > maxNameLength {
		return "", fmt.Errorf("%w (at most %d characters)", task.ErrNameTooLong, maxNameLength)
	}
	return name, nil
}
