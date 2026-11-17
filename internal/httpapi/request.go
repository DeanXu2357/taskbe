package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/DeanXu2357/taskbe/internal/task"
)

const maxBodyBytes = 1 << 20

type requestError struct {
	status  int
	code    errorCode
	message string
}

type createTaskRequest struct {
	Name *string `json:"name"`
	// An omitted or null status leaves the zero value, StatusIncomplete, which is the documented default.
	Status task.Status `json:"status"`
}

type replaceTaskRequest struct {
	ID      *int64       `json:"id"`
	Name    *string      `json:"name"`
	Status  *task.Status `json:"status"`
	Version *int64       `json:"version"`
}

func decodeJSON(r *http.Request, dst any) *requestError {
	// http.MaxBytesReader is not used because Go 1.18 has no http.MaxBytesError to tell its error apart.
	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes+1))
	if err != nil {
		return &requestError{http.StatusBadRequest, codeInvalidJSON, "request body could not be read"}
	}
	if len(body) > maxBodyBytes {
		return &requestError{http.StatusRequestEntityTooLarge, codePayloadTooLarge, "request body must not exceed 1 MiB"}
	}
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return &requestError{http.StatusBadRequest, codeInvalidJSON, "request body is empty"}
	}
	// json.Unmarshal treats a null body as a no-op and returns no error.
	if bytes.Equal(trimmed, []byte("null")) {
		return &requestError{http.StatusBadRequest, codeInvalidJSON, "request body must be a JSON object"}
	}
	if err := json.Unmarshal(body, dst); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			if typeErr.Field == "" {
				return &requestError{http.StatusBadRequest, codeInvalidJSON, "request body must be a JSON object"}
			}
			return &requestError{http.StatusBadRequest, codeInvalidJSON, fmt.Sprintf("invalid value for field %q", typeErr.Field)}
		}
		return &requestError{http.StatusBadRequest, codeInvalidJSON, "request body is not valid JSON"}
	}
	return nil
}
