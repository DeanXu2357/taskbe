package logx

import (
	"context"
	"log"
)

// Logger matches the method set of *slog.Logger, so a *slog.Logger can be injected once the minimum Go version allows it.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

type StdLogger struct {
	l *log.Logger
}

func New(l *log.Logger) *StdLogger {
	return &StdLogger{l: l}
}

func (s *StdLogger) Debug(msg string, args ...any) { s.print("DEBUG", msg, args) }
func (s *StdLogger) Info(msg string, args ...any)  { s.print("INFO", msg, args) }
func (s *StdLogger) Warn(msg string, args ...any)  { s.print("WARN", msg, args) }
func (s *StdLogger) Error(msg string, args ...any) { s.print("ERROR", msg, args) }

func (s *StdLogger) DebugContext(_ context.Context, msg string, args ...any) { s.Debug(msg, args...) }
func (s *StdLogger) InfoContext(_ context.Context, msg string, args ...any)  { s.Info(msg, args...) }
func (s *StdLogger) WarnContext(_ context.Context, msg string, args ...any)  { s.Warn(msg, args...) }
func (s *StdLogger) ErrorContext(_ context.Context, msg string, args ...any) { s.Error(msg, args...) }

func (s *StdLogger) print(level, msg string, args []any) {
	s.l.Println(append([]any{level, msg}, args...)...)
}
