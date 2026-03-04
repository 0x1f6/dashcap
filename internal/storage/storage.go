// Package storage provides platform-specific disk operations used by dashcap.
package storage

import "os"

// DiskOps abstracts platform-specific disk and file operations.
type DiskOps interface {
	// FreeBytes returns the number of free bytes available on the filesystem
	// that contains the given path.
	FreeBytes(path string) (uint64, error)

	// TotalBytes returns the total size in bytes of the filesystem
	// that contains the given path.
	TotalBytes(path string) (uint64, error)

	// Preallocate ensures that file f has at least size bytes allocated on disk.
	// The file offset is not changed. On platforms without native preallocation
	// this may write zeroes to extend the file.
	Preallocate(f *os.File, size int64) error

	// LockFile acquires an exclusive advisory lock on f.
	// Returns an error if the lock is already held by another process.
	LockFile(f *os.File) error

	// UnlockFile releases the advisory lock on f.
	UnlockFile(f *os.File) error
}

// New returns the platform-appropriate DiskOps implementation.
func New() DiskOps {
	return newPlatform()
}
