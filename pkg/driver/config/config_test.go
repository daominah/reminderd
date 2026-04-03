package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/daominah/reminderd/pkg/logic"
)

func TestLoad_CreatesDefaultFileWhenMissing(t *testing.T) {
	// GIVEN a config path that does not exist
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileConfigStore(path)

	// WHEN Load is called
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// THEN the config file is created with default values
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
	expected := logic.DefaultConfig()
	if cfg.ContinuousActiveLimit != expected.ContinuousActiveLimit {
		t.Errorf("expected ContinuousActiveLimit %v, got %v",
			expected.ContinuousActiveLimit, cfg.ContinuousActiveLimit)
	}
	if cfg.WebUIPort != expected.WebUIPort {
		t.Errorf("expected WebUIPort %d, got %d", expected.WebUIPort, cfg.WebUIPort)
	}
}

func TestLoad_MergesMissingFieldsWithDefaults(t *testing.T) {
	// GIVEN a config file with only one field set
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	partial := map[string]any{"WebUIPort": 9999}
	data, _ := json.MarshalIndent(partial, "", "\t")
	os.WriteFile(path, data, 0644)
	store := NewFileConfigStore(path)

	// WHEN Load is called
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// THEN the specified field is preserved
	if cfg.WebUIPort != 9999 {
		t.Errorf("expected WebUIPort 9999, got %d", cfg.WebUIPort)
	}

	// AND missing fields get defaults
	expected := logic.DefaultConfig()
	if cfg.ContinuousActiveLimit != expected.ContinuousActiveLimit {
		t.Errorf("expected ContinuousActiveLimit %v, got %v",
			expected.ContinuousActiveLimit, cfg.ContinuousActiveLimit)
	}

	// AND the file is rewritten with all fields
	reread, _ := os.ReadFile(path)
	var full map[string]any
	json.Unmarshal(reread, &full)
	if _, ok := full["ContinuousActiveLimit"]; !ok {
		t.Error("expected ContinuousActiveLimit to be written back to file")
	}
}

func TestLoadIfChanged_ReturnsFalseWhenUnchanged(t *testing.T) {
	// GIVEN a config file that has been loaded once
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileConfigStore(path)
	store.Load()

	// WHEN LoadIfChanged is called without modifying the file
	_, changed, err := store.LoadIfChanged()
	if err != nil {
		t.Fatalf("LoadIfChanged error: %v", err)
	}

	// THEN changed is false
	if changed {
		t.Error("expected changed=false when file is unchanged")
	}
}

func TestLoadIfChanged_ReturnsTrueWhenModified(t *testing.T) {
	// GIVEN a config file that has been loaded once
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileConfigStore(path)
	store.Load()

	// WHEN the file is modified (change mod time)
	time.Sleep(10 * time.Millisecond)
	newCfg := map[string]any{"WebUIPort": 8888}
	data, _ := json.MarshalIndent(newCfg, "", "\t")
	os.WriteFile(path, data, 0644)

	// AND LoadIfChanged is called
	cfg, changed, err := store.LoadIfChanged()
	if err != nil {
		t.Fatalf("LoadIfChanged error: %v", err)
	}

	// THEN changed is true and the new value is returned
	if !changed {
		t.Error("expected changed=true after file modification")
	}
	if cfg.WebUIPort != 8888 {
		t.Errorf("expected WebUIPort 8888, got %d", cfg.WebUIPort)
	}
}

func TestSave_WritesConfigToFile(t *testing.T) {
	// GIVEN a config store
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	store := NewFileConfigStore(path)

	// WHEN a config is saved
	cfg := logic.DefaultConfig()
	cfg.WebUIPort = 12345
	err := store.Save(cfg)
	if err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// THEN the file contains the saved value
	data, _ := os.ReadFile(path)
	var loaded map[string]any
	json.Unmarshal(data, &loaded)
	port, ok := loaded["WebUIPort"]
	if !ok {
		t.Fatal("expected WebUIPort in saved file")
	}
	if int(port.(float64)) != 12345 {
		t.Errorf("expected WebUIPort 12345, got %v", port)
	}
}
