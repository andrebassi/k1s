package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/andrebassi/k1s/internal/ui/styles"
)

type StatusBar struct {
	context   string
	namespace string
	resource  string
	status    string
	width     int
}

func NewStatusBar() StatusBar {
	return StatusBar{}
}

func (s *StatusBar) SetContext(ctx string) {
	s.context = ctx
}

func (s *StatusBar) SetNamespace(ns string) {
	s.namespace = ns
}

func (s *StatusBar) SetResource(res string) {
	s.resource = res
}

func (s *StatusBar) SetStatus(status string) {
	s.status = status
}

func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

func (s StatusBar) View() string {
	left := s.renderLeft()
	right := s.renderRight()

	leftLen := lipgloss.Width(left)
	rightLen := lipgloss.Width(right)
	padding := s.width - leftLen - rightLen

	if padding < 0 {
		padding = 1
	}

	return styles.StatusBarStyle.Width(s.width).Render(
		left + strings.Repeat(" ", padding) + right,
	)
}

func (s StatusBar) renderLeft() string {
	var parts []string

	if s.context != "" {
		parts = append(parts, fmt.Sprintf("ctx:%s", styles.StatusBarKeyStyle.Render(s.context)))
	}

	if s.namespace != "" {
		parts = append(parts, fmt.Sprintf("ns:%s", styles.StatusBarKeyStyle.Render(s.namespace)))
	}

	if s.resource != "" {
		parts = append(parts, fmt.Sprintf("res:%s", styles.StatusBarKeyStyle.Render(s.resource)))
	}

	return strings.Join(parts, " | ")
}

func (s StatusBar) renderRight() string {
	if s.status != "" {
		return s.status
	}
	return "? help | q quit"
}

type Breadcrumb struct {
	items []string
	width int
}

func NewBreadcrumb() Breadcrumb {
	return Breadcrumb{}
}

func (b *Breadcrumb) SetItems(items ...string) {
	b.items = items
}

func (b *Breadcrumb) SetWidth(width int) {
	b.width = width
}

func (b Breadcrumb) View() string {
	if len(b.items) == 0 {
		return ""
	}

	var parts []string
	for i, item := range b.items {
		if i == len(b.items)-1 {
			parts = append(parts, styles.BreadcrumbActiveStyle.Render(item))
		} else {
			parts = append(parts, styles.BreadcrumbStyle.Render(item))
		}
	}

	sep := styles.BreadcrumbStyle.Render(" > ")
	return strings.Join(parts, sep)
}
