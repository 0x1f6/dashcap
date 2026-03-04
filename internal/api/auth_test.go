package api_test

import (
	"context"
	"encoding/hex"
	"fmt"
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

const testToken = "test-secret-token"

// newAuthTestServer creates a server with auth enabled using testToken.
func newAuthTestServer(t *testing.T) string {
	t.Helper()

	cfg := &config.Config{
		Interface:         "test0",
		BufferSize:        3 * 1024,
		SegmentSize:       1024,
		SegmentCount:      3,
		DataDir:           t.TempDir(),
		SavedDir:          "saved",
		MinFreeAfterAlloc: 0,
		DefaultDuration:       5 * time.Minute,
		APIPort:           0,
		APIToken:          testToken,
	}

	ring, err := buffer.NewRingManager(cfg, apiTestDisk{}, layers.LinkTypeEthernet)
	if err != nil {
		t.Fatalf("NewRingManager: %v", err)
	}
	t.Cleanup(func() { _ = ring.Close() })

	disp := trigger.NewDispatcher(cfg, ring)

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

func doRequest(t *testing.T, method, url, token string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := os.ReadFile("/dev/null") // read nothing, just for discard
	_ = b
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	return resp.StatusCode, string(buf[:n])
}

func TestAuthHealthExempt(t *testing.T) {
	base := newAuthTestServer(t)

	// Health should work without auth
	code, body := doRequest(t, "GET", base+"/api/v1/health", "")
	if code != http.StatusOK {
		t.Errorf("health status: got %d, want 200", code)
	}
	if !strings.Contains(body, "ok") {
		t.Errorf("health body should contain 'ok', got: %s", body)
	}
}

func TestAuthValidToken(t *testing.T) {
	base := newAuthTestServer(t)

	code, body := doRequest(t, "GET", base+"/api/v1/status", testToken)
	if code != http.StatusOK {
		t.Errorf("status: got %d, want 200", code)
	}
	if !strings.Contains(body, "test0") {
		t.Errorf("body should contain 'test0', got: %s", body)
	}
}

func TestAuthMissingHeader(t *testing.T) {
	base := newAuthTestServer(t)

	code, body := doRequest(t, "GET", base+"/api/v1/status", "")
	if code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", code)
	}
	if !strings.Contains(body, "unauthorized") {
		t.Errorf("body should contain 'unauthorized', got: %s", body)
	}
}

func TestAuthInvalidToken(t *testing.T) {
	base := newAuthTestServer(t)

	code, body := doRequest(t, "GET", base+"/api/v1/status", "wrong-token")
	if code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", code)
	}
	if !strings.Contains(body, "unauthorized") {
		t.Errorf("body should contain 'unauthorized', got: %s", body)
	}
}

func TestAuthTriggerRequiresToken(t *testing.T) {
	base := newAuthTestServer(t)

	// Without token: 401
	code, _ := doRequest(t, "POST", base+"/api/v1/trigger", "")
	if code != http.StatusUnauthorized {
		t.Errorf("trigger without token: got %d, want 401", code)
	}

	// With token: 202
	code, _ = doRequest(t, "POST", base+"/api/v1/trigger", testToken)
	if code != http.StatusAccepted {
		t.Errorf("trigger with token: got %d, want 202", code)
	}
}

func TestAuthNoAuthDisabled(t *testing.T) {
	// When APIToken is empty (no-auth mode), all requests should work
	base := newTestServer(t) // uses default config with empty token

	code, _ := doRequest(t, "GET", base+"/api/v1/status", "")
	if code != http.StatusOK {
		t.Errorf("status without auth: got %d, want 200", code)
	}
}

func TestTokenFormat(t *testing.T) {
	// Verify hex token properties: a 64-char hex string decodes to 32 bytes
	token := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	b, err := hex.DecodeString(token)
	if err != nil {
		t.Fatalf("decode hex: %v", err)
	}
	if len(b) != 32 {
		t.Errorf("decoded length: got %d, want 32", len(b))
	}
	if len(token) != 64 {
		t.Errorf("token length: got %d, want 64", len(token))
	}
}
