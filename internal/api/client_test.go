package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/ats/conn123/candidate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("User-Agent") != "UnifiedCLI/1.0.0" {
			t.Errorf("unexpected user-agent header: %s", r.Header.Get("User-Agent"))
		}
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", r.URL.Query().Get("limit"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL, "1.0.0")
	body, statusCode, err := c.Do("GET", "ats", "conn123", "candidate", "", nil, map[string]string{"limit": "10"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 200 {
		t.Errorf("expected status 200, got %d", statusCode)
	}
	if strings.TrimSpace(string(body)) != `[{"id":"1"}]` {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/conn456/contact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		reqBody, _ := io.ReadAll(r.Body)
		if strings.TrimSpace(string(reqBody)) != `{"name":"John"}` {
			t.Errorf("unexpected body: %s", reqBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":"new1","name":"John"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL, "1.0.0")
	data := []byte(`{"name":"John"}`)
	body, statusCode, err := c.Do("POST", "crm", "conn456", "contact", "", data, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 201 {
		t.Errorf("expected status 201, got %d", statusCode)
	}
	if !strings.Contains(string(body), `"name":"John"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientGetWithID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ats/conn123/candidate/id789" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"id":"id789"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL, "1.0.0")
	body, _, err := c.Do("GET", "ats", "conn123", "candidate", "id789", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(body), `"id789"`) {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestClientHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := NewClient("test-key", server.URL, "1.0.0")
	body, statusCode, err := c.Do("GET", "ats", "conn123", "candidate", "bad", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if statusCode != 404 {
		t.Errorf("expected status 404, got %d", statusCode)
	}
	if !strings.Contains(string(body), "not found") {
		t.Errorf("unexpected body: %s", body)
	}
}
