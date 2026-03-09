//go:build !windows

package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"dashcap/internal/trigger"
)

// setupSignalTrigger registers a SIGUSR1 handler that triggers a
// default-duration capture save. Returns a cleanup function.
func setupSignalTrigger(d *trigger.Dispatcher) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	go func() {
		for range ch {
			slog.Info("SIGUSR1 received, triggering capture save")
			if _, err := d.Trigger("signal", trigger.TriggerOpts{}); err != nil {
				slog.Debug("signal trigger rejected", "error", err)
			}
		}
	}()
	return func() { signal.Stop(ch) }
}
