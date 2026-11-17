package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DeanXu2357/taskbe/internal/task"
)

type errorCode string

const (
	codeInvalidJSON      errorCode = "invalid_json"
	codeValidationFailed errorCode = "validation_failed"
	codeNotFound         errorCode = "not_found"
	codeMethodNotAllowed errorCode = "method_not_allowed"
	codeConflict         errorCode = "conflict"
	codePayloadTooLarge  errorCode = "payload_too_large"
	codeInternalError    errorCode = "internal_error"
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    errorCode `json:"code"`
	Message string    `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code errorCode, message string) {
	writeJSON(w, status, errorResponse{Error: errorBody{Code: code, Message: message}})
}

func writeMethodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, codeMethodNotAllowed, "method not allowed")
}

type dataResponse struct {
	Data any `json:"data"`
}

type listResponse struct {
	Data       []taskResponse `json:"data"`
	NextCursor *string        `json:"next_cursor"`
}

type taskResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Status    int       `json:"status"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toTaskResponse(t task.Task) taskResponse {
	return taskResponse{
		ID:        t.ID,
		Name:      t.Name,
		Status:    int(t.Status),
		Version:   t.Version,
		CreatedAt: t.CreatedAt.UTC(),
		UpdatedAt: t.UpdatedAt.UTC(),
	}
}
