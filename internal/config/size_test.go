package config

import "testing"

func TestParseSizeGB(t *testing.T) {
	var n int64
	if err := ParseSize("2GB", &n); err != nil {
		t.Fatalf("ParseSize: %v", err)
	}
	if n != 2<<30 {
		t.Errorf("got %d, want %d", n, int64(2<<30))
	}
}

func TestParseSizeMB(t *testing.T) {
	var n int64
	if err := ParseSize("100MB", &n); err != nil {
		t.Fatalf("ParseSize: %v", err)
	}
	if n != 100<<20 {
		t.Errorf("got %d, want %d", n, int64(100<<20))
	}
}

func TestParseSizeKB(t *testing.T) {
	var n int64
	if err := ParseSize("512KB", &n); err != nil {
		t.Fatalf("ParseSize: %v", err)
	}
	if n != 512<<10 {
		t.Errorf("got %d, want %d", n, int64(512<<10))
	}
}

func TestParseSizePlainBytes(t *testing.T) {
	var n int64
	if err := ParseSize("1073741824", &n); err != nil {
		t.Fatalf("ParseSize: %v", err)
	}
	if n != 1073741824 {
		t.Errorf("got %d, want 1073741824", n)
	}
}

func TestParseSizeEmpty(t *testing.T) {
	var n int64 = 42
	if err := ParseSize("", &n); err != nil {
		t.Fatalf("ParseSize: %v", err)
	}
	if n != 42 {
		t.Errorf("empty string should not change dest, got %d", n)
	}
}

func TestParseSizeInvalid(t *testing.T) {
	var n int64
	if err := ParseSize("notasize", &n); err == nil {
		t.Error("expected error for invalid size, got nil")
	}
}
