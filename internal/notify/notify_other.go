//go:build !linux

// Package notify provides systemd readiness notification.
// On non-Linux platforms, all operations are no-ops.
package notify

// Ready is a no-op on non-Linux platforms.
func Ready() {}
