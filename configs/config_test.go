package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify sensible defaults
	if cfg.LastNamespace != "default" {
		t.Errorf("DefaultConfig().LastNamespace = %q, want %q", cfg.LastNamespace, "default")
	}

	if cfg.LastResourceType != "deployments" {
		t.Errorf("DefaultConfig().LastResourceType = %q, want %q", cfg.LastResourceType, "deployments")
	}

	if cfg.LogLineLimit <= 0 {
		t.Errorf("DefaultConfig().LogLineLimit = %d, should be positive", cfg.LogLineLimit)
	}

	if cfg.RefreshInterval <= 0 {
		t.Errorf("DefaultConfig().RefreshInterval = %d, should be positive", cfg.RefreshInterval)
	}

	if cfg.FavoriteItems == nil {
		// nil is acceptable, but if not nil should be empty
	} else if len(cfg.FavoriteItems) != 0 {
		t.Errorf("DefaultConfig().FavoriteItems should be empty, got %v", cfg.FavoriteItems)
	}
}

func TestAddFavorite(t *testing.T) {
	cfg := DefaultConfig()

	// Add first favorite
	cfg.AddFavorite("deploy/nginx")
	if len(cfg.FavoriteItems) != 1 {
		t.Errorf("After AddFavorite, len(FavoriteItems) = %d, want 1", len(cfg.FavoriteItems))
	}
	if cfg.FavoriteItems[0] != "deploy/nginx" {
		t.Errorf("FavoriteItems[0] = %q, want %q", cfg.FavoriteItems[0], "deploy/nginx")
	}

	// Add second favorite
	cfg.AddFavorite("deploy/redis")
	if len(cfg.FavoriteItems) != 2 {
		t.Errorf("After second AddFavorite, len(FavoriteItems) = %d, want 2", len(cfg.FavoriteItems))
	}

	// Add duplicate - should not increase count
	cfg.AddFavorite("deploy/nginx")
	if len(cfg.FavoriteItems) != 2 {
		t.Errorf("After duplicate AddFavorite, len(FavoriteItems) = %d, want 2", len(cfg.FavoriteItems))
	}
}

func TestRemoveFavorite(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FavoriteItems = []string{"deploy/nginx", "deploy/redis", "deploy/postgres"}

	// Remove middle item
	cfg.RemoveFavorite("deploy/redis")
	if len(cfg.FavoriteItems) != 2 {
		t.Errorf("After RemoveFavorite, len(FavoriteItems) = %d, want 2", len(cfg.FavoriteItems))
	}

	// Verify correct items remain
	if cfg.FavoriteItems[0] != "deploy/nginx" || cfg.FavoriteItems[1] != "deploy/postgres" {
		t.Errorf("Wrong items after remove: %v", cfg.FavoriteItems)
	}

	// Remove non-existent item - should not panic or change list
	cfg.RemoveFavorite("deploy/nonexistent")
	if len(cfg.FavoriteItems) != 2 {
		t.Errorf("After removing non-existent, len(FavoriteItems) = %d, want 2", len(cfg.FavoriteItems))
	}

	// Remove first item
	cfg.RemoveFavorite("deploy/nginx")
	if len(cfg.FavoriteItems) != 1 {
		t.Errorf("After removing first, len(FavoriteItems) = %d, want 1", len(cfg.FavoriteItems))
	}

	// Remove last item
	cfg.RemoveFavorite("deploy/postgres")
	if len(cfg.FavoriteItems) != 0 {
		t.Errorf("After removing last, len(FavoriteItems) = %d, want 0", len(cfg.FavoriteItems))
	}
}

func TestIsFavorite(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FavoriteItems = []string{"deploy/nginx", "deploy/redis"}

	tests := []struct {
		item     string
		expected bool
	}{
		{"deploy/nginx", true},
		{"deploy/redis", true},
		{"deploy/postgres", false},
		{"", false},
		{"deploy/NGINX", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			result := cfg.IsFavorite(tt.item)
			if result != tt.expected {
				t.Errorf("IsFavorite(%q) = %v, want %v", tt.item, result, tt.expected)
			}
		})
	}
}

func TestSetters(t *testing.T) {
	cfg := DefaultConfig()

	cfg.SetLastNamespace("kube-system")
	if cfg.LastNamespace != "kube-system" {
		t.Errorf("After SetLastNamespace, LastNamespace = %q, want %q", cfg.LastNamespace, "kube-system")
	}

	cfg.SetLastContext("prod-cluster")
	if cfg.LastContext != "prod-cluster" {
		t.Errorf("After SetLastContext, LastContext = %q, want %q", cfg.LastContext, "prod-cluster")
	}

	cfg.SetLastResourceType("statefulsets")
	if cfg.LastResourceType != "statefulsets" {
		t.Errorf("After SetLastResourceType, LastResourceType = %q, want %q", cfg.LastResourceType, "statefulsets")
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := defaultConfigPath()
	if err != nil {
		t.Fatalf("defaultConfigPath() error = %v", err)
	}

	// Should end with .config/k1s/configs.json
	if filepath.Base(path) != "configs.json" {
		t.Errorf("defaultConfigPath() = %q, should end with configs.json", path)
	}

	dir := filepath.Dir(path)
	if filepath.Base(dir) != "k1s" {
		t.Errorf("defaultConfigPath() parent dir = %q, should be k1s", filepath.Base(dir))
	}
}

func TestDefaultConfigPathError(t *testing.T) {
	// Save original function
	originalFunc := userHomeDirFunc
	defer func() { userHomeDirFunc = originalFunc }()

	// Override to return error
	userHomeDirFunc = func() (string, error) {
		return "", os.ErrPermission
	}

	_, err := defaultConfigPath()
	if err == nil {
		t.Error("defaultConfigPath() should error when UserHomeDir fails")
	}
}

func TestConfigPath(t *testing.T) {
	// Test that configPath uses the function variable
	path, err := configPath()
	if err != nil {
		t.Fatalf("configPath() error = %v", err)
	}

	if path == "" {
		t.Error("configPath() should not return empty string")
	}
}

// setupTestConfig creates a temp directory and overrides configPathFunc for testing.
// It returns a cleanup function that must be called when done.
func setupTestConfig(t *testing.T) (string, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "k1s-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	configFile := filepath.Join(tmpDir, "configs.json")

	// Save original function
	originalFunc := configPathFunc

	// Override with test function
	configPathFunc = func() (string, error) {
		return configFile, nil
	}

	cleanup := func() {
		configPathFunc = originalFunc
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestLoadNonExistentFile(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Load should return default config when file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Should have default values
	if cfg.LastNamespace != "default" {
		t.Errorf("Load() LastNamespace = %q, want %q", cfg.LastNamespace, "default")
	}
}

func TestLoadExistingFile(t *testing.T) {
	tmpDir, cleanup := setupTestConfig(t)
	defer cleanup()

	configFile := filepath.Join(tmpDir, "configs.json")

	// Create a config file
	cfg := &Config{
		LastNamespace:    "test-namespace",
		LastContext:      "test-context",
		LastResourceType: "pods",
		LogLineLimit:     1000,
		RefreshInterval:  10,
		Theme:            "dark",
		FavoriteItems:    []string{"deploy/test1", "deploy/test2"},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load and verify
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedCfg.LastNamespace != "test-namespace" {
		t.Errorf("Loaded LastNamespace = %q, want %q", loadedCfg.LastNamespace, "test-namespace")
	}

	if loadedCfg.LastContext != "test-context" {
		t.Errorf("Loaded LastContext = %q, want %q", loadedCfg.LastContext, "test-context")
	}

	if len(loadedCfg.FavoriteItems) != 2 {
		t.Errorf("Loaded FavoriteItems length = %d, want 2", len(loadedCfg.FavoriteItems))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir, cleanup := setupTestConfig(t)
	defer cleanup()

	configFile := filepath.Join(tmpDir, "configs.json")

	// Write invalid JSON
	if err := os.WriteFile(configFile, []byte("{ invalid json }"), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Load should return default config on invalid JSON
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error on invalid JSON: %v", err)
	}

	// Should return default config
	if cfg.LastNamespace != "default" {
		t.Errorf("Load() on invalid JSON should return default, got LastNamespace = %q", cfg.LastNamespace)
	}
}

func TestLoadReadError(t *testing.T) {
	// Save original function
	originalFunc := configPathFunc
	defer func() { configPathFunc = originalFunc }()

	// Override to return a directory path (which can't be read as file)
	tmpDir, err := os.MkdirTemp("", "k1s-test-dir")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	configPathFunc = func() (string, error) {
		return tmpDir, nil // Return directory instead of file
	}

	// Load should return error when file can't be read (but exists)
	_, err = Load()
	if err == nil {
		t.Error("Load() should error when given a directory path")
	}
}

func TestSave(t *testing.T) {
	tmpDir, cleanup := setupTestConfig(t)
	defer cleanup()

	configFile := filepath.Join(tmpDir, "configs.json")

	cfg := DefaultConfig()
	cfg.LastNamespace = "save-test"

	// Save should create the file
	err := cfg.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Save() should create config file")
	}

	// Verify by loading
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}

	if loadedCfg.LastNamespace != "save-test" {
		t.Errorf("After Save and Load, LastNamespace = %q, want %q", loadedCfg.LastNamespace, "save-test")
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Save original function
	originalFunc := configPathFunc
	defer func() { configPathFunc = originalFunc }()

	tmpDir, err := os.MkdirTemp("", "k1s-test-save")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a nested path that doesn't exist yet
	configFile := filepath.Join(tmpDir, "nested", "dir", "configs.json")

	configPathFunc = func() (string, error) {
		return configFile, nil
	}

	cfg := DefaultConfig()
	err = cfg.Save()
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify directory was created
	dir := filepath.Dir(configFile)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Save() should create parent directories")
	}
}

func TestConfigPathError(t *testing.T) {
	// Save original function
	originalFunc := configPathFunc
	defer func() { configPathFunc = originalFunc }()

	// Override to return error
	configPathFunc = func() (string, error) {
		return "", os.ErrPermission
	}

	// Load should return default config on path error
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error on path error: %v", err)
	}

	if cfg.LastNamespace != "default" {
		t.Errorf("Load() on path error should return default, got LastNamespace = %q", cfg.LastNamespace)
	}

	// Save should return error
	err = cfg.Save()
	if err == nil {
		t.Error("Save() should error when configPath returns error")
	}
}

func TestSaveMkdirError(t *testing.T) {
	// Save original function
	originalFunc := configPathFunc
	defer func() { configPathFunc = originalFunc }()

	// Use a path where we can't create directories (e.g., under /proc on Linux or invalid path)
	configPathFunc = func() (string, error) {
		return "/dev/null/invalid/path/configs.json", nil
	}

	cfg := DefaultConfig()
	err := cfg.Save()
	if err == nil {
		t.Error("Save() should error when MkdirAll fails")
	}
}

func TestSaveMarshalError(t *testing.T) {
	_, cleanup := setupTestConfig(t)
	defer cleanup()

	// Save original marshal function
	originalMarshal := jsonMarshalFunc
	defer func() { jsonMarshalFunc = originalMarshal }()

	// Override to return error
	jsonMarshalFunc = func(v any, prefix, indent string) ([]byte, error) {
		return nil, os.ErrPermission
	}

	cfg := DefaultConfig()
	err := cfg.Save()
	if err == nil {
		t.Error("Save() should error when json marshal fails")
	}
}
