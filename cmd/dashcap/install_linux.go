//go:build linux

package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/user"

	"github.com/spf13/cobra"
)

//go:embed install_dist
var distFS embed.FS

type installTarget struct {
	embedPath string // path inside embed.FS
	destPath  string // absolute path on disk
}

var installTargets = []installTarget{
	{"install_dist/dashcap@.service", "/etc/systemd/system/dashcap@.service"},
	{"install_dist/dashcap.sysusers", "/usr/lib/sysusers.d/dashcap.conf"},
	{"install_dist/dashcap.tmpfiles", "/usr/lib/tmpfiles.d/dashcap.conf"},
}

func installServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-service",
		Short: "Install systemd service files and create dashcap user (requires root)",
		Long: `Install the dashcap systemd template unit, sysusers.d and tmpfiles.d
configs, and create the dashcap system user and group.

Requires root privileges. Idempotent — safe to re-run.

To uninstall:
  systemctl stop 'dashcap@*'
  systemctl disable 'dashcap@*'
  rm /etc/systemd/system/dashcap@.service
  rm /usr/lib/sysusers.d/dashcap.conf
  rm /usr/lib/tmpfiles.d/dashcap.conf
  systemctl daemon-reload
  userdel dashcap`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runInstallService()
		},
	}
}

func runInstallService() error {
	// 1. Root check.
	if os.Getuid() != 0 {
		return fmt.Errorf("install-service requires root privileges")
	}

	// 2. Create dashcap user/group if not present.
	if _, err := user.Lookup("dashcap"); err != nil {
		fmt.Println("Creating dashcap system user...")
		cmd := exec.Command("useradd", "--system", "--no-create-home", "--shell", "/usr/sbin/nologin", "dashcap")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
	} else {
		fmt.Println("User dashcap already exists, skipping.")
	}

	// 3. Write embedded files.
	for _, t := range installTargets {
		data, err := distFS.ReadFile(t.embedPath)
		if err != nil {
			return fmt.Errorf("read embedded %s: %w", t.embedPath, err)
		}
		if err := os.WriteFile(t.destPath, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", t.destPath, err)
		}
		fmt.Printf("Installed %s\n", t.destPath)
	}

	// 4. Create /etc/dashcap with correct ownership.
	if err := os.MkdirAll("/etc/dashcap", 0o750); err != nil {
		return fmt.Errorf("create /etc/dashcap: %w", err)
	}
	if err := chownDashcap("/etc/dashcap", true); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not set ownership on /etc/dashcap: %v\n", err)
	}

	// 5. Reload systemd.
	fmt.Println("Running systemctl daemon-reload...")
	cmd := exec.Command("systemctl", "daemon-reload")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}

	// 6. Print next steps.
	fmt.Println()
	fmt.Println("Installation complete. Next steps:")
	fmt.Println()
	fmt.Println("  # Enable and start for an interface:")
	fmt.Println("  systemctl enable --now dashcap@eth0")
	fmt.Println()
	fmt.Println("  # Allow a user to trigger captures:")
	fmt.Println("  usermod -aG dashcap <username>")
	fmt.Println()
	fmt.Println("  # View logs:")
	fmt.Println("  journalctl -u dashcap@eth0 -f")

	return nil
}
