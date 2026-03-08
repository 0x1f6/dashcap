package client

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// newTestClient creates a Client pointing at the given httptest.Server.
func newTestClient(t *testing.T, srv *httptest.Server, token string) *Client {
	t.Helper()
	// Parse host:port from srv.URL (e.g. "http://127.0.0.1:12345")
	url := srv.URL
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	parts := strings.SplitN(url, ":", 2)
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])
	return New(Options{Host: host, Port: port, Token: token})
}

func jsonHandler(v any) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v)
	})
}

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]string{"status": "ok"}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.Health()
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("Health().Status = %q, want %q", resp.Status, "ok")
	}
}

func TestStatus(t *testing.T) {
	srv := httptest.NewServer(jsonHandler(map[string]any{
		"interface":     "eth0",
		"uptime":        "5m0s",
		"segment_count": 10,
		"total_packets": 1000,
		"total_bytes":   500000,
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.Status()
	if err != nil {
		t.Fatalf("Status() error: %v", err)
	}
	if resp.Interface != "eth0" {
		t.Errorf("Interface = %q, want %q", resp.Interface, "eth0")
	}
	if resp.TotalPackets != 1000 {
		t.Errorf("TotalPackets = %d, want 1000", resp.TotalPackets)
	}
}

func TestTrigger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req TriggerRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":     "test-1",
			"status": "pending",
			"source": "api",
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.Trigger(TriggerRequest{Duration: "30s"})
	if err != nil {
		t.Fatalf("Trigger() error: %v", err)
	}
	if resp.ID != "test-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "test-1")
	}
}

func TestTriggers(t *testing.T) {
	srv := httptest.NewServer(jsonHandler([]map[string]string{
		{"id": "1", "status": "completed"},
		{"id": "2", "status": "pending"},
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.Triggers()
	if err != nil {
		t.Fatalf("Triggers() error: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("len(Triggers()) = %d, want 2", len(resp))
	}
}

func TestRing(t *testing.T) {
	srv := httptest.NewServer(jsonHandler([]map[string]any{
		{"Index": 0, "Packets": 100, "Bytes": 5000},
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.Ring()
	if err != nil {
		t.Fatalf("Ring() error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("len(Ring()) = %d, want 1", len(resp))
	}
	if resp[0].Packets != 100 {
		t.Errorf("Packets = %d, want 100", resp[0].Packets)
	}
}

func TestTriggerStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/trigger/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "test-1",
			"status":     "completed",
			"source":     "api",
			"saved_path": "/data/saved/test",
			"metadata":   map[string]any{"trigger_id": "test-1", "interface": "eth0"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	resp, err := c.TriggerStatus("test-1")
	if err != nil {
		t.Fatalf("TriggerStatus() error: %v", err)
	}
	if resp.ID != "test-1" {
		t.Errorf("ID = %q, want %q", resp.ID, "test-1")
	}
	if resp.Status != "completed" {
		t.Errorf("Status = %q, want %q", resp.Status, "completed")
	}
	if resp.Metadata == nil {
		t.Error("Metadata should not be nil for completed trigger")
	}
}

func TestTriggerStatusNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "trigger not found"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	_, err := c.TriggerStatus("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "mytoken")
	_, _ = c.Health()
	if gotAuth != "Bearer mytoken" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer mytoken")
	}
}

func TestNoAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "")
	_, _ = c.Health()
	if gotAuth != "" {
		t.Errorf("Authorization = %q, want empty", gotAuth)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid token"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv, "badtoken")
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.Message != "invalid token" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "invalid token")
	}
}
