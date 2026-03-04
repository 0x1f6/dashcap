//go:build darwin

package storage

import (
	"fmt"
	"os"
)

func preallocate(f *os.File, size int64) error {
	// On Darwin use ftruncate as a simple fallback.
	// F_PREALLOCATE via fcntl would be more efficient but requires unsafe.
	if err := f.Truncate(size); err != nil {
		return fmt.Errorf("preallocate %q: %w", f.Name(), err)
	}
	return nil
}
