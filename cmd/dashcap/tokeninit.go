package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

const defaultTokenFilePerms = 0o640
const defaultTokenDirPerms = 0o750

func tokenInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "token-init",
		Short: "Initialize the API token file (requires root)",
		Long: `Generate a cryptographically random API token and write it to the token file.
If the file already contains a token, it is left unchanged (idempotent).
This command is typically called via ExecStartPre in the systemd unit.`,
		SilenceUsage: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runTokenInit("/etc/dashcap/api-token")
		},
	}
}

// runTokenInit generates a token file if it does not exist or is empty.
// tokenPath is the file to write. Requires root for correct ownership.
func runTokenInit(tokenPath string) error {
	if os.Getuid() != 0 {
		return fmt.Errorf("token-init requires root privileges")
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(tokenPath)
	if err := os.MkdirAll(dir, defaultTokenDirPerms); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	// Set directory ownership to root:dashcap.
	if err := chownDashcap(dir, true); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not set ownership on %s: %v\n", dir, err)
	}

	// Check if token file already has content.
	data, err := os.ReadFile(tokenPath)
	if err == nil && len(data) > 0 {
		fmt.Fprintf(os.Stderr, "token file already exists: %s\n", tokenPath)
		return nil
	}

	// Generate new token.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(b)

	if err := os.WriteFile(tokenPath, []byte(token+"\n"), defaultTokenFilePerms); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}

	// Set ownership to root:dashcap (0640).
	if err := chownDashcap(tokenPath, false); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not set ownership on %s: %v\n", tokenPath, err)
	}

	fmt.Fprintf(os.Stderr, "token file created: %s\n", tokenPath)
	return nil
}

// chownDashcap sets ownership of path to root:dashcap.
// If ownerRoot is true, owner is root (UID 0). Otherwise owner is unchanged.
func chownDashcap(path string, ownerRoot bool) error {
	grp, err := user.LookupGroup("dashcap")
	if err != nil {
		return fmt.Errorf("lookup group dashcap: %w", err)
	}
	gid, err := strconv.Atoi(grp.Gid)
	if err != nil {
		return fmt.Errorf("parse gid: %w", err)
	}
	uid := -1 // no change
	if ownerRoot {
		uid = 0
	}
	return os.Chown(path, uid, gid)
}
