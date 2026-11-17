package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/DeanXu2357/taskbe/internal/httpapi"
	"github.com/DeanXu2357/taskbe/internal/task/service"
	"github.com/DeanXu2357/taskbe/internal/task/store/memory"
)

type e2eTask struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Status    int    `json:"status"`
	Version   int64  `json:"version"`
	CreatedAt string `json:"created_at"`
}

func TestEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	httpapi.Register(mux, service.New(memory.New(), time.Now), discardLogger)

	decode := func(body []byte, v any) {
		t.Helper()
		if err := json.Unmarshal(body, v); err != nil {
			t.Fatalf("decode %q: %v", body, err)
		}
	}

	rec := do(mux, http.MethodPost, "/tasks", `{"name":"買早餐"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, want 201", rec.Code)
	}
	var created struct{ Data e2eTask }
	decode(rec.Body.Bytes(), &created)
	if created.Data.ID != 1 || created.Data.Version != 1 {
		t.Fatalf("POST data = %+v, want id 1 version 1", created.Data)
	}

	rec = do(mux, http.MethodGet, "/tasks", "")
	var list struct{ Data []e2eTask }
	decode(rec.Body.Bytes(), &list)
	if rec.Code != http.StatusOK || len(list.Data) != 1 {
		t.Fatalf("GET /tasks = %d %s, want 200 with 1 task", rec.Code, rec.Body)
	}

	rec = do(mux, http.MethodPut, "/tasks/1", `{"name":"買午餐","status":1,"version":1,"created_at":"1999-01-01T00:00:00Z"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want 200: %s", rec.Code, rec.Body)
	}
	var replaced struct{ Data e2eTask }
	decode(rec.Body.Bytes(), &replaced)
	if replaced.Data.Version != 2 || replaced.Data.CreatedAt != created.Data.CreatedAt {
		t.Errorf("PUT data = %+v, want version 2 and created_at %s", replaced.Data, created.Data.CreatedAt)
	}

	rec = do(mux, http.MethodPut, "/tasks/1", `{"name":"買晚餐","status":0,"version":1}`)
	if rec.Code != http.StatusConflict || decodeError(t, rec).Error.Code != "conflict" {
		t.Errorf("stale PUT = %d %s, want 409 conflict", rec.Code, rec.Body)
	}

	rec = do(mux, http.MethodGet, "/tasks/1", "")
	var got struct{ Data e2eTask }
	decode(rec.Body.Bytes(), &got)
	if rec.Code != http.StatusOK || got.Data.Name != "買午餐" || got.Data.Status != 1 {
		t.Errorf("GET /tasks/1 = %d %+v, want 200 買午餐 status 1", rec.Code, got.Data)
	}

	if rec = do(mux, http.MethodDelete, "/tasks/1", ""); rec.Code != http.StatusNoContent {
		t.Errorf("DELETE status = %d, want 204", rec.Code)
	}
	if rec = do(mux, http.MethodGet, "/tasks/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("GET after delete status = %d, want 404", rec.Code)
	}
	rec = do(mux, http.MethodGet, "/tasks", "")
	assertBody(t, rec.Body.String(), `{"data":[],"next_cursor":null}`)
}
