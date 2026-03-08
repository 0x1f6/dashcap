package config

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// fileConfig is the intermediate YAML structure matching configs/dashcap.example.yaml.
type fileConfig struct {
	Interface  string          `yaml:"interface"`
	Buffer     fileBuffer      `yaml:"buffer"`
	Trigger    fileTrigger     `yaml:"trigger"`
	Safety     fileSafety      `yaml:"safety"`
	API        fileAPI         `yaml:"api"`
	Capture    fileCapture     `yaml:"capture"`
	Exclusions []fileExclusion `yaml:"exclusions"`
	Storage    fileStorage     `yaml:"storage"`
	Logging    fileLogging     `yaml:"logging"`
}

type fileBuffer struct {
	Size        string `yaml:"size"`
	SegmentSize string `yaml:"segment_size"`
}

type fileTrigger struct {
	DefaultDuration string `yaml:"default_duration"`
}

type fileSafety struct {
	MinFreeAfterAlloc string  `yaml:"min_free_after_alloc"`
	MinFreePercent    float64 `yaml:"min_free_percent"`
}

type fileAPI struct {
	TCPPort   int    `yaml:"tcp_port"`
	Token     string `yaml:"token"`
	NoAuth    bool   `yaml:"no_auth"`
	TLSCert   string `yaml:"tls_cert"`
	TLSKey    string `yaml:"tls_key"`
	TokenFile string `yaml:"token_file"`
}

type fileCapture struct {
	SnapLen     int  `yaml:"snaplen"`
	Promiscuous bool `yaml:"promiscuous"`
}

type fileExclusion struct {
	Name   string `yaml:"name"`
	Filter string `yaml:"filter"`
}

type fileStorage struct {
	DataDir string `yaml:"data_dir"`
}

type fileLogging struct {
	Level string `yaml:"level"`
}

// LoadFile reads a YAML config file and returns a Config with defaults
// overridden by the file values. Unknown keys cause an error (strict mode).
func LoadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Strict decode: reject unknown keys
	var fc fileConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&fc); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	cfg := Defaults()

	// Map file values onto cfg
	if fc.Interface != "" {
		cfg.Interface = fc.Interface
	}
	if fc.Buffer.Size != "" {
		if err := ParseSize(fc.Buffer.Size, &cfg.BufferSize); err != nil {
			return nil, fmt.Errorf("config buffer.size: %w", err)
		}
	}
	if fc.Buffer.SegmentSize != "" {
		if err := ParseSize(fc.Buffer.SegmentSize, &cfg.SegmentSize); err != nil {
			return nil, fmt.Errorf("config buffer.segment_size: %w", err)
		}
	}
	if fc.Trigger.DefaultDuration != "" {
		d, err := time.ParseDuration(fc.Trigger.DefaultDuration)
		if err != nil {
			return nil, fmt.Errorf("config trigger.default_duration: %w", err)
		}
		cfg.DefaultDuration = d
	}
	if fc.Safety.MinFreeAfterAlloc != "" {
		if err := ParseSize(fc.Safety.MinFreeAfterAlloc, &cfg.MinFreeAfterAlloc); err != nil {
			return nil, fmt.Errorf("config safety.min_free_after_alloc: %w", err)
		}
	}
	if fc.Safety.MinFreePercent != 0 {
		cfg.MinFreePercent = fc.Safety.MinFreePercent
	}
	if hasKey(data, "api", "tcp_port") {
		cfg.APIPort = fc.API.TCPPort
	}
	if fc.API.Token != "" {
		cfg.APIToken = fc.API.Token
	}
	if fc.API.NoAuth {
		cfg.APINoAuth = true
	}
	if fc.API.TLSCert != "" {
		cfg.TLSCert = fc.API.TLSCert
	}
	if fc.API.TLSKey != "" {
		cfg.TLSKey = fc.API.TLSKey
	}
	if fc.API.TokenFile != "" {
		cfg.TokenFile = fc.API.TokenFile
	}
	if fc.Capture.SnapLen != 0 {
		cfg.SnapLen = fc.Capture.SnapLen
	}
	// Promiscuous: fileCapture defaults to false, but our Config default is true.
	// We need to detect if the YAML explicitly set it. We do this by unmarshalling
	// into a map and checking for the key.
	if hasKey(data, "capture", "promiscuous") {
		cfg.Promiscuous = fc.Capture.Promiscuous
	}
	for _, ex := range fc.Exclusions {
		cfg.Exclusions = append(cfg.Exclusions, Exclusion(ex))
	}
	if fc.Storage.DataDir != "" {
		cfg.DataDir = fc.Storage.DataDir
	}
	if fc.Logging.Level != "" {
		switch fc.Logging.Level {
		case "debug":
			cfg.Debug = true
		case "info", "warn", "error":
			cfg.Debug = false
		default:
			return nil, fmt.Errorf("config logging.level: unknown level %q", fc.Logging.Level)
		}
	}

	return cfg, nil
}

// hasKey checks if a nested YAML key exists in the raw data.
func hasKey(data []byte, keys ...string) bool {
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return false
	}
	for i, key := range keys {
		v, ok := m[key]
		if !ok {
			return false
		}
		if i < len(keys)-1 {
			sub, ok := v.(map[string]interface{})
			if !ok {
				return false
			}
			m = sub
		}
	}
	return true
}

// DefaultConfigPath returns the platform-specific default config file path.
func DefaultConfigPath() string {
	if runtime.GOOS == "windows" {
		return `C:\ProgramData\dashcap\dashcap.yaml`
	}
	return "/etc/dashcap/dashcap.yaml"
}

// ResolveConfigFile determines which config file to use.
// If explicit is non-empty, it must exist (returns error if not found).
// If explicit is empty, the platform default is returned if it exists,
// or empty string if no config file is found.
func ResolveConfigFile(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("config file not found: %w", err)
		}
		return explicit, nil
	}
	defaultPath := DefaultConfigPath()
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	}
	return "", nil
}
