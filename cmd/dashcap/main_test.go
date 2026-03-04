package main

import (
	"regexp"
	"testing"
)

func TestParseSizeGB(t *testing.T) {
	var n int64
	if err := parseSize("2GB", &n); err != nil {
		t.Fatalf("parseSize: %v", err)
	}
	if n != 2<<30 {
		t.Errorf("got %d, want %d", n, int64(2<<30))
	}
}

func TestParseSizeMB(t *testing.T) {
	var n int64
	if err := parseSize("100MB", &n); err != nil {
		t.Fatalf("parseSize: %v", err)
	}
	if n != 100<<20 {
		t.Errorf("got %d, want %d", n, int64(100<<20))
	}
}

func TestParseSizeKB(t *testing.T) {
	var n int64
	if err := parseSize("512KB", &n); err != nil {
		t.Fatalf("parseSize: %v", err)
	}
	if n != 512<<10 {
		t.Errorf("got %d, want %d", n, int64(512<<10))
	}
}

func TestParseSizeInvalid(t *testing.T) {
	var n int64
	if err := parseSize("notasize", &n); err == nil {
		t.Error("expected error for invalid size, got nil")
	}
}

func TestDebugFlag(t *testing.T) {
	cmd := rootCmd()
	cmd.SetArgs([]string{"--debug", "--interface", "lo"})

	// Parse flags without executing RunE
	if err := cmd.ParseFlags([]string{"--debug", "--interface", "lo"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		t.Fatalf("GetBool(debug): %v", err)
	}
	if !debug {
		t.Error("expected --debug to be true")
	}
}

func TestDebugFlagDefault(t *testing.T) {
	cmd := rootCmd()

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		t.Fatalf("GetBool(debug): %v", err)
	}
	if debug {
		t.Error("expected --debug default to be false")
	}
}

func TestSanitize(t *testing.T) {
	input := "Wi-Fi 2.4GHz"
	got := sanitize(input)
	ok := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(got)
	if !ok {
		t.Errorf("sanitize(%q) = %q, contains disallowed characters", input, got)
	}
}
