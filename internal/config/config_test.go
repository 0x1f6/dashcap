package config_test

import (
	"strings"
	"testing"

	"dashcap/internal/config"
)

func TestDefaults(t *testing.T) {
	cfg := config.Defaults()

	if cfg.BufferSize != 2*1024*1024*1024 {
		t.Errorf("BufferSize: got %d, want %d", cfg.BufferSize, 2*1024*1024*1024)
	}
	if cfg.SegmentSize != 100*1024*1024 {
		t.Errorf("SegmentSize: got %d, want %d", cfg.SegmentSize, 100*1024*1024)
	}
	if cfg.SegmentCount != 20 {
		t.Errorf("SegmentCount: got %d, want 20", cfg.SegmentCount)
	}
	if cfg.APIPort != 9800 {
		t.Errorf("APIPort: got %d, want 9800", cfg.APIPort)
	}
	if !cfg.Promiscuous {
		t.Error("Promiscuous should be true by default")
	}
}

func TestValidateDerivedSegmentCount(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.BufferSize = 1024 * 1024 * 1024 // 1 GB
	cfg.SegmentSize = 100 * 1024 * 1024 // 100 MB

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SegmentCount != 10 {
		t.Errorf("SegmentCount: got %d, want 10", cfg.SegmentCount)
	}
}

func TestValidateRejectsEmptyInterface(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty interface, got nil")
	}
	if !strings.Contains(err.Error(), "interface") {
		t.Errorf("error should mention 'interface', got: %v", err)
	}
}

func TestValidateRejectsBufferSmallerThanSegment(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.BufferSize = 50 * 1024 * 1024   // 50 MB
	cfg.SegmentSize = 100 * 1024 * 1024 // 100 MB

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when buffer_size < segment_size")
	}
}

func TestValidateRejectsTLSCertWithoutKey(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.TLSCert = "/path/to/cert.pem"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for TLS cert without key")
	}
	if !strings.Contains(err.Error(), "tls-cert") {
		t.Errorf("error should mention 'tls-cert', got: %v", err)
	}
}

func TestValidateRejectsTLSKeyWithoutCert(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.TLSKey = "/path/to/key.pem"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for TLS key without cert")
	}
	if !strings.Contains(err.Error(), "tls-cert") {
		t.Errorf("error should mention 'tls-cert', got: %v", err)
	}
}

func TestValidateAcceptsTLSBothSet(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.TLSCert = "/path/to/cert.pem"
	cfg.TLSKey = "/path/to/key.pem"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRejectsTooFewSegments(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.BufferSize = 100 * 1024 * 1024  // 100 MB
	cfg.SegmentSize = 100 * 1024 * 1024 // 100 MB → count = 1, which is < 2

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error when segment_count < 2")
	}
}

func TestValidateRejectsExclusionEmptyName(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.Exclusions = []config.Exclusion{{Name: "", Filter: "port 80"}}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for exclusion with empty name")
	}
	if !strings.Contains(err.Error(), "name must not be empty") {
		t.Errorf("error should mention 'name must not be empty', got: %v", err)
	}
}

func TestValidateRejectsExclusionEmptyFilter(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.Exclusions = []config.Exclusion{{Name: "test", Filter: ""}}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for exclusion with empty filter")
	}
	if !strings.Contains(err.Error(), "filter must not be empty") {
		t.Errorf("error should mention 'filter must not be empty', got: %v", err)
	}
}

func TestValidateAcceptsValidExclusions(t *testing.T) {
	cfg := config.Defaults()
	cfg.Interface = "eth0"
	cfg.Exclusions = []config.Exclusion{
		{Name: "dns", Filter: "udp port 53"},
		{Name: "backup", Filter: "host 10.0.0.50"},
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildBPFFilterEmpty(t *testing.T) {
	got := config.BuildBPFFilter(nil)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestBuildBPFFilterSingle(t *testing.T) {
	exclusions := []config.Exclusion{{Name: "dns", Filter: "udp port 53"}}
	got := config.BuildBPFFilter(exclusions)
	want := "not (udp port 53)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestBuildBPFFilterMultiple(t *testing.T) {
	exclusions := []config.Exclusion{
		{Name: "dns", Filter: "udp port 53"},
		{Name: "backup", Filter: "host 10.0.0.50 and port 443"},
	}
	got := config.BuildBPFFilter(exclusions)
	want := "not (udp port 53) and not (host 10.0.0.50 and port 443)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
