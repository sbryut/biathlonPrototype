package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"biathlonPrototype/internal/domain"
)

// Config structure for storing competition configuration
type Config struct {
	Laps        int     `json:"laps"`
	LapLen      float64 `json:"lapLen"`
	PenaltyLen  float64 `json:"penaltyLen"`
	FiringLines int     `json:"firingLines"`
	Start       string  `json:"start"`
	StartDelta  string  `json:"startDelta"`

	ParsedStart      time.Time     `json:"-"`
	ParsedStartDelta time.Duration `json:"-"`
}

// LoadConfiguration loads the configuration from a JSON file
func LoadConfiguration(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading configuration file %s: %v", filePath, err)
	}

	var cfg Config
	if err = json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("error parsing JSON config %s: %v", filePath, err)
	}

	cfg.ParsedStart, err = domain.ParseTimeFromString(fmt.Sprintf("[%s]", cfg.Start))
	if err != nil {
		return nil, fmt.Errorf("error parsing start time '%s': %v", cfg.Start, err)
	}

	cfg.ParsedStartDelta, err = domain.ParseDurationFromString(cfg.StartDelta)
	if err != nil {
		return nil, fmt.Errorf("error parsing start delta '%s': %v", cfg.StartDelta, err)
	}

	if cfg.Laps <= 0 || cfg.LapLen <= 0 || cfg.PenaltyLen <= 0 || cfg.FiringLines <= 0 {
		return nil, fmt.Errorf("incorrect values in configuration: Laps, LapLen should be > 0, PenaltyLen > 0, FiringLines > 0")
	}

	return &cfg, nil
}
