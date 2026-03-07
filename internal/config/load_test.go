package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
interface: eth0
buffer:
  size: 5GB
  segment_size: 50MB
trigger:
  default_duration: 10m
safety:
  min_free_after_alloc: 2GB
  min_free_percent: 10
api:
  tcp_port: 8080
  token: mytoken
  no_auth: false
  tls_cert: /cert.pem
  tls_key: /key.pem
capture:
  snaplen: 1500
  promiscuous: false
storage:
  data_dir: /tmp/dashcap
logging:
  level: debug
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}

	if cfg.Interface != "eth0" {
		t.Errorf("Interface: got %q, want %q", cfg.Interface, "eth0")
	}
	if cfg.BufferSize != 5*1024*1024*1024 {
		t.Errorf("BufferSize: got %d, want %d", cfg.BufferSize, int64(5*1024*1024*1024))
	}
	if cfg.SegmentSize != 50*1024*1024 {
		t.Errorf("SegmentSize: got %d, want %d", cfg.SegmentSize, int64(50*1024*1024))
	}
	if cfg.DefaultDuration != 10*time.Minute {
		t.Errorf("DefaultDuration: got %v, want %v", cfg.DefaultDuration, 10*time.Minute)
	}
	if cfg.MinFreeAfterAlloc != 2*1024*1024*1024 {
		t.Errorf("MinFreeAfterAlloc: got %d, want %d", cfg.MinFreeAfterAlloc, int64(2*1024*1024*1024))
	}
	if cfg.MinFreePercent != 10 {
		t.Errorf("MinFreePercent: got %f, want 10", cfg.MinFreePercent)
	}
	if cfg.APIPort != 8080 {
		t.Errorf("APIPort: got %d, want 8080", cfg.APIPort)
	}
	if cfg.APIToken != "mytoken" {
		t.Errorf("APIToken: got %q, want %q", cfg.APIToken, "mytoken")
	}
	if cfg.TLSCert != "/cert.pem" {
		t.Errorf("TLSCert: got %q, want %q", cfg.TLSCert, "/cert.pem")
	}
	if cfg.TLSKey != "/key.pem" {
		t.Errorf("TLSKey: got %q, want %q", cfg.TLSKey, "/key.pem")
	}
	if cfg.SnapLen != 1500 {
		t.Errorf("SnapLen: got %d, want 1500", cfg.SnapLen)
	}
	if cfg.Promiscuous {
		t.Error("Promiscuous: got true, want false")
	}
	if cfg.DataDir != "/tmp/dashcap" {
		t.Errorf("DataDir: got %q, want %q", cfg.DataDir, "/tmp/dashcap")
	}
	if !cfg.Debug {
		t.Error("Debug: got false, want true (level=debug)")
	}
}

func TestLoadFileUnknownKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
interface: eth0
unknown_field: bad
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
	if !strings.Contains(err.Error(), "unknown_field") {
		t.Errorf("error should mention 'unknown_field', got: %v", err)
	}
}

func TestLoadFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `interface: [invalid yaml`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestLoadFileSizeParsing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
buffer:
  size: 1073741824
  segment_size: 50MB
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if cfg.BufferSize != 1073741824 {
		t.Errorf("BufferSize (plain bytes): got %d, want 1073741824", cfg.BufferSize)
	}
	if cfg.SegmentSize != 50*1024*1024 {
		t.Errorf("SegmentSize (50MB): got %d, want %d", cfg.SegmentSize, int64(50*1024*1024))
	}
}

func TestLoadFileDurationParsing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
trigger:
  default_duration: 30s
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if cfg.DefaultDuration != 30*time.Second {
		t.Errorf("DefaultDuration: got %v, want 30s", cfg.DefaultDuration)
	}
}

func TestLoadFileDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Minimal config — only interface set
	content := `interface: lo`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	// Unset fields should retain defaults
	defaults := Defaults()
	if cfg.BufferSize != defaults.BufferSize {
		t.Errorf("BufferSize should be default %d, got %d", defaults.BufferSize, cfg.BufferSize)
	}
	if cfg.APIPort != defaults.APIPort {
		t.Errorf("APIPort should be default %d, got %d", defaults.APIPort, cfg.APIPort)
	}
	if !cfg.Promiscuous {
		t.Error("Promiscuous should remain true (default) when not set in file")
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestResolveConfigFileExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("interface: lo"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ResolveConfigFile(path)
	if err != nil {
		t.Fatalf("ResolveConfigFile: %v", err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestResolveConfigFileExplicitMissing(t *testing.T) {
	_, err := ResolveConfigFile("/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error for missing explicit config, got nil")
	}
}

func TestResolveConfigFileNoDefault(t *testing.T) {
	// With empty explicit and no default file, should return empty string
	got, err := ResolveConfigFile("")
	if err != nil {
		t.Fatalf("ResolveConfigFile: %v", err)
	}
	// May or may not find a default — if the platform default doesn't exist, empty is fine
	if got != "" {
		// Only fail if the file doesn't actually exist
		if _, statErr := os.Stat(got); statErr != nil {
			t.Errorf("returned path %q that doesn't exist", got)
		}
	}
}
