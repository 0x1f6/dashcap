// Package config defines dashcap runtime configuration.
package config

import (
	"fmt"
	"time"
)

// Exclusion defines a named BPF filter expression used to exclude traffic from capture.
type Exclusion struct {
	Name   string // Human-readable identifier (e.g. "backup_traffic")
	Filter string // BPF expression in tcpdump syntax (e.g. "host 10.0.0.50 and port 443")
}

// Config holds all runtime configuration for a dashcap instance.
type Config struct {
	// Interface is the network interface to capture on (e.g. "eth0", "Wi-Fi").
	Interface string

	// Buffer settings
	BufferSize   int64 // Total ring buffer size in bytes
	SegmentSize  int64 // Size of each pcapng segment file in bytes
	SegmentCount int   // Derived: BufferSize / SegmentSize

	// Data directories
	DataDir  string // Base data directory for this instance
	SavedDir string // Sub-directory for triggered captures (relative to DataDir)

	// API
	APIPort   int    // TCP port for REST API (0 = disabled)
	APIToken  string // Bearer token for API auth (auto-generated if empty)
	APINoAuth bool   // Disable API authentication entirely

	// TLS
	TLSCert string // Path to TLS certificate file
	TLSKey  string // Path to TLS private key file

	// Trigger windows
	DefaultDuration time.Duration // Default time window to save on trigger

	// Capture settings
	SnapLen     int // 0 = full packets
	Promiscuous bool

	// Exclusion filters
	Exclusions      []Exclusion // BPF exclusion filter rules
	ActiveBPFFilter string      // Combined BPF expression applied at startup (set by run())

	// Disk safety
	MinFreeAfterAlloc int64   // Minimum free bytes after preallocation
	MinFreePercent    float64 // Minimum free percentage after preallocation

	// Logging
	Debug bool // Enable debug-level logging
}

// Defaults returns a Config populated with sensible Phase 1 defaults.
func Defaults() *Config {
	return &Config{
		BufferSize:        2 * 1024 * 1024 * 1024, // 2 GB
		SegmentSize:       100 * 1024 * 1024,      // 100 MB
		SegmentCount:      20,
		SavedDir:          "saved",
		APIPort:           9800,
		DefaultDuration:   5 * time.Minute,
		SnapLen:           0,
		Promiscuous:       true,
		MinFreeAfterAlloc: 1 * 1024 * 1024 * 1024, // 1 GB
		MinFreePercent:    5,
	}
}

// Validate returns an error if the configuration is invalid.
func (c *Config) Validate() error {
	if c.Interface == "" {
		return errorf("interface must be set")
	}
	if c.SegmentSize <= 0 {
		return errorf("segment_size must be positive")
	}
	if c.BufferSize < c.SegmentSize {
		return errorf("buffer_size must be >= segment_size")
	}
	c.SegmentCount = int(c.BufferSize / c.SegmentSize)
	if c.SegmentCount < 2 {
		return errorf("buffer must contain at least 2 segments")
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return errorf("--tls-cert and --tls-key must both be set")
	}
	for i, ex := range c.Exclusions {
		if ex.Name == "" {
			return errorf(fmt.Sprintf("exclusion #%d: name must not be empty", i+1))
		}
		if ex.Filter == "" {
			return errorf(fmt.Sprintf("exclusion %q: filter must not be empty", ex.Name))
		}
	}
	return nil
}

// BuildBPFFilter combines exclusions into a single BPF expression.
// Returns empty string if there are no exclusions.
// Each exclusion is negated and ANDed: "not (<expr1>) and not (<expr2>)".
func BuildBPFFilter(exclusions []Exclusion) string {
	if len(exclusions) == 0 {
		return ""
	}
	parts := make([]string, len(exclusions))
	for i, ex := range exclusions {
		parts[i] = "not (" + ex.Filter + ")"
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += " and " + p
	}
	return result
}

// errorf is a simple error helper to avoid importing fmt in this package alone.
func errorf(msg string) error {
	return &configError{msg: msg}
}

type configError struct{ msg string }

func (e *configError) Error() string { return "dashcap config: " + e.msg }
