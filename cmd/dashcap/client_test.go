package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUsePretty(t *testing.T) {
	tests := []struct {
		name   string
		pretty bool
		json   bool
		want   bool
	}{
		{"force pretty", true, false, true},
		{"force json", false, true, false},
		// When neither flag is set, result depends on TTY detection.
		// In test context stdout is typically not a TTY.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &clientFlags{pretty: tt.pretty, jsonOut: tt.json}
			got := f.usePretty()
			if got != tt.want {
				t.Errorf("usePretty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsePrettyAutoDetect(t *testing.T) {
	// In tests, stdout is not a TTY, so auto-detect should return false.
	f := &clientFlags{}
	if f.usePretty() {
		// Might be true if running in a real terminal, skip.
		fi, _ := os.Stdout.Stat()
		if fi.Mode()&os.ModeCharDevice == 0 {
			t.Error("usePretty() = true, want false (not a TTY)")
		}
	}
}

func TestClientTokenResolution_FileFallback(t *testing.T) {
	t.Setenv("DASHCAP_API_TOKEN", "")

	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "api-token")
	if err := os.WriteFile(tokenPath, []byte("file-token\n"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	f := &clientFlags{
		host:      "localhost",
		port:      9800,
		tokenFile: tokenPath,
	}
	// newClient should pick up the token from the file without panicking.
	c := f.newClient()
	_ = c
}

func TestClientTokenResolution_NoToken(t *testing.T) {
	t.Setenv("DASHCAP_API_TOKEN", "")

	f := &clientFlags{
		host:      "localhost",
		port:      9800,
		tokenFile: "/nonexistent/path",
	}
	// Should not panic; client created without token.
	c := f.newClient()
	_ = c
}
