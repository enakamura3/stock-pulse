package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestHandler_ListWorkers(t *testing.T) {
	m := NewManager()
	m.Register(NewWorker("TestWorker", "Test Description", time.Minute, func(ctx context.Context) {}))

	h := NewHandler(m)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req, _ := http.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v", status)
	}

	var infos []Info
	if err := json.NewDecoder(rr.Body).Decode(&infos); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(infos) != 1 || infos[0].Name != "TestWorker" {
		t.Errorf("expected 1 TestWorker, got %v", infos)
	}
}

func TestHandler_TriggerWorker(t *testing.T) {
	m := NewManager()
	m.Register(NewWorker("TestWorker", "Test Description", time.Minute, func(ctx context.Context) {}))

	h := NewHandler(m)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	// Valid trigger
	req, _ := http.NewRequest("POST", "/TestWorker/trigger", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status OK, got %v", status)
	}

	// Invalid trigger
	req2, _ := http.NewRequest("POST", "/NonExistent/trigger", nil)
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if status := rr2.Code; status != http.StatusNotFound {
		t.Errorf("expected status NotFound, got %v", status)
	}
}
