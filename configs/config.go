// Package config provides configuration management for k1s.
//
// Configuration is stored as JSON in ~/.config/k1s/configs.json and includes
// user preferences such as last used namespace, context, theme, and favorites.
// The package provides automatic persistence and default values for all settings.
package configs

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the user preferences and application state that persists
// between sessions. All fields are optional and have sensible defaults.
type Config struct {
	// LastNamespace is the most recently used Kubernetes namespace.
	LastNamespace string `json:"last_namespace"`

	// LastContext is the most recently used Kubernetes context.
	LastContext string `json:"last_context"`

	// LastResourceType is the most recently viewed resource type (e.g., "deployments", "pods").
	LastResourceType string `json:"last_resource_type"`

	// FavoriteItems contains user-bookmarked resources for quick access.
	FavoriteItems []string `json:"favorite_items"`

	// LogLineLimit specifies the maximum number of log lines to fetch per container.
	LogLineLimit int `json:"log_line_limit"`

	// RefreshInterval specifies the data refresh interval in seconds.
	RefreshInterval int `json:"refresh_interval_seconds"`

	// Theme specifies the color theme name (reserved for future use).
	Theme string `json:"theme"`
}

// DefaultConfig returns a new Config with sensible default values.
// These defaults are used when no configuration file exists or when
// specific values are not set.
func DefaultConfig() *Config {
	return &Config{
		LastNamespace:    "default",
		LastResourceType: "deployments",
		LogLineLimit:     500,
		RefreshInterval:  5,
		Theme:            "default",
	}
}

// configPath returns the path to the configuration file.
// The path follows XDG conventions: ~/.config/k1s/configs.json
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "k1s", "configs.json"), nil
}

// Load reads the configuration from disk and returns it.
// If the configuration file doesn't exist or is invalid, it returns
// a default configuration without error. This ensures the application
// always starts with a valid configuration.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), nil
	}
	return cfg, nil
}

// Save persists the configuration to disk.
// It creates the configuration directory if it doesn't exist.
// The file is written with pretty-printed JSON for human readability.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SetLastNamespace updates the last used namespace.
func (c *Config) SetLastNamespace(ns string) {
	c.LastNamespace = ns
}

// SetLastContext updates the last used Kubernetes context.
func (c *Config) SetLastContext(ctx string) {
	c.LastContext = ctx
}

// SetLastResourceType updates the last viewed resource type.
func (c *Config) SetLastResourceType(rt string) {
	c.LastResourceType = rt
}

// AddFavorite adds an item to the favorites list if it's not already present.
// Duplicates are silently ignored to maintain a unique set of favorites.
func (c *Config) AddFavorite(item string) {
	for _, f := range c.FavoriteItems {
		if f == item {
			return
		}
	}
	c.FavoriteItems = append(c.FavoriteItems, item)
}

// RemoveFavorite removes an item from the favorites list.
// If the item is not in the list, no action is taken.
func (c *Config) RemoveFavorite(item string) {
	for i, f := range c.FavoriteItems {
		if f == item {
			c.FavoriteItems = append(c.FavoriteItems[:i], c.FavoriteItems[i+1:]...)
			return
		}
	}
}

// IsFavorite checks whether an item is in the favorites list.
func (c *Config) IsFavorite(item string) bool {
	for _, f := range c.FavoriteItems {
		if f == item {
			return true
		}
	}
	return false
}
