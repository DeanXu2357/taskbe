package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/DeanXu2357/taskbe/internal/logx"
	"github.com/DeanXu2357/taskbe/internal/task"
)

const taskPathPrefix = "/tasks/"

type handler struct {
	svc    TaskService
	logger logx.Logger
}

func Register(mux *http.ServeMux, svc TaskService, logger logx.Logger) {
	h := &handler{svc: svc, logger: logger}
	mux.HandleFunc("/tasks", h.collection)
	mux.HandleFunc(taskPathPrefix, h.item)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "resource not found")
	})
}

func (h *handler) collection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listTasks(w, r)
	case http.MethodPost:
		h.createTask(w, r)
	default:
		writeMethodNotAllowed(w, "GET, POST")
	}
}

func (h *handler) item(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, codeNotFound, "task not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.getTask(w, r, id)
	case http.MethodPut:
		h.replaceTask(w, r, id)
	case http.MethodDelete:
		h.deleteTask(w, r, id)
	default:
		writeMethodNotAllowed(w, "GET, PUT, DELETE")
	}
}

func parseID(path string) (int64, bool) {
	s := strings.TrimPrefix(path, taskPathPrefix)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id < 1 || strconv.FormatInt(id, 10) != s {
		return 0, false
	}
	return id, true
}

func (h *handler) listTasks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	params := task.ListParams{Limit: task.DefaultPageSize}
	if s := query.Get("limit"); s != "" {
		limit, err := strconv.Atoi(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, codeValidationFailed, "limit must be an integer")
			return
		}
		params.Limit = limit
	}
	if s := query.Get("cursor"); s != "" {
		afterID, err := decodeCursor(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, codeValidationFailed, "cursor is invalid")
			return
		}
		params.AfterID = afterID
	}

	page, err := h.svc.List(r.Context(), params)
	if err != nil {
		switch {
		case errors.Is(err, task.ErrInvalidLimit):
			writeError(w, http.StatusBadRequest, codeValidationFailed, fmt.Sprintf("invalid query parameter: %s", err.Error()))
		case errors.Is(err, task.ErrInvalidCursor):
			writeError(w, http.StatusBadRequest, codeValidationFailed, "cursor is invalid")
		case errors.Is(err, task.ErrUnexpected):
			h.logger.ErrorContext(r.Context(), "list tasks failed", "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		default:
			h.logger.ErrorContext(r.Context(), "list tasks returned an unhandled error", "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}
		return
	}

	resp := listResponse{Data: make([]taskResponse, 0, len(page.Tasks))}
	for _, t := range page.Tasks {
		resp.Data = append(resp.Data, toTaskResponse(t))
	}
	if page.HasMore && len(page.Tasks) > 0 {
		next := encodeCursor(page.Tasks[len(page.Tasks)-1].ID)
		resp.NextCursor = &next
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if reqErr := decodeJSON(r, &req); reqErr != nil {
		writeError(w, reqErr.status, reqErr.code, reqErr.message)
		return
	}
	if req.Name == nil {
		writeError(w, http.StatusBadRequest, codeValidationFailed, "name is required")
		return
	}

	t, err := h.svc.Create(r.Context(), task.CreateInput{Name: *req.Name, Status: req.Status})
	if err != nil {
		switch {
		case errors.Is(err, task.ErrNameEmpty):
			writeError(w, http.StatusBadRequest, codeValidationFailed, "name must not be empty")
		case errors.Is(err, task.ErrNameTooLong):
			writeError(w, http.StatusBadRequest, codeValidationFailed, fmt.Sprintf("invalid name: %s", err.Error()))
		case errors.Is(err, task.ErrInvalidStatus):
			writeError(w, http.StatusBadRequest, codeValidationFailed, "status must be 0 or 1")
		case errors.Is(err, task.ErrUnexpected):
			h.logger.ErrorContext(r.Context(), "create task failed", "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		default:
			h.logger.ErrorContext(r.Context(), "create task returned an unhandled error", "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}
		return
	}

	w.Header().Set("Location", taskPathPrefix+strconv.FormatInt(t.ID, 10))
	writeJSON(w, http.StatusCreated, dataResponse{Data: toTaskResponse(t)})
}

func (h *handler) getTask(w http.ResponseWriter, r *http.Request, id int64) {
	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, task.ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, "task not found")
		case errors.Is(err, task.ErrUnexpected):
			h.logger.ErrorContext(r.Context(), "get task failed", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		default:
			h.logger.ErrorContext(r.Context(), "get task returned an unhandled error", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: toTaskResponse(t)})
}

func (h *handler) replaceTask(w http.ResponseWriter, r *http.Request, id int64) {
	var req replaceTaskRequest
	if reqErr := decodeJSON(r, &req); reqErr != nil {
		writeError(w, reqErr.status, reqErr.code, reqErr.message)
		return
	}
	if req.ID != nil && *req.ID != id {
		writeError(w, http.StatusBadRequest, codeValidationFailed, "id in body does not match path")
		return
	}
	if req.Name == nil {
		writeError(w, http.StatusBadRequest, codeValidationFailed, "name is required")
		return
	}
	if req.Status == nil {
		writeError(w, http.StatusBadRequest, codeValidationFailed, "status is required")
		return
	}
	if req.Version == nil {
		writeError(w, http.StatusBadRequest, codeValidationFailed, "version is required")
		return
	}

	t, err := h.svc.Replace(r.Context(), id, *req.Version, task.ReplaceInput{Name: *req.Name, Status: *req.Status})
	if err != nil {
		switch {
		case errors.Is(err, task.ErrNameEmpty):
			writeError(w, http.StatusBadRequest, codeValidationFailed, "name must not be empty")
		case errors.Is(err, task.ErrNameTooLong):
			writeError(w, http.StatusBadRequest, codeValidationFailed, fmt.Sprintf("invalid name: %s", err.Error()))
		case errors.Is(err, task.ErrInvalidStatus):
			writeError(w, http.StatusBadRequest, codeValidationFailed, "status must be 0 or 1")
		case errors.Is(err, task.ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, "task not found")
		case errors.Is(err, task.ErrVersionConflict):
			writeError(w, http.StatusConflict, codeConflict, "version does not match the current task")
		case errors.Is(err, task.ErrUnexpected):
			h.logger.ErrorContext(r.Context(), "replace task failed", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		default:
			h.logger.ErrorContext(r.Context(), "replace task returned an unhandled error", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, dataResponse{Data: toTaskResponse(t)})
}

func (h *handler) deleteTask(w http.ResponseWriter, r *http.Request, id int64) {
	if err := h.svc.Delete(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, task.ErrNotFound):
			writeError(w, http.StatusNotFound, codeNotFound, "task not found")
		case errors.Is(err, task.ErrUnexpected):
			h.logger.ErrorContext(r.Context(), "delete task failed", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		default:
			h.logger.ErrorContext(r.Context(), "delete task returned an unhandled error", "id", id, "error", err)
			writeError(w, http.StatusInternalServerError, codeInternalError, "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
