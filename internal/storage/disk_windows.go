//go:build windows

package storage

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DefaultDataDir returns the default data directory for Windows.
func DefaultDataDir() string { return `C:\ProgramData\dashcap` }

// DefaultLockDir returns the default lock directory for Windows.
func DefaultLockDir() string { return `C:\ProgramData\dashcap\locks` }

type winDisk struct{}

func newPlatform() DiskOps { return &winDisk{} }

// FreeBytes returns available bytes using GetDiskFreeSpaceEx.
func (d *winDisk) FreeBytes(path string) (uint64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("FreeBytes: invalid path %q: %w", path, err)
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
		(*uint64)(unsafe.Pointer(&totalBytes)),
		(*uint64)(unsafe.Pointer(&totalFreeBytes)),
	)
	if err != nil {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx %q: %w", path, err)
	}
	return freeBytesAvailable, nil
}

// TotalBytes returns the total size of the filesystem containing path.
func (d *winDisk) TotalBytes(path string) (uint64, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, fmt.Errorf("TotalBytes: invalid path %q: %w", path, err)
	}
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	err = windows.GetDiskFreeSpaceEx(
		pathPtr,
		(*uint64)(unsafe.Pointer(&freeBytesAvailable)),
		(*uint64)(unsafe.Pointer(&totalBytes)),
		(*uint64)(unsafe.Pointer(&totalFreeBytes)),
	)
	if err != nil {
		return 0, fmt.Errorf("GetDiskFreeSpaceEx %q: %w", path, err)
	}
	return totalBytes, nil
}

// Preallocate extends file f to size bytes using SetFilePointer + SetEndOfFile.
func (d *winDisk) Preallocate(f *os.File, size int64) error {
	h := windows.Handle(f.Fd())
	// Move file pointer to desired size
	_, err := windows.Seek(h, size, 0)
	if err != nil {
		return fmt.Errorf("preallocate seek %q: %w", f.Name(), err)
	}
	// Set end of file at current position
	if err := windows.SetEndOfFile(h); err != nil {
		return fmt.Errorf("SetEndOfFile %q: %w", f.Name(), err)
	}
	// Reset pointer to beginning
	if _, err := windows.Seek(h, 0, 0); err != nil {
		return fmt.Errorf("preallocate seek-reset %q: %w", f.Name(), err)
	}
	return nil
}

// LockFile acquires an exclusive lock using LockFileEx.
func (d *winDisk) LockFile(f *os.File) error {
	h := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	// LOCKFILE_EXCLUSIVE_LOCK | LOCKFILE_FAIL_IMMEDIATELY
	const flags = 0x00000002 | 0x00000001
	if err := windows.LockFileEx(h, flags, 0, 1, 0, ol); err != nil {
		return fmt.Errorf("LockFileEx %q: %w", f.Name(), err)
	}
	return nil
}

// UnlockFile releases the lock using UnlockFileEx.
func (d *winDisk) UnlockFile(f *os.File) error {
	h := windows.Handle(f.Fd())
	ol := new(windows.Overlapped)
	if err := windows.UnlockFileEx(h, 0, 1, 0, ol); err != nil {
		return fmt.Errorf("UnlockFileEx %q: %w", f.Name(), err)
	}
	return nil
}
