package config

import (
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
