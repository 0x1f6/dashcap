//go:build linux

// Package notify provides systemd readiness notification.
package notify

import (
	"log/slog"

	"github.com/coreos/go-systemd/v22/daemon"
)

// Ready sends sd_notify READY=1 to systemd. When not running under systemd
// (no NOTIFY_SOCKET), the call is a silent no-op.
func Ready() {
	sent, err := daemon.SdNotify(false, daemon.SdNotifyReady)
	if err != nil {
		slog.Warn("sd_notify failed", "error", err)
		return
	}
	if sent {
		slog.Info("sd_notify READY=1 sent")
	}
}
