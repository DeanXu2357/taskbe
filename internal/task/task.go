package task

import (
	"errors"
	"time"
)

type Status int

const (
	StatusIncomplete Status = 0
	StatusCompleted  Status = 1
)

func (s Status) Valid() bool {
	return s == StatusIncomplete || s == StatusCompleted
}

type Task struct {
	ID        int64
	Name      string
	Status    Status
	CreatedAt time.Time
	UpdatedAt time.Time
}

const MaxNameLength = 200

var (
	ErrNotFound       = errors.New("task not found")
	ErrNotImplemented = errors.New("not implemented")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + " " + e.Message
}
