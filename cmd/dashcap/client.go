package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"dashcap/internal/client"
)

// ANSI color codes for pretty output.
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

// clientFlags holds persistent flags for all client subcommands.
type clientFlags struct {
	host          string
	port          int
	token         string
	tls           bool
	tlsSkipVerify bool
	pretty        bool
	jsonOut       bool
}

func (f *clientFlags) newClient() *client.Client {
	token := f.token
	if token == "" {
		token = os.Getenv("DASHCAP_API_TOKEN")
	}
	return client.New(client.Options{
		Host:          f.host,
		Port:          f.port,
		Token:         token,
		TLS:           f.tls,
		TLSSkipVerify: f.tlsSkipVerify,
	})
}

// usePretty returns true if pretty output should be used.
func (f *clientFlags) usePretty() bool {
	if f.pretty {
		return true
	}
	if f.jsonOut {
		return false
	}
	// Auto-detect: pretty if stdout is a TTY.
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func clientCmd() *cobra.Command {
	flags := &clientFlags{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Interact with a running dashcap instance via its REST API",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			if flags.pretty && flags.jsonOut {
				return fmt.Errorf("--pretty and --json are mutually exclusive")
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&flags.host, "host", "localhost", "API server host")
	cmd.PersistentFlags().IntVar(&flags.port, "port", 9800, "API server port")
	cmd.PersistentFlags().StringVar(&flags.token, "token", "", "Bearer token (default: $DASHCAP_API_TOKEN)")
	cmd.PersistentFlags().BoolVar(&flags.tls, "tls", false, "Use HTTPS")
	cmd.PersistentFlags().BoolVar(&flags.tlsSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")
	cmd.PersistentFlags().BoolVar(&flags.pretty, "pretty", false, "Force human-readable output")
	cmd.PersistentFlags().BoolVar(&flags.jsonOut, "json", false, "Force JSON output")

	cmd.AddCommand(
		healthCmd(flags),
		statusCmd(flags),
		triggerCmd(flags),
		triggersCmd(flags),
		ringCmd(flags),
	)
	return cmd
}

func healthCmd(flags *clientFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := flags.newClient()
			resp, err := c.Health()
			if err != nil {
				return err
			}
			if flags.usePretty() {
				fmt.Printf("%s%sHealth:%s %s%s%s\n", colorBold, colorCyan, colorReset, colorGreen, resp.Status, colorReset)
				return nil
			}
			return writeJSON(resp)
		},
	}
}

func statusCmd(flags *clientFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show capture status",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := flags.newClient()
			resp, err := c.Status()
			if err != nil {
				return err
			}
			if flags.usePretty() {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintf(w, "%s%sInterface:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Interface)
				_, _ = fmt.Fprintf(w, "%s%sUptime:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Uptime)
				_, _ = fmt.Fprintf(w, "%s%sSegments:%s\t%d\n", colorBold, colorCyan, colorReset, resp.SegmentCount)
				_, _ = fmt.Fprintf(w, "%s%sTotal Packets:%s\t%d\n", colorBold, colorCyan, colorReset, resp.TotalPackets)
				_, _ = fmt.Fprintf(w, "%s%sTotal Bytes:%s\t%d\n", colorBold, colorCyan, colorReset, resp.TotalBytes)
				return w.Flush()
			}
			return writeJSON(resp)
		},
	}
}

func triggerCmd(flags *clientFlags) *cobra.Command {
	var duration, since string

	cmd := &cobra.Command{
		Use:   "trigger",
		Short: "Trigger a capture save",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if duration != "" && since != "" {
				return fmt.Errorf("--duration and --since are mutually exclusive")
			}
			c := flags.newClient()
			resp, err := c.Trigger(client.TriggerRequest{
				Duration: duration,
				Since:    since,
			})
			if err != nil {
				return err
			}
			if flags.usePretty() {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintf(w, "%s%sTrigger ID:%s\t%s\n", colorBold, colorCyan, colorReset, resp.ID)
				_, _ = fmt.Fprintf(w, "%s%sTimestamp:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Timestamp)
				_, _ = fmt.Fprintf(w, "%s%sSource:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Source)
				_, _ = fmt.Fprintf(w, "%s%sStatus:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Status)
				if resp.SavedPath != "" {
					_, _ = fmt.Fprintf(w, "%s%sSaved Path:%s\t%s\n", colorBold, colorCyan, colorReset, resp.SavedPath)
				}
				if resp.Warning != "" {
					_, _ = fmt.Fprintf(w, "%s%sWarning:%s\t%s%s%s\n", colorBold, colorYellow, colorReset, colorYellow, resp.Warning, colorReset)
				}
				if resp.Error != "" {
					_, _ = fmt.Fprintf(w, "%s%sError:%s\t%s\n", colorBold, colorCyan, colorReset, resp.Error)
				}
				return w.Flush()
			}
			return writeJSON(resp)
		},
	}
	cmd.Flags().StringVar(&duration, "duration", "", "Time window duration (e.g. 30s, 5m)")
	cmd.Flags().StringVar(&since, "since", "", "Absolute start time (RFC 3339)")
	return cmd
}

func triggersCmd(flags *clientFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "triggers",
		Short: "List trigger history",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := flags.newClient()
			resp, err := c.Triggers()
			if err != nil {
				return err
			}
			if flags.usePretty() {
				if len(resp) == 0 {
					fmt.Println("No triggers recorded.")
					return nil
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintf(w, "%s%sID\tTIMESTAMP\tSOURCE\tSTATUS\tPATH%s\n", colorBold, colorCyan, colorReset)
				for _, t := range resp {
					_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", t.ID, t.Timestamp, t.Source, t.Status, t.SavedPath)
				}
				return w.Flush()
			}
			return writeJSON(resp)
		},
	}
}

func ringCmd(flags *clientFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "ring",
		Short: "Show ring buffer segments",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			c := flags.newClient()
			resp, err := c.Ring()
			if err != nil {
				return err
			}
			if flags.usePretty() {
				if len(resp) == 0 {
					fmt.Println("No ring segments.")
					return nil
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				_, _ = fmt.Fprintf(w, "%s%sINDEX\tPACKETS\tBYTES\tSTART\tEND%s\n", colorBold, colorCyan, colorReset)
				for _, s := range resp {
					_, _ = fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\n", s.Index, s.Packets, s.Bytes, s.StartTime, s.EndTime)
				}
				return w.Flush()
			}
			return writeJSON(resp)
		},
	}
}

func writeJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
