// Package components provides reusable UI components for the k1s TUI.
//
// This package contains all visual components used in the application,
// including navigation panels, log viewers, metrics displays, and dialogs.
// Each component implements the bubbletea Model interface for consistent
// state management and rendering.
package components

import (
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
// It uses platform-specific commands: pbcopy (macOS), xclip/xsel (Linux), clip (Windows).
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		// Fallback - try xclip
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
