//go:build linux

package storage_test

import (
	"os"
	"testing"

	"dashcap/internal/storage"
)

func TestFreeBytes(t *testing.T) {
	d := storage.New()
	free, err := d.FreeBytes(os.TempDir())
	if err != nil {
		t.Fatalf("FreeBytes: %v", err)
	}
	if free == 0 {
		t.Error("FreeBytes returned 0, expected a positive value")
	}
}

func TestTotalBytes(t *testing.T) {
	d := storage.New()
	total, err := d.TotalBytes(os.TempDir())
	if err != nil {
		t.Fatalf("TotalBytes: %v", err)
	}
	if total == 0 {
		t.Error("TotalBytes returned 0, expected a positive value")
	}
	// Total must be >= free
	free, err := d.FreeBytes(os.TempDir())
	if err != nil {
		t.Fatalf("FreeBytes: %v", err)
	}
	if total < free {
		t.Errorf("TotalBytes (%d) < FreeBytes (%d)", total, free)
	}
}

func TestPreallocate(t *testing.T) {
	d := storage.New()
	f, err := os.CreateTemp(t.TempDir(), "prealloc-*.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	const size = 1 << 20 // 1 MB
	if err := d.Preallocate(f, size); err != nil {
		t.Fatalf("Preallocate: %v", err)
	}

	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != size {
		t.Errorf("file size: got %d, want %d", info.Size(), size)
	}
}

func TestLockFile(t *testing.T) {
	d := storage.New()
	f, err := os.CreateTemp(t.TempDir(), "lock-*.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if err := d.LockFile(f); err != nil {
		t.Fatalf("LockFile: %v", err)
	}
}

func TestFreeBytesInvalidPath(t *testing.T) {
	d := storage.New()
	_, err := d.FreeBytes("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path, got nil")
	}
}

func TestLockUnlockCycle(t *testing.T) {
	d := storage.New()
	f, err := os.CreateTemp(t.TempDir(), "lock-*.lock")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if err := d.LockFile(f); err != nil {
		t.Fatalf("LockFile: %v", err)
	}
	if err := d.UnlockFile(f); err != nil {
		t.Fatalf("UnlockFile: %v", err)
	}
}
