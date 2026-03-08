package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunTokenInit_CreatesNewToken(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "api-token")

	if err := runTokenInit(tokenPath); err != nil {
		t.Fatalf("runTokenInit: %v", err)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// 64 hex chars + newline
	if len(data) != 65 {
		t.Errorf("token file length: got %d, want 65", len(data))
	}
}

func TestRunTokenInit_PreservesExisting(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "api-token")

	// Write an existing token.
	existing := "existing-token-value\n"
	if err := os.WriteFile(tokenPath, []byte(existing), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := runTokenInit(tokenPath); err != nil {
		t.Fatalf("runTokenInit: %v", err)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != existing {
		t.Errorf("token changed: got %q, want %q", string(data), existing)
	}
}

func TestRunTokenInit_RegeneratesEmpty(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root")
	}
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "api-token")

	// Create empty file.
	if err := os.WriteFile(tokenPath, []byte{}, 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := runTokenInit(tokenPath); err != nil {
		t.Fatalf("runTokenInit: %v", err)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Error("token file should not be empty after regeneration")
	}
}

func TestRunTokenInit_RequiresRoot(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("test only meaningful as non-root")
	}
	err := runTokenInit("/tmp/dashcap-test-token")
	if err == nil {
		t.Fatal("expected error for non-root, got nil")
	}
}
