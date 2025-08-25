package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sanchxt/isame-lb/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.NewDefaultConfig()
	srv := New(cfg)

	if srv == nil {
		t.Fatal("New() returned nil")
	}

	if srv.config != cfg {
		t.Error("New() did not set config correctly")
	}
}

func TestServer_healthHandler(t *testing.T) {
	cfg := config.NewDefaultConfig()
	srv := New(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	srv.healthHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("healthHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("healthHandler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}

	expected := `{"status":"ok"}`
	if rr.Body.String() != expected {
		t.Errorf("healthHandler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestServer_rootHandler(t *testing.T) {
	cfg := config.NewDefaultConfig()
	srv := New(cfg)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	srv.rootHandler(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("rootHandler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("rootHandler returned wrong content type: got %v want %v",
			contentType, "application/json")
	}

	expected := `{"service":"isame-lb","phase":"0"}`
	if rr.Body.String() != expected {
		t.Errorf("rootHandler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
