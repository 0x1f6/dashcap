package main

import (
	"regexp"
	"testing"
)

func TestDebugFlag(t *testing.T) {
	cmd := rootCmd()
	cmd.SetArgs([]string{"--debug", "--interface", "lo"})

	// Parse flags without executing RunE
	if err := cmd.ParseFlags([]string{"--debug", "--interface", "lo"}); err != nil {
		t.Fatalf("ParseFlags: %v", err)
	}

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		t.Fatalf("GetBool(debug): %v", err)
	}
	if !debug {
		t.Error("expected --debug to be true")
	}
}

func TestDebugFlagDefault(t *testing.T) {
	cmd := rootCmd()

	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		t.Fatalf("GetBool(debug): %v", err)
	}
	if debug {
		t.Error("expected --debug default to be false")
	}
}

func TestSanitize(t *testing.T) {
	input := "Wi-Fi 2.4GHz"
	got := sanitize(input)
	ok := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(got)
	if !ok {
		t.Errorf("sanitize(%q) = %q, contains disallowed characters", input, got)
	}
}
