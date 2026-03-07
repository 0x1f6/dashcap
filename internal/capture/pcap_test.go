package capture_test

import (
	"strings"
	"testing"

	"github.com/google/gopacket/layers"

	"dashcap/internal/capture"
	"dashcap/internal/config"
)

func TestValidateExclusionsValid(t *testing.T) {
	exclusions := []config.Exclusion{
		{Name: "dns", Filter: "udp port 53"},
		{Name: "http", Filter: "tcp port 80"},
	}
	if err := capture.ValidateExclusions(exclusions, layers.LinkTypeEthernet, 65535); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateExclusionsInvalid(t *testing.T) {
	exclusions := []config.Exclusion{
		{Name: "good", Filter: "udp port 53"},
		{Name: "bad_rule", Filter: "invalid syntax !!!"},
	}
	err := capture.ValidateExclusions(exclusions, layers.LinkTypeEthernet, 65535)
	if err == nil {
		t.Fatal("expected error for invalid BPF, got nil")
	}
	if !strings.Contains(err.Error(), "bad_rule") {
		t.Errorf("error should mention 'bad_rule', got: %v", err)
	}
}

func TestValidateExclusionsEmpty(t *testing.T) {
	if err := capture.ValidateExclusions(nil, layers.LinkTypeEthernet, 65535); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
