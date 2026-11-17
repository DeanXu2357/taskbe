package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/DeanXu2357/taskbe/internal/logx"
)

const taskPathPrefix = "/tasks/"

type handler struct {
	svc    TaskService
	logger logx.Logger
}

func New(svc TaskService, logger logx.Logger) http.Handler {
	h := &handler{svc: svc, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", h.collection)
	mux.HandleFunc(taskPathPrefix, h.item)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	})
	return mux
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
		writeError(w, http.StatusNotFound, "not_found", "task not found")
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
	writeNotImplemented(w)
}

func (h *handler) createTask(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

func (h *handler) getTask(w http.ResponseWriter, r *http.Request, id int64) {
	writeNotImplemented(w)
}

func (h *handler) replaceTask(w http.ResponseWriter, r *http.Request, id int64) {
	writeNotImplemented(w)
}

func (h *handler) deleteTask(w http.ResponseWriter, r *http.Request, id int64) {
	writeNotImplemented(w)
}
