package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	LastNamespace    string   `json:"last_namespace"`
	LastContext      string   `json:"last_context"`
	LastResourceType string   `json:"last_resource_type"`
	FavoriteItems    []string `json:"favorite_items"`
	LogLineLimit     int      `json:"log_line_limit"`
	RefreshInterval  int      `json:"refresh_interval_seconds"`
	Theme            string   `json:"theme"`
}

func DefaultConfig() *Config {
	return &Config{
		LastNamespace:    "default",
		LastResourceType: "deployments",
		LogLineLimit:     500,
		RefreshInterval:  5,
		Theme:            "default",
	}
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "k1s", "config.json"), nil
}

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

func (c *Config) SetLastNamespace(ns string) {
	c.LastNamespace = ns
}

func (c *Config) SetLastContext(ctx string) {
	c.LastContext = ctx
}

func (c *Config) SetLastResourceType(rt string) {
	c.LastResourceType = rt
}

func (c *Config) AddFavorite(item string) {
	for _, f := range c.FavoriteItems {
		if f == item {
			return
		}
	}
	c.FavoriteItems = append(c.FavoriteItems, item)
}

func (c *Config) RemoveFavorite(item string) {
	for i, f := range c.FavoriteItems {
		if f == item {
			c.FavoriteItems = append(c.FavoriteItems[:i], c.FavoriteItems[i+1:]...)
			return
		}
	}
}

func (c *Config) IsFavorite(item string) bool {
	for _, f := range c.FavoriteItems {
		if f == item {
			return true
		}
	}
	return false
}
