//go:build linux

package storage

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func preallocate(f *os.File, size int64) error {
	err := unix.Fallocate(int(f.Fd()), 0, 0, size)
	if err != nil {
		// Fallback: ftruncate extends the file (may create a sparse file)
		if err2 := f.Truncate(size); err2 != nil {
			return fmt.Errorf("preallocate %q: fallocate: %w; truncate: %w", f.Name(), err, err2)
		}
	}
	return nil
}
