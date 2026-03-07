package config

import "fmt"

// ParseSize parses human-readable size strings like "2GB", "500MB", "100KB"
// into bytes. Plain integer strings are treated as bytes.
func ParseSize(s string, dest *int64) error {
	if s == "" {
		return nil
	}
	suffixes := map[string]int64{
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}
	for suffix, mult := range suffixes {
		if len(s) > len(suffix) && s[len(s)-len(suffix):] == suffix {
			var n int64
			if _, err := fmt.Sscan(s[:len(s)-len(suffix)], &n); err != nil {
				return fmt.Errorf("invalid size %q: %w", s, err)
			}
			*dest = n * mult
			return nil
		}
	}
	// Plain number in bytes
	var n int64
	if _, err := fmt.Sscan(s, &n); err != nil {
		return fmt.Errorf("invalid size %q: %w", s, err)
	}
	*dest = n
	return nil
}
