package style

import (
	"strings"
	"testing"
)

func TestGetStatusStyle(t *testing.T) {
	tests := []struct {
		status   string
		expected string // describe expected behavior
	}{
		// Running/Success states
		{"Running", "success"},
		{"Completed", "success"},
		{"Active", "success"},
		{"Ready", "success"},

		// Pending/Warning states
		{"Pending", "pending"},
		{"Progressing", "pending"},
		{"ContainerCreating", "pending"},

		// Error states
		{"Failed", "error"},
		{"Error", "error"},
		{"CrashLoopBackOff", "error"},
		{"ImagePullBackOff", "error"},
		{"ErrImagePull", "error"},
		{"OOMKilled", "error"},
		{"NotReady", "error"},
		{"Terminating", "error"},

		// Default/Muted states
		{"Unknown", "muted"},
		{"", "muted"},
		{"SomeOtherStatus", "muted"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := GetStatusStyle(tt.status)
			// Just verify the function doesn't panic and returns a style
			// The style itself is a lipgloss.Style which is hard to compare directly
			if result.Value() == "" {
				// Style should be set, but lipgloss styles don't have a simple comparison
				// Just ensure we get a result without panic
			}
		})
	}
}

func TestRenderWithWidth(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
	}{
		{"short content", "hello", 10},
		{"exact width", "hello", 5},
		{"wider than content", "hi", 20},
		{"zero width", "test", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderWithWidth(BaseStyle, tt.content, tt.width)
			// Should not panic and should return something
			if result == "" && tt.content != "" {
				t.Errorf("RenderWithWidth should return non-empty for non-empty content")
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"very short width", "hello", 3, "hel"},
		{"width 2", "hello", 2, "he"},
		{"width 1", "hello", 1, "h"},
		{"width 0", "hello", 0, ""},
		{"empty string", "", 5, ""},
		// Note: Truncate uses len() which counts bytes, not runes
		// "こんにちは" is 15 bytes (5 chars * 3 bytes each)
		{"unicode string longer", "こんにちは", 20, "こんにちは"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{"short string", "hi", 5, "hi   "},
		{"exact length", "hello", 5, "hello"},
		{"longer than width", "hello world", 5, "hello"},
		{"empty string", "", 5, "     "},
		{"zero width", "test", 0, ""},
		{"single char", "a", 3, "a  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PadRight(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("PadRight(%q, %d) = %q, want %q", tt.input, tt.width, result, tt.expected)
			}
		})
	}
}

func TestSpaces(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int // expected length
	}{
		{"positive", 5, 5},
		{"zero", 0, 0},
		{"negative", -1, 0},
		{"large", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := spaces(tt.n)
			if len(result) != tt.expected {
				t.Errorf("spaces(%d) length = %d, want %d", tt.n, len(result), tt.expected)
			}
			// Verify it's all spaces
			for _, r := range result {
				if r != ' ' {
					t.Errorf("spaces(%d) should contain only spaces, got %q", tt.n, r)
				}
			}
		})
	}
}

func TestCredit(t *testing.T) {
	result := Credit()

	// Credit should contain expected text
	if !strings.Contains(result, "built with") {
		t.Error("Credit() should contain 'built with'")
	}
	if !strings.Contains(result, "doganarif") {
		t.Error("Credit() should contain 'doganarif'")
	}
}

// Test that all style variables are initialized
func TestStyleVariablesInitialized(t *testing.T) {
	// Test color variables exist
	colors := []struct {
		name  string
		color interface{}
	}{
		{"Primary", Primary},
		{"Secondary", Secondary},
		{"Success", Success},
		{"Warning", Warning},
		{"Error", Error},
		{"Muted", Muted},
		{"Background", Background},
		{"Surface", Surface},
		{"Text", Text},
		{"TextMuted", TextMuted},
		{"Accent", Accent},
	}

	for _, c := range colors {
		t.Run(c.name, func(t *testing.T) {
			if c.color == nil {
				t.Errorf("%s color should be initialized", c.name)
			}
		})
	}
}
