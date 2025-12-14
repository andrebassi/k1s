// Package tui provides the terminal user interface for k1s.
// This file contains the node selection panel rendering logic.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// renderNodesPanel renders the node selection panel in the namespace view.
// Displays a list of cluster nodes with their status and pod count.
// Supports search/filter functionality and keyboard navigation.
//
// The panel includes:
// - Title with active indicator (● when focused)
// - Search bar when in search mode (/ prefix)
// - Filter indicator when filter is applied
// - Table header with columns: #, NODE, STATUS, PODS
// - Scrollable list of nodes with cursor highlighting
//
// Parameters:
//   - width: Available width for the panel
//   - height: Available height for the panel
//
// Returns the rendered panel as a string.
func (m Model) renderNodesPanel(width, height int) string {
	var b strings.Builder

	// Get filtered nodes based on search query
	nodes := m.filteredNodes()

	// Header styles
	iconStyle := lipgloss.NewStyle().Foreground(style.Primary).Bold(true)
	titleStyle := lipgloss.NewStyle().Foreground(style.Text).Bold(true)

	// Render title with active indicator
	if m.nodesPanelActive {
		b.WriteString(iconStyle.Render("● "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(titleStyle.Render("SELECT NODE"))
	b.WriteString("\n")

	// Show search bar or filter indicator on separate line
	if m.nodeSearching {
		// Active search mode with input cursor
		searchStyle := lipgloss.NewStyle().
			Foreground(style.Text).
			Background(style.Surface).
			Padding(0, 1)
		b.WriteString(searchStyle.Render("/ " + m.nodeSearchQuery + "_"))
		b.WriteString("\n\n")
	} else if m.nodeSearchQuery != "" {
		// Filter applied, show indicator with clear hint
		filterStyle := lipgloss.NewStyle().
			Foreground(style.Secondary).
			Bold(true)
		clearHint := style.HelpDescStyle.Render(" (c to clear)")
		b.WriteString(filterStyle.Render(fmt.Sprintf("Filter: %s", m.nodeSearchQuery)))
		b.WriteString(clearHint)
		b.WriteString("\n\n")
	} else {
		// Empty line to maintain consistent table position
		b.WriteString("\n\n")
	}

	// Table header
	header := fmt.Sprintf("  %-3s %-40s %-8s %s", "#", "NODE", "STATUS", "PODS")
	b.WriteString(style.TableHeaderStyle.Render(header))
	b.WriteString("\n")

	// Calculate visible window for scrolling
	// Account for header, search bar, and footer spacing
	maxVisible := height - 8
	if maxVisible < 5 {
		maxVisible = 5
	}

	// Calculate scroll window to keep cursor centered
	startIdx := 0
	endIdx := len(nodes)
	if len(nodes) > maxVisible {
		startIdx = m.nodeCursor - maxVisible/2
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx = startIdx + maxVisible
		if endIdx > len(nodes) {
			endIdx = len(nodes)
			startIdx = endIdx - maxVisible
			if startIdx < 0 {
				startIdx = 0
			}
		}
	}

	// Render visible nodes
	for i := startIdx; i < endIdx; i++ {
		node := nodes[i]
		idx := fmt.Sprintf("%d", i+1)

		// Style status based on Ready/NotReady
		statusStyle := style.StatusRunning
		if node.Status != "Ready" {
			statusStyle = style.StatusError
		}

		// Cursor indicator for selected row
		cursor := "  "
		nodeName := style.Truncate(node.Name, 40)
		// Pad status to 8 chars before styling to maintain alignment
		statusPadded := fmt.Sprintf("%-8s", node.Status)

		if m.nodesPanelActive && i == m.nodeCursor {
			// Highlighted row (selected)
			cursor = style.CursorStyle.Render("> ")
			rowStyle := lipgloss.NewStyle().Background(style.Surface)
			row := fmt.Sprintf("%s%-3s %-40s %s %d",
				cursor,
				idx,
				nodeName,
				statusStyle.Render(statusPadded),
				node.PodCount,
			)
			b.WriteString(rowStyle.Render(row))
		} else {
			// Normal row
			b.WriteString(fmt.Sprintf("%s%-3s %-40s %s %d",
				cursor,
				idx,
				nodeName,
				statusStyle.Render(statusPadded),
				node.PodCount,
			))
		}
		b.WriteString("\n")
	}

	return b.String()
}
