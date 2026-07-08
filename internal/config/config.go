package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type rawConfig struct {
	ToleranceCents            *int64         `json:"tolerance_cents"`
	MatchingWindowDays        *int           `json:"matching_window_days"`
	OutputFormat              *string        `json:"output_format"`
	PartitionBy               *string        `json:"partition_by"`
	IncludePendingSettlements *bool          `json:"include_pending_settlements"`
	QueryLimit                *int           `json:"query_limit"`
	Filters                   map[string]any `json:"filters"`
}

// This function will load the config file , however a missing or empty file will not initialize and error
func LoadConfig(path string) (rawConfig, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return rawConfig{}, nil
	}
	if err != nil {
		return rawConfig{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg rawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return rawConfig{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}
