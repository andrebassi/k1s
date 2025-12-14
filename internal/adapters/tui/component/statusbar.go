package component

import (
	"strings"

	"github.com/andrebassi/k1s/internal/adapters/tui/style"
)

// Breadcrumb displays a navigation path showing the current location
// in the application hierarchy (e.g., namespace > pods > pod-name).
type Breadcrumb struct {
	items []string
	width int
}

// NewBreadcrumb creates a new empty breadcrumb component.
func NewBreadcrumb() Breadcrumb {
	return Breadcrumb{}
}

// SetItems updates the breadcrumb path items.
func (b *Breadcrumb) SetItems(items ...string) {
	b.items = items
}

// SetWidth sets the available width for rendering.
func (b *Breadcrumb) SetWidth(width int) {
	b.width = width
}

// View renders the breadcrumb as a string with separators.
// The last item is highlighted as the active item.
func (b Breadcrumb) View() string {
	if len(b.items) == 0 {
		return ""
	}

	var parts []string
	for i, item := range b.items {
		if i == len(b.items)-1 {
			parts = append(parts, style.BreadcrumbActiveStyle.Render(item))
		} else {
			parts = append(parts, style.BreadcrumbStyle.Render(item))
		}
	}

	sep := style.BreadcrumbStyle.Render(" > ")
	return strings.Join(parts, sep)
}
