//go:build linux || darwin

package storage

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// DefaultDataDir returns the default data directory for Unix platforms.
func DefaultDataDir() string { return "/var/lib/dashcap" }

// DefaultLockDir returns the default lock directory for Unix platforms.
func DefaultLockDir() string { return "/run/dashcap" }

type unixDisk struct{}

func newPlatform() DiskOps { return &unixDisk{} }

// FreeBytes returns available bytes on the filesystem containing path.
func (d *unixDisk) FreeBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("statfs %q: %w", path, err)
	}
	//nolint:unconvert // Bavail type varies between Linux (uint64) and Darwin (uint32 on older SDKs)
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}

// TotalBytes returns the total size of the filesystem containing path.
func (d *unixDisk) TotalBytes(path string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("statfs %q: %w", path, err)
	}
	//nolint:unconvert // Blocks type varies between Linux and Darwin
	return uint64(stat.Blocks) * uint64(stat.Bsize), nil
}

// Preallocate allocates size bytes for file f using fallocate on Linux and
// fcntl(F_PREALLOCATE) on Darwin, falling back to ftruncate if unsupported.
func (d *unixDisk) Preallocate(f *os.File, size int64) error {
	return preallocate(f, size)
}

// LockFile acquires an exclusive flock on f.
func (d *unixDisk) LockFile(f *os.File) error {
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		return fmt.Errorf("flock %q: %w", f.Name(), err)
	}
	return nil
}

// UnlockFile releases the flock on f.
func (d *unixDisk) UnlockFile(f *os.File) error {
	if err := unix.Flock(int(f.Fd()), unix.LOCK_UN); err != nil {
		return fmt.Errorf("funlock %q: %w", f.Name(), err)
	}
	return nil
}
