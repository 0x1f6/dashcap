//go:build !linux

package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func installServiceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install-service",
		Short: "Install systemd service files (Linux only)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("install-service is only supported on Linux with systemd")
		},
	}
}
