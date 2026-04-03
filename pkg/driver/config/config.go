package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/daominah/reminderd/pkg/logic"
)

// configJSON is the JSON-friendly representation of logic.Config.
// Durations are stored as human-readable strings (e.g. "60m", "10s").
type configJSON struct {
	ContinuousActiveLimit       string `json:"ContinuousActiveLimit"`
	IdleDurationToConsiderBreak string `json:"IdleDurationToConsiderBreak"`
	NotificationInitialBackoff  string `json:"NotificationInitialBackoff"`
	WebUIPort                   int    `json:"WebUIPort"`
}

func toJSON(cfg logic.Config) configJSON {
	return configJSON{
		ContinuousActiveLimit:       cfg.ContinuousActiveLimit.String(),
		IdleDurationToConsiderBreak: cfg.IdleDurationToConsiderBreak.String(),
		NotificationInitialBackoff:  cfg.NotificationInitialBackoff.String(),
		WebUIPort:                   cfg.WebUIPort,
	}
}

func fromJSON(j configJSON) (logic.Config, error) {
	var cfg logic.Config
	var err error

	cfg.ContinuousActiveLimit, err = time.ParseDuration(j.ContinuousActiveLimit)
	if err != nil {
		return cfg, fmt.Errorf("error parsing ContinuousActiveLimit: %w", err)
	}
	cfg.IdleDurationToConsiderBreak, err = time.ParseDuration(j.IdleDurationToConsiderBreak)
	if err != nil {
		return cfg, fmt.Errorf("error parsing IdleDurationToConsiderBreak: %w", err)
	}
	cfg.NotificationInitialBackoff, err = time.ParseDuration(j.NotificationInitialBackoff)
	if err != nil {
		return cfg, fmt.Errorf("error parsing NotificationInitialBackoff: %w", err)
	}
	cfg.WebUIPort = j.WebUIPort
	return cfg, nil
}

// FileConfigStore loads and saves config from a JSON file.
type FileConfigStore struct {
	Path        string
	lastModTime time.Time
}

func NewFileConfigStore(path string) *FileConfigStore {
	return &FileConfigStore{Path: path}
}

// Load reads config from the file, merges missing fields with defaults,
// and writes the full config back. If the file does not exist, it is
// created with default values.
func (s *FileConfigStore) Load() (logic.Config, error) {
	defaults := logic.DefaultConfig()
	defaultJSON := toJSON(defaults)

	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			if saveErr := s.Save(defaults); saveErr != nil {
				return defaults, fmt.Errorf("error saving default config: %w", saveErr)
			}
			s.updateModTime()
			return defaults, nil
		}
		return defaults, fmt.Errorf("error os.ReadFile: %w", err)
	}

	// Unmarshal into default-prefilled struct so missing fields keep defaults.
	j := defaultJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return defaults, fmt.Errorf("error json.Unmarshal config: %w", err)
	}

	cfg, err := fromJSON(j)
	if err != nil {
		return defaults, err
	}

	// Write back to fill any missing fields in the file.
	if saveErr := s.Save(cfg); saveErr != nil {
		return cfg, fmt.Errorf("error saving merged config: %w", saveErr)
	}
	s.updateModTime()
	return cfg, nil
}

// LoadIfChanged checks the file modification time and reloads only if changed.
// Returns the config, whether it changed, and any error.
func (s *FileConfigStore) LoadIfChanged() (logic.Config, bool, error) {
	info, err := os.Stat(s.Path)
	if err != nil {
		return logic.DefaultConfig(), false, fmt.Errorf("error os.Stat config: %w", err)
	}
	if !info.ModTime().After(s.lastModTime) {
		return logic.Config{}, false, nil
	}
	cfg, err := s.Load()
	if err != nil {
		return logic.DefaultConfig(), false, err
	}
	return cfg, true, nil
}

// Save writes the config to the file with indented JSON.
func (s *FileConfigStore) Save(cfg logic.Config) error {
	j := toJSON(cfg)
	data, err := json.MarshalIndent(j, "", "\t")
	if err != nil {
		return fmt.Errorf("error json.MarshalIndent: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(s.Path, data, 0644); err != nil {
		return fmt.Errorf("error os.WriteFile: %w", err)
	}
	return nil
}

func (s *FileConfigStore) updateModTime() {
	if info, err := os.Stat(s.Path); err == nil {
		s.lastModTime = info.ModTime()
	}
}
