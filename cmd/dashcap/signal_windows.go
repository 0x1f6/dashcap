//go:build windows

package main

import "dashcap/internal/trigger"

// setupSignalTrigger is a no-op on Windows (SIGUSR1 is not available).
func setupSignalTrigger(_ *trigger.Dispatcher) (stop func()) {
	return func() {}
}
