// Package tui provides the terminal user interface for k1s.
// This file contains the main View function that renders the application UI.
package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/tui/component"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// View renders the current application state to a string.
// This is the main rendering function called by bubbletea on each frame.
//
// The rendering order (back to front):
// 1. Error state - Shows error message if m.err is set
// 2. Loading state - Shows centered spinner while data loads
// 3. Main content - Navigator view or Dashboard view
// 4. Overlays (highest priority, rendered on top):
//   - Confirm dialog (delete confirmation)
//   - Workload action menu (scale, restart, delete options)
//   - Help panel (keyboard shortcuts)
//   - ConfigMap viewer (view/copy ConfigMap data)
//   - Secret viewer (view/copy Secret data)
//   - Docker Registry viewer (view/copy image pull secrets)
//
// The main content is wrapped in a bordered box with a status bar below.
func (m Model) View() string {
	// Error state takes priority
	if m.err != nil {
		return style.StatusError.Render("Error: " + m.err.Error())
	}

	// Loading state shows centered spinner
	if m.loading {
		loadingMsg := m.spinner.View() + " Loading..."
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, loadingMsg)
	}

	// Calculate dimensions for content box
	contentHeight := m.height - 2 // 2 for border top/bottom
	contentWidth := m.width - 2   // 2 for border left/right

	// Render main content based on current view
	var content string
	switch m.view {
	case ViewNavigator:
		content = m.renderNavigatorContent(contentWidth)
	case ViewDashboard:
		content = m.dashboard.View()
	}

	// Check for overlay components (rendered on top of main content)
	if overlay := m.renderOverlay(); overlay != "" {
		return overlay
	}

	// Render main content with border and status bar
	return m.renderMainContent(content, contentWidth, contentHeight)
}

// renderNavigatorContent renders the navigator view content.
// In namespace mode with nodes available, shows a split view:
// - Left panel: Namespace/resource selector
// - Right panel: Node selector
// Otherwise shows just the navigator.
func (m Model) renderNavigatorContent(contentWidth int) string {
	// In namespace mode, show nodes panel on the right
	if m.navigator.Mode() == component.ModeNamespace && len(m.nodes) > 0 {
		leftWidth := contentWidth / 2
		rightWidth := contentWidth - leftWidth - 3 // 3 for separator

		// Set panel active state for indicator
		m.navigator.SetPanelActive(!m.nodesPanelActive)

		leftContent := m.navigator.View()
		rightContent := m.renderNodesPanel(rightWidth, m.height-3)

		// Join panels side by side with separator
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(leftContent),
			lipgloss.NewStyle().Foreground(style.Surface).Render(" │ "),
			lipgloss.NewStyle().Width(rightWidth).Render(rightContent),
		)
	}

	// Default: show navigator only
	m.navigator.SetPanelActive(true)
	return m.navigator.View()
}

// renderOverlay checks for and renders any visible overlay components.
// Returns empty string if no overlay is visible.
// Overlays are rendered centered on screen with dimmed background.
func (m Model) renderOverlay() string {
	// Confirm dialog (highest priority)
	if m.confirmDialog.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.confirmDialog.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Workload action menu
	if m.workloadActionMenu.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.workloadActionMenu.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Help panel
	if m.help.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.help.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// ConfigMap viewer (full screen, top-left aligned)
	if m.configMapViewer.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Left, lipgloss.Top,
			m.configMapViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Secret viewer (full screen, top-left aligned)
	if m.secretViewer.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Left, lipgloss.Top,
			m.secretViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	// Docker Registry viewer (full screen, top-left aligned)
	if m.dockerRegistryViewer.IsVisible() {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Left, lipgloss.Top,
			m.dockerRegistryViewer.View(),
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(style.Background),
		)
	}

	return ""
}

// renderMainContent renders the main content area with border and status bar.
// The content is wrapped in a rounded border box with a status message below.
func (m Model) renderMainContent(content string, contentWidth, contentHeight int) string {
	// Reserve 1 line for status bar
	boxHeight := contentHeight - 1

	// Create bordered box for content
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.Surface).
		Width(contentWidth).
		Height(boxHeight)

	boxedContent := boxStyle.Render(content)

	// Status bar at bottom (same width as box including borders)
	statusStyle := lipgloss.NewStyle().
		Foreground(style.Warning).
		Bold(true).
		Padding(0, 2).
		Width(contentWidth + 2) // +2 for border
	statusBar := statusStyle.Render(m.statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusBar)
}
