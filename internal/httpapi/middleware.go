package httpapi

import (
	"net/http"
	"time"

	"github.com/DeanXu2357/taskbe/internal/logx"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

func LogRequests(logger logx.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		logger.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start),
		)
	})
}

func RecoverPanics(logger logx.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			// http.ErrAbortHandler is the documented way to abort a response; let net/http handle it.
			if v == http.ErrAbortHandler {
				panic(v)
			}
			logger.ErrorContext(r.Context(), "handler panicked", "panic", v)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}()
		next.ServeHTTP(w, r)
	})
}
