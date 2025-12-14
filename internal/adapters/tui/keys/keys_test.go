package keys

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	// Helper function to check if a binding has keys
	hasKeys := func(b key.Binding) bool {
		return len(b.Keys()) > 0
	}

	// Helper function to check if a binding has help text
	hasHelp := func(b key.Binding) bool {
		h := b.Help()
		return h.Key != "" && h.Desc != ""
	}

	// Test navigation keys
	navigationBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Up", km.Up},
		{"Down", km.Down},
		{"Left", km.Left},
		{"Right", km.Right},
		{"Home", km.Home},
		{"End", km.End},
		{"PageUp", km.PageUp},
		{"PageDown", km.PageDown},
	}

	for _, tt := range navigationBindings {
		t.Run("navigation/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}

	// Test action keys
	actionBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Enter", km.Enter},
		{"Back", km.Back},
		{"Quit", km.Quit},
		{"Help", km.Help},
		{"Refresh", km.Refresh},
		{"Search", km.Search},
		{"Clear", km.Clear},
	}

	for _, tt := range actionBindings {
		t.Run("action/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}

	// Test panel navigation keys
	panelBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"NextPanel", km.NextPanel},
		{"PrevPanel", km.PrevPanel},
		{"Panel1", km.Panel1},
		{"Panel2", km.Panel2},
		{"Panel3", km.Panel3},
		{"Panel4", km.Panel4},
	}

	for _, tt := range panelBindings {
		t.Run("panel/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}

	// Test mode switches
	modeBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"Namespace", km.Namespace},
		{"ResourceType", km.ResourceType},
	}

	for _, tt := range modeBindings {
		t.Run("mode/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}

	// Test log actions
	logBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"ToggleFollow", km.ToggleFollow},
		{"JumpToError", km.JumpToError},
		{"ToggleWrap", km.ToggleWrap},
	}

	for _, tt := range logBindings {
		t.Run("log/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}

	// Test event/manifest actions
	miscBindings := []struct {
		name    string
		binding key.Binding
	}{
		{"ToggleAllEvents", km.ToggleAllEvents},
		{"ToggleFullView", km.ToggleFullView},
		{"CopyCommands", km.CopyCommands},
		{"PodActions", km.PodActions},
		{"Scale", km.Scale},
		{"Restart", km.Restart},
	}

	for _, tt := range miscBindings {
		t.Run("misc/"+tt.name, func(t *testing.T) {
			if !hasKeys(tt.binding) {
				t.Errorf("%s should have keys defined", tt.name)
			}
			if !hasHelp(tt.binding) {
				t.Errorf("%s should have help text", tt.name)
			}
		})
	}
}

// Test specific key assignments
func TestKeyAssignments(t *testing.T) {
	km := DefaultKeyMap()

	tests := []struct {
		name         string
		binding      key.Binding
		expectedKeys []string
	}{
		{"Up includes k", km.Up, []string{"up", "k"}},
		{"Down includes j", km.Down, []string{"down", "j"}},
		{"Left includes h", km.Left, []string{"left", "h"}},
		{"Right includes l", km.Right, []string{"right", "l"}},
		{"Quit includes q", km.Quit, []string{"q", "ctrl+c"}},
		{"Help is ?", km.Help, []string{"?"}},
		{"Search is /", km.Search, []string{"/"}},
		{"NextPanel is tab", km.NextPanel, []string{"tab"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := tt.binding.Keys()
			for _, expected := range tt.expectedKeys {
				found := false
				for _, k := range keys {
					if k == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected key %q not found in binding %s (has %v)", expected, tt.name, keys)
				}
			}
		})
	}
}

// Test that KeyMap struct can be instantiated
func TestKeyMapStruct(t *testing.T) {
	// Test that an empty KeyMap can be created
	var km KeyMap
	if km.Up.Keys() != nil && len(km.Up.Keys()) > 0 {
		t.Error("Empty KeyMap should have no keys by default")
	}
}
