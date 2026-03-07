// dashcap — network packet dashcam
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"dashcap/internal/api"
	"dashcap/internal/buffer"
	"dashcap/internal/capture"
	"dashcap/internal/config"
	"dashcap/internal/storage"
	"dashcap/internal/trigger"
)

// Build-time variables injected via -ldflags.
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cfg := config.Defaults()

	var (
		configFile     string
		bufferSizeStr  string
		segmentSizeStr string
	)

	cmd := &cobra.Command{
		Use:   "dashcap",
		Short: "Network packet dashcam — continuous capture with on-demand persistence",
		Long: fmt.Sprintf(
			"dashcap v%s (%s built %s)\n\nContinuously captures packets into a pre-allocated ring buffer.\nTrigger a save via the REST API to preserve a capture window.",
			version, commit, buildTime,
		),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Load config file (if any)
			cfgPath, err := config.ResolveConfigFile(configFile)
			if err != nil {
				return err
			}
			if cfgPath != "" {
				fileCfg, err := config.LoadFile(cfgPath)
				if err != nil {
					return err
				}
				*cfg = *fileCfg
				slog.Info("config loaded", "path", cfgPath)
			}

			// Apply CLI flags that were explicitly set (override config file)
			applyFlags(cmd, cfg)

			// Parse size strings for flags that were set via CLI
			if cmd.Flags().Changed("buffer-size") {
				if err := config.ParseSize(bufferSizeStr, &cfg.BufferSize); err != nil {
					return fmt.Errorf("--buffer-size: %w", err)
				}
			}
			if cmd.Flags().Changed("segment-size") {
				if err := config.ParseSize(segmentSizeStr, &cfg.SegmentSize); err != nil {
					return fmt.Errorf("--segment-size: %w", err)
				}
			}

			if cfg.Interface == "" {
				return fmt.Errorf("interface must be set (use -i flag or config file)")
			}

			return run(cfg)
		},
	}

	cmd.Flags().StringVar(&configFile, "config", "", "Path to YAML config file (default: "+config.DefaultConfigPath()+")")
	cmd.Flags().StringVarP(&cfg.Interface, "interface", "i", "", "Network interface to capture on")
	cmd.Flags().StringVar(&bufferSizeStr, "buffer-size", "2GB", "Total ring buffer size (e.g. 2GB, 500MB)")
	cmd.Flags().StringVar(&segmentSizeStr, "segment-size", "100MB", "Size of each ring segment (e.g. 100MB)")
	cmd.Flags().StringVar(&cfg.DataDir, "data-dir", "", "Data directory (default: "+storage.DefaultDataDir()+"/<interface>)")
	cmd.Flags().IntVar(&cfg.APIPort, "api-port", cfg.APIPort, "TCP port for REST API (0 = disabled)")
	cmd.Flags().DurationVar(&cfg.DefaultDuration, "default-duration", cfg.DefaultDuration, "Default time window to save on trigger")
	cmd.Flags().BoolVar(&cfg.Promiscuous, "promiscuous", cfg.Promiscuous, "Enable promiscuous mode")
	cmd.Flags().IntVar(&cfg.SnapLen, "snaplen", cfg.SnapLen, "Snapshot length (0 = full packet)")
	cmd.Flags().StringVar(&cfg.APIToken, "api-token", "", "Bearer token for API auth (default: auto-generated)")
	cmd.Flags().BoolVar(&cfg.APINoAuth, "no-auth", false, "Disable API authentication")
	cmd.Flags().StringVar(&cfg.TLSCert, "tls-cert", "", "Path to TLS certificate file")
	cmd.Flags().StringVar(&cfg.TLSKey, "tls-key", "", "Path to TLS private key file")
	cmd.Flags().BoolVar(&cfg.Debug, "debug", false, "Enable debug-level logging")

	cmd.AddCommand(versionCmd())
	cmd.AddCommand(clientCmd())
	return cmd
}

// applyFlags overrides cfg with CLI flag values that the user explicitly set.
func applyFlags(cmd *cobra.Command, cfg *config.Config) {
	f := cmd.Flags()
	if f.Changed("interface") {
		cfg.Interface, _ = f.GetString("interface")
	}
	if f.Changed("data-dir") {
		cfg.DataDir, _ = f.GetString("data-dir")
	}
	if f.Changed("api-port") {
		cfg.APIPort, _ = f.GetInt("api-port")
	}
	if f.Changed("default-duration") {
		cfg.DefaultDuration, _ = f.GetDuration("default-duration")
	}
	if f.Changed("promiscuous") {
		cfg.Promiscuous, _ = f.GetBool("promiscuous")
	}
	if f.Changed("snaplen") {
		cfg.SnapLen, _ = f.GetInt("snaplen")
	}
	if f.Changed("api-token") {
		cfg.APIToken, _ = f.GetString("api-token")
	}
	if f.Changed("no-auth") {
		cfg.APINoAuth, _ = f.GetBool("no-auth")
	}
	if f.Changed("tls-cert") {
		cfg.TLSCert, _ = f.GetString("tls-cert")
	}
	if f.Changed("tls-key") {
		cfg.TLSKey, _ = f.GetString("tls-key")
	}
	if f.Changed("debug") {
		cfg.Debug, _ = f.GetBool("debug")
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("dashcap %s (%s) built %s\n", version, commit, buildTime)
		},
	}
}

func run(cfg *config.Config) error {
	// Configure logger
	logLevel := slog.LevelInfo
	if cfg.Debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})))

	// Apply DataDir default
	if cfg.DataDir == "" {
		cfg.DataDir = filepath.Join(storage.DefaultDataDir(), sanitize(cfg.Interface))
	}
	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Validate config (also computes SegmentCount)
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Resolve API token: --api-token flag > DASHCAP_API_TOKEN env > auto-generated
	if !cfg.APINoAuth && cfg.APIPort > 0 {
		if cfg.APIToken == "" {
			cfg.APIToken = os.Getenv("DASHCAP_API_TOKEN")
		}
		if cfg.APIToken == "" {
			tok, err := generateToken()
			if err != nil {
				return fmt.Errorf("generate api token: %w", err)
			}
			cfg.APIToken = tok
		}
		slog.Info("API token generated", "token", cfg.APIToken)
	}

	slog.Info("dashcap starting", "version", version, "interface", cfg.Interface)
	slog.Info("ring buffer configured", "segments", cfg.SegmentCount, "segment_mb", cfg.SegmentSize/1024/1024, "total_mb", int64(cfg.SegmentCount)*cfg.SegmentSize/1024/1024)

	// Acquire interface lock
	disk := storage.New()
	lockPath := filepath.Join(storage.DefaultLockDir(), "dashcap-"+sanitize(cfg.Interface)+".lock")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o750); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}
	lockFile, err := os.OpenFile(lockPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer func() { _ = lockFile.Close() }()
	if err := disk.LockFile(lockFile); err != nil {
		return fmt.Errorf("interface %s already in use (lock: %s): %w", cfg.Interface, lockPath, err)
	}
	defer func() { _ = disk.UnlockFile(lockFile) }()
	defer func() { _ = os.Remove(lockPath) }()

	// Open capture source
	src, err := capture.OpenLive(cfg.Interface, cfg.SnapLen, cfg.Promiscuous)
	if err != nil {
		return fmt.Errorf("open capture on %s: %w", cfg.Interface, err)
	}

	// Resolve hostname once for pcapng SHB metadata
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	shb := buffer.SHBInfo{
		Version:   version,
		Hostname:  hostname,
		Interface: cfg.Interface,
	}

	// Initialise ring manager (disk safety check + prealloc)
	ring, err := buffer.NewRingManager(cfg, disk, src.LinkType(), shb)
	if err != nil {
		return fmt.Errorf("ring manager: %w", err)
	}
	defer func() { _ = ring.Close() }()

	slog.Info("ring pre-allocated", "path", cfg.DataDir)

	// Trigger dispatcher
	dispatcher := trigger.NewDispatcher(cfg, ring, shb)

	// REST API server
	var apiServer *api.Server
	if cfg.APIPort > 0 {
		if !cfg.APINoAuth && cfg.TLSCert == "" {
			slog.Warn("API auth enabled without TLS — tokens sent in cleartext")
		}
		apiServer = api.New(cfg, ring, dispatcher)
		go func() {
			proto := "HTTP"
			if cfg.TLSCert != "" {
				proto = "HTTPS"
			}
			slog.Info("REST API listening", "port", cfg.APIPort, "proto", proto)
			if err := apiServer.ListenAndServe(); err != nil {
				slog.Info("API server stopped", "error", err)
			}
		}()
	}

	// Capture loop
	captureDone := make(chan struct{})
	go func() {
		captureLoop(src, ring, cfg.SegmentSize)
		close(captureDone)
	}()

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	sig := <-sigCh
	slog.Info("received signal, shutting down", "signal", sig)

	// Graceful shutdown: stop capture source first so the loop exits,
	// then wait for it to finish before closing the ring.
	src.Close()
	<-captureDone

	if apiServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = apiServer.Shutdown(ctx)
	}

	return nil
}

// captureLoop reads packets and writes them to the ring buffer, rotating
// segments when they exceed segmentSize bytes.
func captureLoop(src capture.Source, ring *buffer.RingManager, segmentSize int64) {
	for {
		data, ci, err := src.ReadPacketData()
		if err != nil {
			// io.EOF means the source was closed — exit the loop.
			if errors.Is(err, io.EOF) {
				return
			}
			slog.Debug("capture read error", "error", err)
			continue
		}

		w := ring.CurrentWriter()
		if err := w.WritePacket(ci, data); err != nil {
			slog.Debug("write packet error", "error", err)
			continue
		}

		if w.BytesWritten() >= segmentSize {
			if err := ring.Rotate(); err != nil {
				slog.Debug("ring rotate error", "error", err)
			}
		}
	}
}

// sanitize replaces characters that are unsafe in file paths.
func sanitize(s string) string {
	out := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			out[i] = c
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}

// generateToken returns a cryptographically random 64-character hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
