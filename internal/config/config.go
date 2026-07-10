package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type FileConfig struct {
	ToleranceCents            *int64         `json:"tolerance_cents"`
	MatchingWindowDays        *int           `json:"matching_window_days"`
	OutputFormat              *string        `json:"output_format"`
	PartitionBy               *string        `json:"partition_by"`
	IncludePendingSettlements *bool          `json:"include_pending_settlements"`
	QueryLimit                *int           `json:"query_limit"`
	Filters                   map[string]any `json:"filters"`
}

// This function will load the config file , however a missing or empty file will not initialize and error
func LoadConfig(path string) (FileConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return FileConfig{}, nil
	}
	if err != nil {
		return FileConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// ParseConfig parses already-read configuration bytes (e.g. from inside a
// zip archive, via source.Package.Bytes) rather than a disk path.
func ParseConfig(data []byte) (FileConfig, error) {
	var cfg FileConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return FileConfig{}, fmt.Errorf("parsing configuration: %w", err)
	}
	return cfg, nil
}

// DefaultConfig returns an empty configuration -- used when no
// configuration.json is found anywhere (CLI override path, zip, or local
// project root), so every parameter falls through to hardcoded defaults.
func DefaultConfig() FileConfig {
	return FileConfig{}
}
