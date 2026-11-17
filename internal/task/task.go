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
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

const MaxNameLength = 200

var (
	ErrNotFound        = errors.New("task not found")
	ErrNameEmpty       = errors.New("name must not be empty")
	ErrNameTooLong     = errors.New("name must be at most 200 characters")
	ErrInvalidStatus   = errors.New("status must be 0 or 1")
	ErrVersionConflict = errors.New("task was modified by another request")
	ErrInvalidLimit    = errors.New("limit must be between 1 and 100")
	ErrInvalidCursor   = errors.New("cursor is invalid")
	ErrUnexpected      = errors.New("unexpected error")
)
