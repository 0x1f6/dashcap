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

func TestSanitize(t *testing.T) {
	input := "Wi-Fi 2.4GHz"
	got := sanitize(input)
	ok := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(got)
	if !ok {
		t.Errorf("sanitize(%q) = %q, contains disallowed characters", input, got)
	}
}
