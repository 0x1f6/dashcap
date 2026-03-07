package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket/layers"

	"dashcap/internal/api"
	"dashcap/internal/buffer"
	"dashcap/internal/config"
	"dashcap/internal/trigger"
)

// apiTestDisk implements storage.DiskOps for API tests.
type apiTestDisk struct{}

func (apiTestDisk) FreeBytes(_ string) (uint64, error)       { return 1 << 30, nil }
func (apiTestDisk) TotalBytes(_ string) (uint64, error)      { return 100 << 30, nil }
func (apiTestDisk) Preallocate(f *os.File, size int64) error { return f.Truncate(size) }
func (apiTestDisk) LockFile(_ *os.File) error                { return nil }
func (apiTestDisk) UnlockFile(_ *os.File) error              { return nil }

// newTestServer creates a Server backed by a minimal in-memory ring,
// starts it on a random OS-assigned port, and returns the base URL.
func newTestServer(t *testing.T) string {
	t.Helper()

	cfg := &config.Config{
		Interface:         "test0",
		BufferSize:        3 * 1024,
		SegmentSize:       1024,
		SegmentCount:      3,
		DataDir:           t.TempDir(),
		SavedDir:          "saved",
		MinFreeAfterAlloc: 0,
		DefaultDuration:   5 * time.Minute,
		APIPort:           0,
	}

	ring, err := buffer.NewRingManager(cfg, apiTestDisk{}, layers.LinkTypeEthernet, buffer.SHBInfo{})
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	t.Cleanup(func() { _ = ring.Close() })

	disp := trigger.NewDispatcher(cfg, ring, buffer.SHBInfo{})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen: %v", err)
	}

	srv := api.New(cfg, ring, disp)
	go func() { _ = srv.Serve(l) }()

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	})

	return fmt.Sprintf("http://%s", l.Addr().String())
}

func getBody(t *testing.T, url string) (int, string) {
	t.Helper()
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestHealth(t *testing.T) {
	base := newTestServer(t)

	code, body := getBody(t, base+"/api/v1/health")
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	if !strings.Contains(body, "ok") {
		t.Errorf("body should contain 'ok', got: %s", body)
	}
}

func TestStatus(t *testing.T) {
	base := newTestServer(t)

	code, body := getBody(t, base+"/api/v1/status")
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	if !strings.Contains(body, "test0") {
		t.Errorf("body should contain interface name 'test0', got: %s", body)
	}
}

func TestTriggerEndpoint(t *testing.T) {
	base := newTestServer(t)

	resp, err := http.Post(base+"/api/v1/trigger", "application/json", nil) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /api/v1/trigger: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status: got %d, want 202", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	id, _ := result["id"].(string)
	if id == "" {
		t.Errorf("response 'id' field should be non-empty, got: %v", result)
	}
}

func TestTriggersEndpoint(t *testing.T) {
	base := newTestServer(t)

	code, body := getBody(t, base+"/api/v1/triggers")
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	// Must be a JSON array
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "[") {
		t.Errorf("body should be a JSON array, got: %s", body)
	}
}

func TestRingEndpoint(t *testing.T) {
	base := newTestServer(t)

	code, body := getBody(t, base+"/api/v1/ring")
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	body = strings.TrimSpace(body)
	if !strings.HasPrefix(body, "[") {
		t.Errorf("body should be a JSON array, got: %s", body)
	}
}

func postTrigger(t *testing.T, base, body string) (int, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	resp, err := http.Post(base+"/api/v1/trigger", "application/json", reader) //nolint:noctx
	if err != nil {
		t.Fatalf("POST /api/v1/trigger: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func TestTriggerWithDuration(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, `{"duration":"10m"}`)
	if code != http.StatusAccepted {
		t.Errorf("status: got %d, want 202; body: %s", code, body)
	}
}

func TestTriggerWithSince(t *testing.T) {
	base := newTestServer(t)
	since := time.Now().Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	code, body := postTrigger(t, base, fmt.Sprintf(`{"since":"%s"}`, since))
	if code != http.StatusAccepted {
		t.Errorf("status: got %d, want 202; body: %s", code, body)
	}
}

func TestTriggerBothDurationAndSince(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, `{"duration":"10m","since":"2025-01-01T00:00:00Z"}`)
	if code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body: %s", code, body)
	}
	if !strings.Contains(body, "mutually exclusive") {
		t.Errorf("body should mention 'mutually exclusive', got: %s", body)
	}
}

func TestTriggerInvalidDuration(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, `{"duration":"notaduration"}`)
	if code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body: %s", code, body)
	}
	if !strings.Contains(body, "invalid duration") {
		t.Errorf("body should mention 'invalid duration', got: %s", body)
	}
}

func TestTriggerZeroDuration(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, `{"duration":"0s"}`)
	if code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body: %s", code, body)
	}
	if !strings.Contains(body, "positive") {
		t.Errorf("body should mention 'positive', got: %s", body)
	}
}

func TestTriggerFutureSince(t *testing.T) {
	base := newTestServer(t)
	future := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	code, body := postTrigger(t, base, fmt.Sprintf(`{"since":"%s"}`, future))
	if code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body: %s", code, body)
	}
	if !strings.Contains(body, "future") {
		t.Errorf("body should mention 'future', got: %s", body)
	}
}

func TestTriggerNoBodyUsesDefault(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, "")
	if code != http.StatusAccepted {
		t.Errorf("status: got %d, want 202; body: %s", code, body)
	}
}

func TestTriggerEmptyObjectUsesDefault(t *testing.T) {
	base := newTestServer(t)
	code, body := postTrigger(t, base, `{}`)
	if code != http.StatusAccepted {
		t.Errorf("status: got %d, want 202; body: %s", code, body)
	}
}
