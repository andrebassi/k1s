package component

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/adapters/repository"
)

// ============================================
// ConfirmDialog Tests
// ============================================

func TestNewConfirmDialog(t *testing.T) {
	cd := NewConfirmDialog()
	if cd.selected != false {
		t.Error("NewConfirmDialog should default selected to false (No)")
	}
	if cd.visible {
		t.Error("NewConfirmDialog should not be visible by default")
	}
}

func TestConfirmDialog_Init(t *testing.T) {
	cd := NewConfirmDialog()
	cmd := cd.Init()
	if cmd != nil {
		t.Error("ConfirmDialog.Init() should return nil")
	}
}

func TestConfirmDialog_ShowHide(t *testing.T) {
	cd := NewConfirmDialog()

	if cd.IsVisible() {
		t.Error("Dialog should not be visible initially")
	}

	cd.Show("Test Title", "Test Message", "test-action", "test-data")

	if !cd.IsVisible() {
		t.Error("Dialog should be visible after Show()")
	}
	if cd.title != "Test Title" {
		t.Errorf("title = %q, want %q", cd.title, "Test Title")
	}
	if cd.message != "Test Message" {
		t.Errorf("message = %q, want %q", cd.message, "Test Message")
	}
	if cd.action != "test-action" {
		t.Errorf("action = %q, want %q", cd.action, "test-action")
	}

	cd.Hide()
	if cd.IsVisible() {
		t.Error("Dialog should not be visible after Hide()")
	}
}

func TestConfirmDialog_View_Hidden(t *testing.T) {
	cd := NewConfirmDialog()
	view := cd.View()
	if view != "" {
		t.Error("Hidden dialog View() should return empty string")
	}
}

func TestConfirmDialog_View_Visible(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Confirm Delete", "Are you sure?", "delete", nil)
	view := cd.View()

	if view == "" {
		t.Error("Visible dialog View() should not return empty string")
	}
	if !strings.Contains(view, "Confirm Delete") {
		t.Error("View should contain the title")
	}
	if !strings.Contains(view, "Yes") {
		t.Error("View should contain Yes button")
	}
	if !strings.Contains(view, "No") {
		t.Error("View should contain No button")
	}
}

func TestConfirmDialog_Update_NotVisible(t *testing.T) {
	cd := NewConfirmDialog()
	_, cmd := cd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden dialog should return nil cmd")
	}
}

func TestConfirmDialog_Update_EscKey(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Test", "Test", "action", nil)

	cd, cmd := cd.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cd.visible {
		t.Error("Esc should hide the dialog")
	}
	if cmd == nil {
		t.Error("Esc should return a command")
	}

	// Execute the command to get the result
	msg := cmd()
	result, ok := msg.(ConfirmResult)
	if !ok {
		t.Error("Command should return ConfirmResult")
	}
	if result.Confirmed {
		t.Error("Esc should not confirm")
	}
}

func TestConfirmDialog_Update_YKey(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Test", "Test", "action", "data")

	cd, cmd := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if cd.visible {
		t.Error("Y should hide the dialog")
	}
	if cmd == nil {
		t.Error("Y should return a command")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResult)
	if !ok {
		t.Error("Command should return ConfirmResult")
	}
	if !result.Confirmed {
		t.Error("Y should confirm")
	}
	if result.Action != "action" {
		t.Errorf("Action = %q, want %q", result.Action, "action")
	}
}

func TestConfirmDialog_Update_NKey(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Test", "Test", "action", nil)

	cd, cmd := cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if cd.visible {
		t.Error("N should hide the dialog")
	}

	msg := cmd()
	result := msg.(ConfirmResult)
	if result.Confirmed {
		t.Error("N should not confirm")
	}
}

func TestConfirmDialog_Update_Navigation(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Test", "Test", "action", nil)

	// Default is No (selected = false)
	if cd.selected {
		t.Error("Default selection should be No")
	}

	// Left arrow selects Yes
	cd, _ = cd.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if !cd.selected {
		t.Error("Left should select Yes")
	}

	// Right arrow selects No
	cd, _ = cd.Update(tea.KeyMsg{Type: tea.KeyRight})
	if cd.selected {
		t.Error("Right should select No")
	}

	// Tab toggles
	cd, _ = cd.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !cd.selected {
		t.Error("Tab should toggle selection")
	}

	// H key selects Yes
	cd.selected = false
	cd, _ = cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if !cd.selected {
		t.Error("H should select Yes")
	}

	// L key selects No
	cd, _ = cd.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if cd.selected {
		t.Error("L should select No")
	}
}

func TestConfirmDialog_Update_Enter(t *testing.T) {
	cd := NewConfirmDialog()
	cd.Show("Test", "Test", "action", nil)

	// With No selected (default)
	cd, cmd := cd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	result := msg.(ConfirmResult)
	if result.Confirmed {
		t.Error("Enter with No selected should not confirm")
	}

	// With Yes selected
	cd.Show("Test", "Test", "action", nil)
	cd.selected = true
	cd, cmd = cd.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	result = msg.(ConfirmResult)
	if !result.Confirmed {
		t.Error("Enter with Yes selected should confirm")
	}
}

// ============================================
// HelpPanel Tests
// ============================================

func TestNewHelpPanel(t *testing.T) {
	hp := NewHelpPanel()
	if hp.visible {
		t.Error("NewHelpPanel should not be visible by default")
	}
	if len(hp.entries) == 0 {
		t.Error("NewHelpPanel should have default entries")
	}
}

func TestHelpPanel_Toggle(t *testing.T) {
	hp := NewHelpPanel()

	hp.Toggle()
	if !hp.visible {
		t.Error("Toggle should make panel visible")
	}

	hp.Toggle()
	if hp.visible {
		t.Error("Toggle should make panel hidden")
	}
}

func TestHelpPanel_ShowHide(t *testing.T) {
	hp := NewHelpPanel()

	hp.Show()
	if !hp.IsVisible() {
		t.Error("Show should make panel visible")
	}

	hp.Hide()
	if hp.IsVisible() {
		t.Error("Hide should make panel hidden")
	}
}

func TestHelpPanel_SetSize(t *testing.T) {
	hp := NewHelpPanel()
	hp.SetSize(100, 50)
	if hp.width != 100 {
		t.Errorf("width = %d, want 100", hp.width)
	}
	if hp.height != 50 {
		t.Errorf("height = %d, want 50", hp.height)
	}
}

func TestHelpPanel_View_Hidden(t *testing.T) {
	hp := NewHelpPanel()
	view := hp.View()
	if view != "" {
		t.Error("Hidden HelpPanel View() should return empty string")
	}
}

func TestHelpPanel_View_Visible(t *testing.T) {
	hp := NewHelpPanel()
	hp.Show()
	view := hp.View()

	if view == "" {
		t.Error("Visible HelpPanel View() should not return empty string")
	}
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("View should contain title")
	}
}

func TestHelpPanel_ShortHelp(t *testing.T) {
	hp := NewHelpPanel()
	help := hp.ShortHelp()

	if help == "" {
		t.Error("ShortHelp should not return empty string")
	}
	if !strings.Contains(help, "nav") {
		t.Error("ShortHelp should contain 'nav'")
	}
	if !strings.Contains(help, "help") {
		t.Error("ShortHelp should contain 'help'")
	}
}

func TestDefaultHelpEntries(t *testing.T) {
	entries := defaultHelpEntries()
	if len(entries) == 0 {
		t.Error("defaultHelpEntries should return non-empty slice")
	}

	// Check that each group has entries
	for i, group := range entries {
		if len(group) == 0 {
			t.Errorf("Group %d should have entries", i)
		}
	}
}

// ============================================
// Breadcrumb Tests
// ============================================

func TestNewBreadcrumb(t *testing.T) {
	b := NewBreadcrumb()
	if len(b.items) != 0 {
		t.Error("NewBreadcrumb should have empty items")
	}
}

func TestBreadcrumb_SetItems(t *testing.T) {
	b := NewBreadcrumb()
	b.SetItems("namespace", "pods", "nginx-abc123")

	if len(b.items) != 3 {
		t.Errorf("items length = %d, want 3", len(b.items))
	}
}

func TestBreadcrumb_SetWidth(t *testing.T) {
	b := NewBreadcrumb()
	b.SetWidth(100)
	if b.width != 100 {
		t.Errorf("width = %d, want 100", b.width)
	}
}

func TestBreadcrumb_View_Empty(t *testing.T) {
	b := NewBreadcrumb()
	view := b.View()
	if view != "" {
		t.Error("Empty breadcrumb View() should return empty string")
	}
}

func TestBreadcrumb_View_WithItems(t *testing.T) {
	b := NewBreadcrumb()
	b.SetItems("default", "deployments", "nginx")
	view := b.View()

	if view == "" {
		t.Error("Breadcrumb with items should not return empty view")
	}
	if !strings.Contains(view, "default") {
		t.Error("View should contain first item")
	}
	if !strings.Contains(view, "nginx") {
		t.Error("View should contain last item")
	}
	if !strings.Contains(view, ">") {
		t.Error("View should contain separator")
	}
}

func TestBreadcrumb_View_SingleItem(t *testing.T) {
	b := NewBreadcrumb()
	b.SetItems("single")
	view := b.View()

	if view == "" {
		t.Error("Single item breadcrumb should return non-empty view")
	}
	// Single item may or may not contain styled separator
	_ = strings.Contains(view, ">")
}

// ============================================
// HelpEntry struct test
// ============================================

func TestHelpEntry(t *testing.T) {
	entry := HelpEntry{Key: "enter", Desc: "select"}
	if entry.Key != "enter" {
		t.Errorf("Key = %q, want %q", entry.Key, "enter")
	}
	if entry.Desc != "select" {
		t.Errorf("Desc = %q, want %q", entry.Desc, "select")
	}
}

// ============================================
// ConfirmResult struct test
// ============================================

func TestConfirmResult(t *testing.T) {
	result := ConfirmResult{
		Confirmed: true,
		Action:    "delete",
		Data:      "pod-name",
	}
	if !result.Confirmed {
		t.Error("Confirmed should be true")
	}
	if result.Action != "delete" {
		t.Errorf("Action = %q, want %q", result.Action, "delete")
	}
	if result.Data != "pod-name" {
		t.Errorf("Data = %v, want %q", result.Data, "pod-name")
	}
}

// ============================================
// EventsPanel Tests
// ============================================

func TestNewEventsPanel(t *testing.T) {
	ep := NewEventsPanel()
	if ep.ready {
		t.Error("NewEventsPanel should not be ready initially")
	}
	if ep.showAll {
		t.Error("NewEventsPanel should show warnings only by default")
	}
	if ep.searching {
		t.Error("NewEventsPanel should not be searching initially")
	}
}

func TestEventsPanel_Init(t *testing.T) {
	ep := NewEventsPanel()
	cmd := ep.Init()
	if cmd != nil {
		t.Error("EventsPanel.Init() should return nil")
	}
}

func TestEventsPanel_SetSize(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)
	if ep.width != 100 {
		t.Errorf("width = %d, want 100", ep.width)
	}
	if !ep.ready {
		t.Error("SetSize should mark panel as ready")
	}
}

func TestEventsPanel_SetEvents(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	events := []repository.EventInfo{
		{Type: "Warning", Reason: "BackOff", Message: "Back-off restarting"},
		{Type: "Normal", Reason: "Pulled", Message: "Successfully pulled image"},
	}
	ep.SetEvents(events)

	if ep.EventCount() != 2 {
		t.Errorf("EventCount() = %d, want 2", ep.EventCount())
	}
	if ep.WarningCount() != 1 {
		t.Errorf("WarningCount() = %d, want 1", ep.WarningCount())
	}
}

func TestEventsPanel_View_NotReady(t *testing.T) {
	ep := NewEventsPanel()
	view := ep.View()
	if !strings.Contains(view, "Loading") {
		t.Error("Not ready EventsPanel should show loading message")
	}
}

func TestEventsPanel_View_Ready(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)
	view := ep.View()
	if !strings.Contains(view, "Events") {
		t.Error("Ready EventsPanel should show Events title")
	}
}

func TestEventsPanel_ToggleShowAll(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	// Default is warnings only
	if ep.showAll {
		t.Error("Default should show warnings only")
	}

	// Toggle with 'w' key
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if !ep.showAll {
		t.Error("After 'w' key should show all events")
	}

	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	if ep.showAll {
		t.Error("After second 'w' key should show warnings only")
	}
}

func TestEventsPanel_Navigation(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	events := []repository.EventInfo{
		{Type: "Warning", Reason: "Event1", Message: "Message 1"},
		{Type: "Warning", Reason: "Event2", Message: "Message 2"},
		{Type: "Warning", Reason: "Event3", Message: "Message 3"},
	}
	ep.SetEvents(events)

	// Initial cursor at 0
	if ep.cursor != 0 {
		t.Errorf("Initial cursor = %d, want 0", ep.cursor)
	}

	// Move down with 'j'
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if ep.cursor != 1 {
		t.Errorf("After j, cursor = %d, want 1", ep.cursor)
	}

	// Move down with 'down' arrow
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyDown})
	if ep.cursor != 2 {
		t.Errorf("After down, cursor = %d, want 2", ep.cursor)
	}

	// Move up with 'k'
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if ep.cursor != 1 {
		t.Errorf("After k, cursor = %d, want 1", ep.cursor)
	}

	// Move up with 'up' arrow
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyUp})
	if ep.cursor != 0 {
		t.Errorf("After up, cursor = %d, want 0", ep.cursor)
	}
}

func TestEventsPanel_Search(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	// Start search with '/'
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !ep.IsSearching() {
		t.Error("After '/' should be in search mode")
	}

	// Exit search with Enter
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if ep.IsSearching() {
		t.Error("After Enter should exit search mode")
	}
}

func TestEventsPanel_ClearSearch(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)
	ep.filter = "test"
	ep.searching = true

	ep.ClearSearch()

	if ep.filter != "" {
		t.Error("ClearSearch should clear filter")
	}
	if ep.searching {
		t.Error("ClearSearch should stop searching")
	}
}

func TestEventsPanel_SelectedEvent(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	// No events - should return nil
	event := ep.SelectedEvent()
	if event != nil {
		t.Error("SelectedEvent should return nil when no events")
	}

	// With events
	events := []repository.EventInfo{
		{Type: "Warning", Reason: "Test", Message: "Test message"},
	}
	ep.SetEvents(events)

	event = ep.SelectedEvent()
	if event == nil {
		t.Fatal("SelectedEvent should return event when events exist")
	}
	if event.Reason != "Test" {
		t.Errorf("SelectedEvent.Reason = %q, want %q", event.Reason, "Test")
	}
}

func TestEventsPanel_GetDisplayedEvents_FilterByType(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)

	events := []repository.EventInfo{
		{Type: "Warning", Reason: "Warning1", Message: "Warning message"},
		{Type: "Normal", Reason: "Normal1", Message: "Normal message"},
		{Type: "Warning", Reason: "Warning2", Message: "Another warning"},
	}
	ep.SetEvents(events)

	// Warnings only (default)
	displayed := ep.getDisplayedEvents()
	if len(displayed) != 2 {
		t.Errorf("Warnings only should show 2 events, got %d", len(displayed))
	}

	// Show all
	ep.showAll = true
	displayed = ep.getDisplayedEvents()
	if len(displayed) != 3 {
		t.Errorf("Show all should show 3 events, got %d", len(displayed))
	}
}

func TestEventsPanel_GetDisplayedEvents_SearchFilter(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetSize(100, 50)
	ep.showAll = true

	events := []repository.EventInfo{
		{Type: "Warning", Reason: "BackOff", Message: "Back-off restarting"},
		{Type: "Normal", Reason: "Pulled", Message: "Successfully pulled image"},
		{Type: "Warning", Reason: "Failed", Message: "Failed to pull image"},
	}
	ep.SetEvents(events)
	ep.filter = "pull"

	displayed := ep.getDisplayedEvents()
	if len(displayed) != 2 {
		t.Errorf("Filter 'pull' should show 2 events, got %d", len(displayed))
	}
}

// ============================================
// ActionMenu Tests
// ============================================

func TestNewActionMenu(t *testing.T) {
	am := NewActionMenu()
	if am.visible {
		t.Error("NewActionMenu should not be visible by default")
	}
	if am.selected != 0 {
		t.Error("NewActionMenu should have selected = 0")
	}
}

func TestActionMenu_Init(t *testing.T) {
	am := NewActionMenu()
	cmd := am.Init()
	if cmd != nil {
		t.Error("ActionMenu.Init() should return nil")
	}
}

func TestActionMenu_ShowHide(t *testing.T) {
	am := NewActionMenu()

	items := []MenuItem{
		{Label: "Item 1", Value: "value1", Shortcut: "1"},
		{Label: "Item 2", Value: "value2", Shortcut: "2"},
	}
	am.Show("Test Menu", items)

	if !am.IsVisible() {
		t.Error("ActionMenu should be visible after Show()")
	}
	if am.title != "Test Menu" {
		t.Errorf("title = %q, want %q", am.title, "Test Menu")
	}
	if len(am.items) != 2 {
		t.Errorf("items count = %d, want 2", len(am.items))
	}

	am.Hide()
	if am.IsVisible() {
		t.Error("ActionMenu should not be visible after Hide()")
	}
}

func TestActionMenu_View_Hidden(t *testing.T) {
	am := NewActionMenu()
	view := am.View()
	if view != "" {
		t.Error("Hidden ActionMenu View() should return empty string")
	}
}

func TestActionMenu_View_NoItems(t *testing.T) {
	am := NewActionMenu()
	am.visible = true
	view := am.View()
	if view != "" {
		t.Error("ActionMenu with no items should return empty view")
	}
}

func TestActionMenu_View_Visible(t *testing.T) {
	am := NewActionMenu()
	items := []MenuItem{
		{Label: "Copy Value", Value: "test-value", Shortcut: "1"},
	}
	am.Show("Actions", items)

	view := am.View()
	if view == "" {
		t.Error("Visible ActionMenu should return non-empty view")
	}
	if !strings.Contains(view, "Actions") {
		t.Error("View should contain title")
	}
}

func TestActionMenu_Update_NotVisible(t *testing.T) {
	am := NewActionMenu()
	_, cmd := am.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden menu should return nil cmd")
	}
}

func TestActionMenu_Update_EscKey(t *testing.T) {
	am := NewActionMenu()
	items := []MenuItem{{Label: "Test", Value: "test"}}
	am.Show("Test", items)

	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if am.visible {
		t.Error("Esc should hide the menu")
	}
}

func TestActionMenu_Update_Navigation(t *testing.T) {
	am := NewActionMenu()
	items := []MenuItem{
		{Label: "Item 1", Value: "1"},
		{Label: "Item 2", Value: "2"},
		{Label: "Item 3", Value: "3"},
	}
	am.Show("Test", items)

	// Initial selection is 0
	if am.selected != 0 {
		t.Errorf("Initial selection = %d, want 0", am.selected)
	}

	// Move down with 'j'
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if am.selected != 1 {
		t.Errorf("After j, selected = %d, want 1", am.selected)
	}

	// Move down with 'down'
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyDown})
	if am.selected != 2 {
		t.Errorf("After down, selected = %d, want 2", am.selected)
	}

	// Try to go past end
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyDown})
	if am.selected != 2 {
		t.Errorf("Should not go past end, selected = %d, want 2", am.selected)
	}

	// Move up with 'k'
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if am.selected != 1 {
		t.Errorf("After k, selected = %d, want 1", am.selected)
	}

	// Move up with 'up'
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyUp})
	if am.selected != 0 {
		t.Errorf("After up, selected = %d, want 0", am.selected)
	}

	// Try to go past start
	am, _ = am.Update(tea.KeyMsg{Type: tea.KeyUp})
	if am.selected != 0 {
		t.Errorf("Should not go past start, selected = %d, want 0", am.selected)
	}
}

func TestMenuItem(t *testing.T) {
	item := MenuItem{
		Label:    "Copy kubectl command",
		Value:    "kubectl get pods",
		Shortcut: "1",
	}
	if item.Label != "Copy kubectl command" {
		t.Errorf("Label = %q, want %q", item.Label, "Copy kubectl command")
	}
	if item.Value != "kubectl get pods" {
		t.Errorf("Value = %q, want %q", item.Value, "kubectl get pods")
	}
	if item.Shortcut != "1" {
		t.Errorf("Shortcut = %q, want %q", item.Shortcut, "1")
	}
}

func TestActionMenuResult(t *testing.T) {
	result := ActionMenuResult{
		Item:   MenuItem{Label: "Test", Value: "value"},
		Copied: true,
		Err:    nil,
	}
	if !result.Copied {
		t.Error("Copied should be true")
	}
	if result.Err != nil {
		t.Error("Err should be nil")
	}
	if result.Item.Label != "Test" {
		t.Errorf("Item.Label = %q, want %q", result.Item.Label, "Test")
	}
}

// ============================================
// MetricsPanel Tests
// ============================================

func TestNewMetricsPanel(t *testing.T) {
	mp := NewMetricsPanel()
	if mp.ready {
		t.Error("NewMetricsPanel should not be ready initially")
	}
}

func TestMetricsPanel_Init(t *testing.T) {
	mp := NewMetricsPanel()
	cmd := mp.Init()
	if cmd != nil {
		t.Error("MetricsPanel.Init() should return nil")
	}
}

func TestMetricsPanel_SetSize(t *testing.T) {
	mp := NewMetricsPanel()
	mp.SetSize(100, 50)
	if mp.width != 100 {
		t.Errorf("width = %d, want 100", mp.width)
	}
	if !mp.ready {
		t.Error("SetSize should mark panel as ready")
	}
}

func TestMetricsPanel_View_NotReady(t *testing.T) {
	mp := NewMetricsPanel()
	view := mp.View()
	if !strings.Contains(view, "Loading") {
		t.Error("Not ready MetricsPanel should show loading message")
	}
}

func TestMetricsPanel_View_Ready(t *testing.T) {
	mp := NewMetricsPanel()
	mp.SetSize(100, 50)
	view := mp.View()
	if view == "" {
		t.Error("Ready MetricsPanel should return non-empty view")
	}
}

func TestMetricsPanel_SetMetrics(t *testing.T) {
	mp := NewMetricsPanel()
	mp.SetSize(100, 50)

	metrics := &repository.PodMetrics{
		Name:      "test-pod",
		Namespace: "default",
		Containers: []repository.ContainerMetrics{
			{
				Name:        "app",
				CPUUsage:    "100m",
				MemoryUsage: "256Mi",
				CPUPercent:  25.0,
				MemPercent:  50.0,
			},
		},
	}
	mp.SetMetrics(metrics)

	if mp.metrics == nil {
		t.Error("SetMetrics should set the metrics")
	}
	if mp.metrics.Name != "test-pod" {
		t.Errorf("metrics.Name = %q, want %q", mp.metrics.Name, "test-pod")
	}
}

func TestMetricsPanel_SetNode(t *testing.T) {
	mp := NewMetricsPanel()
	mp.SetSize(100, 50)

	node := &repository.NodeInfo{
		Name:     "worker-1",
		Status:   "Ready",
		Version:  "v1.28.0",
		CPU:      "4",
		Memory:   "8Gi",
	}
	mp.SetNode(node)

	if mp.node == nil {
		t.Error("SetNode should set the node")
	}
	if mp.node.Name != "worker-1" {
		t.Errorf("node.Name = %q, want %q", mp.node.Name, "worker-1")
	}
}

func TestMetricsPanel_Update(t *testing.T) {
	mp := NewMetricsPanel()
	mp.SetSize(100, 50)

	// Test scroll down
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Verify no panic occurs

	// Test scroll up
	mp, _ = mp.Update(tea.KeyMsg{Type: tea.KeyUp})
	// Verify no panic occurs
}

// ============================================
// LogsPanel Tests
// ============================================

func TestNewLogsPanel(t *testing.T) {
	lp := NewLogsPanel()
	if lp.ready {
		t.Error("NewLogsPanel should not be ready initially")
	}
	if lp.searching {
		t.Error("NewLogsPanel should not be searching initially")
	}
}

func TestLogsPanel_Init(t *testing.T) {
	lp := NewLogsPanel()
	cmd := lp.Init()
	if cmd != nil {
		t.Error("LogsPanel.Init() should return nil")
	}
}

func TestLogsPanel_SetSize(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)
	if lp.width != 100 {
		t.Errorf("width = %d, want 100", lp.width)
	}
	if !lp.ready {
		t.Error("SetSize should mark panel as ready")
	}
}

func TestLogsPanel_View_NotReady(t *testing.T) {
	lp := NewLogsPanel()
	view := lp.View()
	if !strings.Contains(view, "Loading") {
		t.Error("Not ready LogsPanel should show loading message")
	}
}

func TestLogsPanel_SetLogs(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)

	logs := []repository.LogLine{
		{Content: "Starting application", Container: "app", IsError: false},
		{Content: "Error: connection refused", Container: "app", IsError: true},
		{Content: "Shutting down", Container: "app", IsError: false},
	}
	lp.SetLogs(logs)

	if lp.LogCount() != 3 {
		t.Errorf("LogCount() = %d, want 3", lp.LogCount())
	}
	if lp.ErrorCount() != 1 {
		t.Errorf("ErrorCount() = %d, want 1", lp.ErrorCount())
	}
}

func TestLogsPanel_SetContainers(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)

	containers := []string{"app", "sidecar"}
	lp.SetContainers(containers)

	if len(lp.containers) != 2 {
		t.Errorf("len(containers) = %d, want 2", len(lp.containers))
	}
}

func TestLogsPanel_ToggleFollowing(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)

	// Default is following
	if !lp.following {
		t.Error("Default should be following")
	}

	// Toggle with 'f' key
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if lp.following {
		t.Error("After 'f' key should not be following")
	}

	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if !lp.following {
		t.Error("After second 'f' key should be following")
	}
}

func TestLogsPanel_Navigation(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)

	logs := []repository.LogLine{
		{Content: "Log 1"},
		{Content: "Log 2"},
		{Content: "Log 3"},
	}
	lp.SetLogs(logs)

	// Test scroll down
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Test scroll up
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	// Test page down
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	// Test page up
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	// Verify no panic occurs
}

func TestLogsPanel_Search(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)

	// Start search with '/'
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !lp.IsSearching() {
		t.Error("After '/' should be in search mode")
	}

	// Exit search with Enter
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if lp.IsSearching() {
		t.Error("After Enter should exit search mode")
	}
}

func TestLogsPanel_ClearSearch(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(100, 50)
	lp.filter = "test"
	lp.searching = true

	lp.ClearSearch()

	if lp.filter != "" {
		t.Error("ClearSearch should clear filter")
	}
	if lp.searching {
		t.Error("ClearSearch should stop searching")
	}
}

// ============================================
// PodActionMenu Tests
// ============================================

func TestNewPodActionMenu(t *testing.T) {
	pam := NewPodActionMenu()
	if pam.visible {
		t.Error("NewPodActionMenu should not be visible by default")
	}
}

func TestPodActionMenu_Init(t *testing.T) {
	pam := NewPodActionMenu()
	cmd := pam.Init()
	if cmd != nil {
		t.Error("PodActionMenu.Init() should return nil")
	}
}

func TestPodActionMenu_ShowHide(t *testing.T) {
	pam := NewPodActionMenu()

	items := []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Logs", Action: "exec"},
	}
	pam.Show("Pod Actions", items)

	if !pam.IsVisible() {
		t.Error("PodActionMenu should be visible after Show()")
	}
	if pam.title != "Pod Actions" {
		t.Errorf("title = %q, want %q", pam.title, "Pod Actions")
	}
	if len(pam.items) != 2 {
		t.Errorf("items count = %d, want 2", len(pam.items))
	}

	pam.Hide()
	if pam.IsVisible() {
		t.Error("PodActionMenu should not be visible after Hide()")
	}
}

func TestPodActionMenu_View_Hidden(t *testing.T) {
	pam := NewPodActionMenu()
	view := pam.View()
	if view != "" {
		t.Error("Hidden PodActionMenu View() should return empty string")
	}
}

func TestPodActionMenu_Update_NotVisible(t *testing.T) {
	pam := NewPodActionMenu()
	_, cmd := pam.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden menu should return nil cmd")
	}
}

func TestPodActionMenu_Update_EscKey(t *testing.T) {
	pam := NewPodActionMenu()
	items := []PodActionItem{{Label: "Test", Action: "delete"}}
	pam.Show("Test", items)

	pam, _ = pam.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if pam.visible {
		t.Error("Esc should hide the menu")
	}
}

func TestPodActionItem(t *testing.T) {
	item := PodActionItem{
		Label:       "Delete Pod",
		Description: "Permanently delete this pod",
		Action:      "delete",
		Command:     "kubectl delete pod",
	}
	if item.Label != "Delete Pod" {
		t.Errorf("Label = %q, want %q", item.Label, "Delete Pod")
	}
	if item.Action != "delete" {
		t.Errorf("Action = %q, want %q", item.Action, "delete")
	}
	if item.Description != "Permanently delete this pod" {
		t.Errorf("Description = %q, want %q", item.Description, "Permanently delete this pod")
	}
	if item.Command != "kubectl delete pod" {
		t.Errorf("Command = %q, want %q", item.Command, "kubectl delete pod")
	}
}

// ============================================
// WorkloadActionMenu Tests
// ============================================

func TestNewWorkloadActionMenu(t *testing.T) {
	wam := NewWorkloadActionMenu()
	if wam.IsVisible() {
		t.Error("NewWorkloadActionMenu should not be visible by default")
	}
}

func TestWorkloadActionMenu_Init(t *testing.T) {
	wam := NewWorkloadActionMenu()
	cmd := wam.Init()
	if cmd != nil {
		t.Error("WorkloadActionMenu.Init() should return nil")
	}
}

func TestWorkloadActionMenu_ShowHide(t *testing.T) {
	wam := NewWorkloadActionMenu()

	items := []WorkloadActionItem{
		{Label: "Scale", Action: "scale"},
		{Label: "Restart", Action: "restart"},
	}
	wam.Show("Workload Actions", items)

	if !wam.IsVisible() {
		t.Error("WorkloadActionMenu should be visible after Show()")
	}
	if wam.title != "Workload Actions" {
		t.Errorf("title = %q, want %q", wam.title, "Workload Actions")
	}

	wam.Hide()
	if wam.IsVisible() {
		t.Error("WorkloadActionMenu should not be visible after Hide()")
	}
}

func TestWorkloadActionMenu_View_Hidden(t *testing.T) {
	wam := NewWorkloadActionMenu()
	view := wam.View()
	if view != "" {
		t.Error("Hidden WorkloadActionMenu View() should return empty string")
	}
}

func TestWorkloadActionMenu_Update_NotVisible(t *testing.T) {
	wam := NewWorkloadActionMenu()
	_, cmd := wam.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden menu should return nil cmd")
	}
}

func TestWorkloadActionMenu_Update_EscKey(t *testing.T) {
	wam := NewWorkloadActionMenu()
	items := []WorkloadActionItem{{Label: "Test", Action: "scale"}}
	wam.Show("Test", items)

	wam, _ = wam.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if wam.visible {
		t.Error("Esc should hide the menu")
	}
}

func TestWorkloadActionItem(t *testing.T) {
	item := WorkloadActionItem{
		Label:       "Scale Up",
		Description: "Increase replicas",
		Action:      "scale",
		Replicas:    5,
		Command:     "kubectl scale --replicas=5",
	}
	if item.Label != "Scale Up" {
		t.Errorf("Label = %q, want %q", item.Label, "Scale Up")
	}
	if item.Action != "scale" {
		t.Errorf("Action = %q, want %q", item.Action, "scale")
	}
	if item.Description != "Increase replicas" {
		t.Errorf("Description = %q, want %q", item.Description, "Increase replicas")
	}
	if item.Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", item.Replicas)
	}
}

// ============================================
// HPAViewer Tests
// ============================================

func TestNewHPAViewer(t *testing.T) {
	hv := NewHPAViewer()
	if hv.IsVisible() {
		t.Error("NewHPAViewer should not be visible by default")
	}
}

func TestHPAViewer_Init(t *testing.T) {
	hv := NewHPAViewer()
	cmd := hv.Init()
	if cmd != nil {
		t.Error("HPAViewer.Init() should return nil")
	}
}

func TestHPAViewer_ShowHide(t *testing.T) {
	hv := NewHPAViewer()
	hpa := &repository.HPAData{
		Name:            "test-hpa",
		Namespace:       "default",
		Age:             "5d",
		Reference:       "Deployment/test-app",
		MinReplicas:     1,
		MaxReplicas:     10,
		CurrentReplicas: 3,
		DesiredReplicas: 5,
	}
	hv.Show(hpa, "default")

	if !hv.IsVisible() {
		t.Error("HPAViewer should be visible after Show()")
	}
	if hv.namespace != "default" {
		t.Errorf("namespace = %q, want %q", hv.namespace, "default")
	}

	hv.Hide()
	if hv.IsVisible() {
		t.Error("HPAViewer should not be visible after Hide()")
	}
}

func TestHPAViewer_View_Hidden(t *testing.T) {
	hv := NewHPAViewer()
	view := hv.View()
	if view != "" {
		t.Error("Hidden HPAViewer View() should return empty string")
	}
}

func TestHPAViewer_Update_NotVisible(t *testing.T) {
	hv := NewHPAViewer()
	_, cmd := hv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden viewer should return nil cmd")
	}
}

func TestHPAViewer_Update_EscKey(t *testing.T) {
	hv := NewHPAViewer()
	hpa := &repository.HPAData{Name: "test-hpa"}
	hv.Show(hpa, "default")

	hv, cmd := hv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if hv.visible {
		t.Error("Esc should hide the viewer")
	}
	if cmd == nil {
		t.Error("Esc should return HPAViewerClosed message")
	}
}

func TestHPAViewer_Update_QKey(t *testing.T) {
	hv := NewHPAViewer()
	hpa := &repository.HPAData{Name: "test-hpa"}
	hv.Show(hpa, "default")

	hv, cmd := hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if hv.visible {
		t.Error("q key should hide the viewer")
	}
	if cmd == nil {
		t.Error("q key should return a command")
	}
}

func TestHPAViewer_Update_Scrolling(t *testing.T) {
	hv := NewHPAViewer()
	hv.height = 50
	hv.width = 100
	hpa := &repository.HPAData{
		Name:            "test-hpa",
		Namespace:       "default",
		MinReplicas:     1,
		MaxReplicas:     10,
		CurrentReplicas: 3,
		DesiredReplicas: 5,
		Metrics: []repository.HPAMetricDetail{
			{Type: "Resource", Name: "cpu", Current: "50%", Target: "80%"},
		},
	}
	hv.Show(hpa, "default")

	// Test down key
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Test up key
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyUp})
	// Test j key
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Test k key
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	// Test pgdown
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	// Test pgup
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	// Test g (go to top)
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if hv.scroll != 0 {
		t.Errorf("g key should set scroll to 0, got %d", hv.scroll)
	}
}

func TestHPAViewer_SetSize(t *testing.T) {
	hv := NewHPAViewer()
	hv.SetSize(100, 50)
	if hv.width != 100 {
		t.Errorf("width = %d, want 100", hv.width)
	}
	if hv.height != 50 {
		t.Errorf("height = %d, want 50", hv.height)
	}
}

func TestHPAViewerClosed(t *testing.T) {
	msg := HPAViewerClosed{}
	_ = msg // Just ensure the type exists
}

// ============================================
// ConfigMapViewer Tests
// ============================================

func TestNewConfigMapViewer(t *testing.T) {
	cv := NewConfigMapViewer()
	if cv.IsVisible() {
		t.Error("NewConfigMapViewer should not be visible by default")
	}
}

func TestConfigMapViewer_Init(t *testing.T) {
	cv := NewConfigMapViewer()
	cmd := cv.Init()
	if cmd != nil {
		t.Error("ConfigMapViewer.Init() should return nil")
	}
}

func TestConfigMapViewer_ShowHide(t *testing.T) {
	cv := NewConfigMapViewer()
	cm := &repository.ConfigMapData{
		Name:      "test-cm",
		Namespace: "default",
		Age:       "5d",
		Data:      map[string]string{"key1": "value1"},
	}
	cv.Show(cm, "default")

	if !cv.IsVisible() {
		t.Error("ConfigMapViewer should be visible after Show()")
	}
	if cv.namespace != "default" {
		t.Errorf("namespace = %q, want %q", cv.namespace, "default")
	}

	cv.Hide()
	if cv.IsVisible() {
		t.Error("ConfigMapViewer should not be visible after Hide()")
	}
}

func TestConfigMapViewer_View_Hidden(t *testing.T) {
	cv := NewConfigMapViewer()
	view := cv.View()
	if view != "" {
		t.Error("Hidden ConfigMapViewer View() should return empty string")
	}
}

func TestConfigMapViewer_Update_NotVisible(t *testing.T) {
	cv := NewConfigMapViewer()
	_, cmd := cv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden viewer should return nil cmd")
	}
}

func TestConfigMapViewer_Update_EscKey(t *testing.T) {
	cv := NewConfigMapViewer()
	cm := &repository.ConfigMapData{Name: "test-cm"}
	cv.Show(cm, "default")

	cv, cmd := cv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cv.visible {
		t.Error("Esc should hide the viewer")
	}
	if cmd == nil {
		t.Error("Esc should return ConfigMapViewerClosed message")
	}
}

func TestConfigMapViewer_Update_Navigation(t *testing.T) {
	cv := NewConfigMapViewer()
	cv.height = 50
	cv.width = 100
	cm := &repository.ConfigMapData{
		Name:      "test-cm",
		Namespace: "default",
		Age:       "5d",
		Data:      map[string]string{"key1": "value1", "key2": "value2"},
	}
	cv.Show(cm, "default")

	// Test down key
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyDown})
	// Test up key
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyUp})
	// Test j key
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	// Test k key
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
}

func TestConfigMapViewer_Update_ActionMenu(t *testing.T) {
	cv := NewConfigMapViewer()
	cm := &repository.ConfigMapData{Name: "test-cm"}
	cv.Show(cm, "default")

	// Press 'a' to open action menu
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cv.mode != ConfigMapViewerModeAction {
		t.Error("'a' key should open action menu")
	}
}

func TestConfigMapViewer_SetSize(t *testing.T) {
	cv := NewConfigMapViewer()
	cv.SetSize(100, 50)
	if cv.width != 100 {
		t.Errorf("width = %d, want 100", cv.width)
	}
	if cv.height != 50 {
		t.Errorf("height = %d, want 50", cv.height)
	}
}

func TestConfigMapViewer_SetNamespaces(t *testing.T) {
	cv := NewConfigMapViewer()
	namespaces := []string{"default", "kube-system", "test"}
	cv.SetNamespaces(namespaces)
	if len(cv.namespaces) != 3 {
		t.Errorf("namespaces count = %d, want 3", len(cv.namespaces))
	}
}

func TestConfigMapViewer_SetStatusMsg(t *testing.T) {
	cv := NewConfigMapViewer()
	cv.SetStatusMsg("Copied!")
	if cv.statusMsg != "Copied!" {
		t.Errorf("statusMsg = %q, want %q", cv.statusMsg, "Copied!")
	}
}

func TestConfigMapViewerClosed(t *testing.T) {
	msg := ConfigMapViewerClosed{}
	_ = msg // Just ensure the type exists
}

func TestConfigMapValueCopied(t *testing.T) {
	msg := ConfigMapValueCopied{Key: "test-key"}
	if msg.Key != "test-key" {
		t.Errorf("Key = %q, want %q", msg.Key, "test-key")
	}
}

func TestConfigMapCopyRequest(t *testing.T) {
	req := ConfigMapCopyRequest{
		ConfigMapName:   "test-cm",
		SourceNamespace: "default",
		TargetNamespace: "production",
		AllNamespaces:   false,
	}
	if req.ConfigMapName != "test-cm" {
		t.Errorf("ConfigMapName = %q, want %q", req.ConfigMapName, "test-cm")
	}
}

// ============================================
// SecretViewer Tests
// ============================================

func TestNewSecretViewer(t *testing.T) {
	sv := NewSecretViewer()
	if sv.IsVisible() {
		t.Error("NewSecretViewer should not be visible by default")
	}
}

func TestSecretViewer_Init(t *testing.T) {
	sv := NewSecretViewer()
	cmd := sv.Init()
	if cmd != nil {
		t.Error("SecretViewer.Init() should return nil")
	}
}

func TestSecretViewer_ShowHide(t *testing.T) {
	sv := NewSecretViewer()
	secret := &repository.SecretData{
		Name:      "test-secret",
		Namespace: "default",
		Type:      "Opaque",
		Age:       "5d",
		Data:      map[string]string{"key1": "decoded-value"},
	}
	sv.Show(secret, "default")

	if !sv.IsVisible() {
		t.Error("SecretViewer should be visible after Show()")
	}
	if sv.namespace != "default" {
		t.Errorf("namespace = %q, want %q", sv.namespace, "default")
	}

	sv.Hide()
	if sv.IsVisible() {
		t.Error("SecretViewer should not be visible after Hide()")
	}
}

func TestSecretViewer_View_Hidden(t *testing.T) {
	sv := NewSecretViewer()
	view := sv.View()
	if view != "" {
		t.Error("Hidden SecretViewer View() should return empty string")
	}
}

func TestSecretViewer_Update_NotVisible(t *testing.T) {
	sv := NewSecretViewer()
	_, cmd := sv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden viewer should return nil cmd")
	}
}

func TestSecretViewer_Update_EscKey(t *testing.T) {
	sv := NewSecretViewer()
	secret := &repository.SecretData{Name: "test-secret"}
	sv.Show(secret, "default")

	sv, cmd := sv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if sv.visible {
		t.Error("Esc should hide the viewer")
	}
	if cmd == nil {
		t.Error("Esc should return SecretViewerClosed message")
	}
}

func TestSecretViewer_Update_Navigation(t *testing.T) {
	sv := NewSecretViewer()
	sv.height = 50
	sv.width = 100
	secret := &repository.SecretData{
		Name:      "test-secret",
		Namespace: "default",
		Type:      "Opaque",
		Age:       "5d",
		Data:      map[string]string{"key1": "value1", "key2": "value2"},
	}
	sv.Show(secret, "default")

	// Test navigation keys
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyDown})
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyUp})
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
}

func TestSecretViewer_Update_ActionMenu(t *testing.T) {
	sv := NewSecretViewer()
	secret := &repository.SecretData{Name: "test-secret"}
	sv.Show(secret, "default")

	// Press 'a' to open action menu
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if sv.mode != SecretViewerModeAction {
		t.Error("'a' key should open action menu")
	}
}

func TestSecretViewer_SetSize(t *testing.T) {
	sv := NewSecretViewer()
	sv.SetSize(100, 50)
	if sv.width != 100 {
		t.Errorf("width = %d, want 100", sv.width)
	}
	if sv.height != 50 {
		t.Errorf("height = %d, want 50", sv.height)
	}
}

func TestSecretViewer_SetNamespaces(t *testing.T) {
	sv := NewSecretViewer()
	namespaces := []string{"default", "kube-system", "test"}
	sv.SetNamespaces(namespaces)
	if len(sv.namespaces) != 3 {
		t.Errorf("namespaces count = %d, want 3", len(sv.namespaces))
	}
}

func TestSecretViewer_GetSecret(t *testing.T) {
	sv := NewSecretViewer()
	secret := &repository.SecretData{Name: "test-secret"}
	sv.Show(secret, "default")

	got := sv.GetSecret()
	if got == nil || got.Name != "test-secret" {
		t.Error("GetSecret should return the secret")
	}
}

func TestSecretViewer_GetNamespace(t *testing.T) {
	sv := NewSecretViewer()
	secret := &repository.SecretData{Name: "test-secret"}
	sv.Show(secret, "production")

	ns := sv.GetNamespace()
	if ns != "production" {
		t.Errorf("GetNamespace = %q, want %q", ns, "production")
	}
}

func TestSecretViewer_SetStatusMsg(t *testing.T) {
	sv := NewSecretViewer()
	sv.SetStatusMsg("Copied!")
	if sv.statusMsg != "Copied!" {
		t.Errorf("statusMsg = %q, want %q", sv.statusMsg, "Copied!")
	}
}

func TestSecretViewerClosed(t *testing.T) {
	msg := SecretViewerClosed{}
	_ = msg // Just ensure the type exists
}

func TestSecretValueCopied(t *testing.T) {
	msg := SecretValueCopied{Key: "test-key"}
	if msg.Key != "test-key" {
		t.Errorf("Key = %q, want %q", msg.Key, "test-key")
	}
}

func TestSecretCopyRequest(t *testing.T) {
	req := SecretCopyRequest{
		SecretName:      "test-secret",
		SourceNamespace: "default",
		TargetNamespace: "production",
		AllNamespaces:   false,
	}
	if req.SecretName != "test-secret" {
		t.Errorf("SecretName = %q, want %q", req.SecretName, "test-secret")
	}
}

// ============================================
// Navigator Tests
// ============================================

func TestNewNavigator(t *testing.T) {
	nav := NewNavigator()
	if nav.mode != ModeWorkloads {
		t.Errorf("mode = %v, want ModeWorkloads (0)", nav.mode)
	}
	if nav.resourceType != repository.ResourceDeployments {
		t.Errorf("resourceType = %v, want ResourceDeployments", nav.resourceType)
	}
	if nav.searching {
		t.Error("searching should be false by default")
	}
}

func TestNavigator_Init(t *testing.T) {
	nav := NewNavigator()
	cmd := nav.Init()
	if cmd != nil {
		t.Error("Navigator.Init() should return nil")
	}
}

func TestNavigator_SetSize(t *testing.T) {
	nav := NewNavigator()
	nav.SetSize(100, 50)
	if nav.width != 100 {
		t.Errorf("width = %d, want 100", nav.width)
	}
	if nav.height != 50 {
		t.Errorf("height = %d, want 50", nav.height)
	}
}

func TestNavigator_SetMode(t *testing.T) {
	nav := NewNavigator()
	nav.SetMode(ModeNamespace)
	if nav.mode != ModeNamespace {
		t.Errorf("mode = %v, want ModeNamespace", nav.mode)
	}
}

func TestNavigator_Mode(t *testing.T) {
	nav := NewNavigator()
	nav.SetMode(ModeResources)
	if nav.Mode() != ModeResources {
		t.Errorf("Mode() = %v, want ModeResources", nav.Mode())
	}
}

func TestNavigator_SetWorkloads(t *testing.T) {
	nav := NewNavigator()
	workloads := []repository.WorkloadInfo{
		{Name: "deploy-1", Namespace: "default"},
		{Name: "deploy-2", Namespace: "default"},
	}
	nav.SetWorkloads(workloads)
	if len(nav.workloads) != 2 {
		t.Errorf("workloads count = %d, want 2", len(nav.workloads))
	}
}

func TestNavigator_SetPods(t *testing.T) {
	nav := NewNavigator()
	pods := []repository.PodInfo{
		{Name: "pod-1", Namespace: "default"},
		{Name: "pod-2", Namespace: "default"},
	}
	nav.SetPods(pods)
	if len(nav.pods) != 2 {
		t.Errorf("pods count = %d, want 2", len(nav.pods))
	}
}

func TestNavigator_SetNamespaces(t *testing.T) {
	nav := NewNavigator()
	namespaces := []repository.NamespaceInfo{
		{Name: "default", Status: "Active"},
		{Name: "kube-system", Status: "Active"},
	}
	nav.SetNamespaces(namespaces)
	if len(nav.namespaces) != 2 {
		t.Errorf("namespaces count = %d, want 2", len(nav.namespaces))
	}
}

func TestNavigator_SetHPAs(t *testing.T) {
	nav := NewNavigator()
	hpas := []repository.HPAInfo{
		{Name: "hpa-1", Reference: "Deployment/test"},
	}
	nav.SetHPAs(hpas)
	if len(nav.hpas) != 1 {
		t.Errorf("hpas count = %d, want 1", len(nav.hpas))
	}
}

func TestNavigator_SetConfigMaps(t *testing.T) {
	nav := NewNavigator()
	cms := []repository.ConfigMapInfo{
		{Name: "cm-1", Keys: 3},
	}
	nav.SetConfigMaps(cms)
	if len(nav.configmaps) != 1 {
		t.Errorf("configmaps count = %d, want 1", len(nav.configmaps))
	}
}

func TestNavigator_SetSecrets(t *testing.T) {
	nav := NewNavigator()
	secrets := []repository.SecretInfo{
		{Name: "secret-1", Type: "Opaque", Keys: 2},
	}
	nav.SetSecrets(secrets)
	if len(nav.secrets) != 1 {
		t.Errorf("secrets count = %d, want 1", len(nav.secrets))
	}
}

func TestNavigator_SetResourceType(t *testing.T) {
	nav := NewNavigator()
	nav.SetResourceType(repository.ResourceStatefulSets)
	if nav.resourceType != repository.ResourceStatefulSets {
		t.Errorf("resourceType = %v, want ResourceStatefulSets", nav.resourceType)
	}
}

func TestNavigator_ResourceType(t *testing.T) {
	nav := NewNavigator()
	nav.SetResourceType(repository.ResourceDaemonSets)
	if nav.ResourceType() != repository.ResourceDaemonSets {
		t.Errorf("ResourceType() = %v, want ResourceDaemonSets", nav.ResourceType())
	}
}

func TestNavigator_ClearSearch(t *testing.T) {
	nav := NewNavigator()
	nav.searchQuery = "test"
	nav.searching = true
	nav.ClearSearch()
	if nav.searchQuery != "" {
		t.Errorf("searchQuery should be empty after ClearSearch()")
	}
	if nav.searching {
		t.Error("searching should be false after ClearSearch()")
	}
}

func TestNavigator_Section(t *testing.T) {
	nav := NewNavigator()
	// Default section should be SectionPods (0)
	if nav.Section() != SectionPods {
		t.Errorf("Section() = %v, want SectionPods", nav.Section())
	}
}

func TestNavigatorMode(t *testing.T) {
	tests := []struct {
		name string
		mode NavigatorMode
	}{
		{"ModeWorkloads", ModeWorkloads},
		{"ModeResources", ModeResources},
		{"ModeNamespace", ModeNamespace},
		{"ModeResourceType", ModeResourceType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nav := NewNavigator()
			nav.SetMode(tt.mode)
			if nav.Mode() != tt.mode {
				t.Errorf("Mode() = %v, want %v", nav.Mode(), tt.mode)
			}
		})
	}
}

func TestPodViewSection(t *testing.T) {
	sections := []PodViewSection{
		SectionPods,
		SectionHPAs,
		SectionConfigMaps,
		SectionSecrets,
		SectionDockerRegistry,
	}

	for i, section := range sections {
		if int(section) != i {
			t.Errorf("Section %d should have value %d", section, i)
		}
	}
}

// ============================================
// DockerRegistryViewer Tests
// ============================================

func TestNewDockerRegistryViewer(t *testing.T) {
	drv := NewDockerRegistryViewer()
	if drv.IsVisible() {
		t.Error("NewDockerRegistryViewer should not be visible by default")
	}
}

func TestDockerRegistryViewer_Init(t *testing.T) {
	drv := NewDockerRegistryViewer()
	cmd := drv.Init()
	if cmd != nil {
		t.Error("DockerRegistryViewer.Init() should return nil")
	}
}

func TestDockerRegistryViewer_ShowHide(t *testing.T) {
	drv := NewDockerRegistryViewer()
	secret := &repository.SecretData{
		Name: "registry-secret",
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string]string{".dockerconfigjson": "{}"},
	}
	drv.Show(secret, "default")

	if !drv.IsVisible() {
		t.Error("DockerRegistryViewer should be visible after Show()")
	}

	drv.Hide()
	if drv.IsVisible() {
		t.Error("DockerRegistryViewer should not be visible after Hide()")
	}
}

func TestDockerRegistryViewer_View_Hidden(t *testing.T) {
	drv := NewDockerRegistryViewer()
	view := drv.View()
	if view != "" {
		t.Error("Hidden DockerRegistryViewer View() should return empty string")
	}
}

func TestDockerRegistryViewer_Update_NotVisible(t *testing.T) {
	drv := NewDockerRegistryViewer()
	_, cmd := drv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("Update on hidden viewer should return nil cmd")
	}
}

func TestDockerRegistryViewer_Update_EscKey(t *testing.T) {
	drv := NewDockerRegistryViewer()
	secret := &repository.SecretData{
		Name: "registry-secret",
		Type: "kubernetes.io/dockerconfigjson",
	}
	drv.Show(secret, "default")

	drv, cmd := drv.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if drv.visible {
		t.Error("Esc should hide the viewer")
	}
	if cmd == nil {
		t.Error("Esc should return DockerRegistryViewerClosed message")
	}
}

func TestDockerRegistryViewer_SetSize(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetSize(100, 50)
	if drv.width != 100 {
		t.Errorf("width = %d, want 100", drv.width)
	}
	if drv.height != 50 {
		t.Errorf("height = %d, want 50", drv.height)
	}
}

func TestDockerRegistryViewer_SetNamespaces(t *testing.T) {
	drv := NewDockerRegistryViewer()
	namespaces := []string{"default", "kube-system"}
	drv.SetNamespaces(namespaces)
	if len(drv.namespaces) != 2 {
		t.Errorf("namespaces count = %d, want 2", len(drv.namespaces))
	}
}

func TestDockerRegistryViewer_SetStatusMsg(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetStatusMsg("Success!")
	if drv.statusMsg != "Success!" {
		t.Errorf("statusMsg = %q, want %q", drv.statusMsg, "Success!")
	}
}

func TestDockerRegistryViewerClosed(t *testing.T) {
	msg := DockerRegistryViewerClosed{}
	_ = msg // Just ensure the type exists
}

func TestDockerRegistryCopyRequest(t *testing.T) {
	req := DockerRegistryCopyRequest{
		SecretName:      "registry-secret",
		SourceNamespace: "default",
		TargetNamespace: "production",
		AllNamespaces:   false,
	}
	if req.SecretName != "registry-secret" {
		t.Errorf("SecretName = %q, want %q", req.SecretName, "registry-secret")
	}
}

// ============================================
// Additional Action Menu Tests
// ============================================

func TestPodActionMenu_Update_Enter(t *testing.T) {
	menu := NewPodActionMenu()
	items := []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Logs", Action: "logs"},
	}
	menu.Show("test-pod", items)

	// Press Enter to select action
	menu, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if menu.visible {
		t.Error("Menu should hide after Enter")
	}
	if cmd == nil {
		t.Error("Enter should return a command")
	}
}

func TestPodActionMenu_Update_UpDown(t *testing.T) {
	menu := NewPodActionMenu()
	items := []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Logs", Action: "logs"},
		{Label: "Exec", Action: "exec"},
	}
	menu.Show("test-pod", items)

	// Press Down
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	if menu.selected != 1 {
		t.Errorf("selected = %d, want 1 after Down", menu.selected)
	}

	// Press Down again
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	if menu.selected != 2 {
		t.Errorf("selected = %d, want 2 after second Down", menu.selected)
	}

	// Press Up
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyUp})
	if menu.selected != 1 {
		t.Errorf("selected = %d, want 1 after Up", menu.selected)
	}
}

func TestPodActionMenu_Update_JK(t *testing.T) {
	menu := NewPodActionMenu()
	items := []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Logs", Action: "logs"},
	}
	menu.Show("test-pod", items)

	// Press j
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if menu.selected != 1 {
		t.Errorf("selected = %d, want 1 after j", menu.selected)
	}

	// Press k
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if menu.selected != 0 {
		t.Errorf("selected = %d, want 0 after k", menu.selected)
	}
}

func TestWorkloadActionMenu_Update_Enter(t *testing.T) {
	menu := NewWorkloadActionMenu()
	items := []WorkloadActionItem{
		{Label: "Scale Up", Action: "scale"},
		{Label: "Restart", Action: "restart"},
	}
	menu.Show("web-app", items)

	// Press Enter to select action
	menu, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if menu.visible {
		t.Error("Menu should hide after Enter")
	}
	if cmd == nil {
		t.Error("Enter should return a command")
	}
}

func TestWorkloadActionMenu_Update_UpDown(t *testing.T) {
	menu := NewWorkloadActionMenu()
	items := []WorkloadActionItem{
		{Label: "Scale Up", Action: "scale"},
		{Label: "Restart", Action: "restart"},
	}
	menu.Show("web-app", items)

	// Press Down
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})
	if menu.selected != 1 {
		t.Errorf("selected = %d, want 1 after Down", menu.selected)
	}

	// Press Up
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyUp})
	if menu.selected != 0 {
		t.Errorf("selected = %d, want 0 after Up", menu.selected)
	}
}

// ============================================
// Additional ConfigMap Viewer Tests
// ============================================

func TestConfigMapViewer_View_Visible2(t *testing.T) {
	cv := NewConfigMapViewer()
	cv.SetSize(80, 40)
	cv.Show(&repository.ConfigMapData{
		Name:      "app-config",
		Namespace: "default",
		Data:      map[string]string{"key1": "value1"},
	}, "default")

	view := cv.View()
	if view == "" {
		t.Error("Visible ConfigMapViewer View() should not return empty string")
	}
	if !strings.Contains(view, "app-config") {
		t.Error("View should contain configmap name")
	}
}

func TestConfigMapViewer_Update_ScrollKeys(t *testing.T) {
	cv := NewConfigMapViewer()
	cv.SetSize(80, 20)
	cv.Show(&repository.ConfigMapData{
		Name:      "app-config",
		Namespace: "default",
		Data:      map[string]string{"key1": strings.Repeat("long value ", 100)},
	}, "default")

	// Press Down
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press Up
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Press PgDown
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Press PgUp
	cv, _ = cv.Update(tea.KeyMsg{Type: tea.KeyPgUp})
}

// ============================================
// Additional Secret Viewer Tests
// ============================================

func TestSecretViewer_View_Visible2(t *testing.T) {
	sv := NewSecretViewer()
	sv.SetSize(80, 40)
	sv.Show(&repository.SecretData{
		Name:      "db-credentials",
		Namespace: "default",
		Type:      "Opaque",
		Data:      map[string]string{"password": "secret123"},
	}, "default")

	view := sv.View()
	if view == "" {
		t.Error("Visible SecretViewer View() should not return empty string")
	}
	if !strings.Contains(view, "db-credentials") {
		t.Error("View should contain secret name")
	}
}

// ============================================
// Additional DockerRegistry Viewer Tests
// ============================================

func TestDockerRegistryViewer_View_Visible(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetSize(80, 40)
	drv.Show(&repository.SecretData{
		Name:      "docker-secret",
		Namespace: "default",
		Type:      "kubernetes.io/dockerconfigjson",
		Data:      map[string]string{".dockerconfigjson": `{"auths":{}}`},
	}, "default")

	view := drv.View()
	if view == "" {
		t.Error("Visible DockerRegistryViewer View() should not return empty string")
	}
}

func TestDockerRegistryViewer_Update_Navigation(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetSize(80, 40)
	drv.Show(&repository.SecretData{
		Name:      "docker-secret",
		Namespace: "default",
		Type:      "kubernetes.io/dockerconfigjson",
		Data:      map[string]string{".dockerconfigjson": `{"auths":{}}`},
	}, "default")

	// Press j to move down
	drv, _ = drv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Press k to move up
	drv, _ = drv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Press q to close
	drv, cmd := drv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("q key should return close command")
	}
}

// ============================================
// Additional HPA Viewer Tests
// ============================================

func TestHPAViewer_View_Visible(t *testing.T) {
	hv := NewHPAViewer()
	hv.SetSize(80, 40)
	hv.Show(&repository.HPAData{
		Name:            "web-hpa",
		Namespace:       "default",
		MinReplicas:     1,
		MaxReplicas:     10,
		CurrentReplicas: 3,
		DesiredReplicas: 3,
		Reference:       "Deployment/web-app",
	}, "default")

	view := hv.View()
	if view == "" {
		t.Error("Visible HPAViewer View() should not return empty string")
	}
	if !strings.Contains(view, "web-hpa") {
		t.Error("View should contain HPA name")
	}
}

func TestHPAViewer_Update_Scroll(t *testing.T) {
	hv := NewHPAViewer()
	hv.SetSize(80, 20)
	hv.Show(&repository.HPAData{
		Name:            "web-hpa",
		Namespace:       "default",
		MinReplicas:     1,
		MaxReplicas:     10,
		CurrentReplicas: 3,
		DesiredReplicas: 3,
		Reference:       "Deployment/web-app",
		Metrics: []repository.HPAMetricDetail{
			{Type: "Resource", Name: "cpu", Current: "50%", Target: "80%"},
			{Type: "Resource", Name: "memory", Current: "60%", Target: "70%"},
		},
	}, "default")

	// Press j to scroll down
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Press k to scroll up
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Press g to go to top
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Press G to go to bottom
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

// ============================================
// Navigator Additional Tests
// ============================================

func TestNavigator_Update_Navigation(t *testing.T) {
	nav := NewNavigator()
	nav.SetNamespaces([]repository.NamespaceInfo{
		{Name: "default", Status: "Active"},
		{Name: "kube-system", Status: "Active"},
	})

	// Press j to move down
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Press k to move up
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Press down arrow
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Press up arrow
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestNavigator_FilterMode(t *testing.T) {
	nav := NewNavigator()
	nav.SetNamespaces([]repository.NamespaceInfo{
		{Name: "default", Status: "Active"},
		{Name: "kube-system", Status: "Active"},
		{Name: "production", Status: "Active"},
	})

	// Press / to enter filter mode
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type filter text
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	// Press c to clear filter
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
}

// ============================================
// Events Panel Additional Tests
// ============================================

func TestEventsPanel_Update_Filter(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetEvents([]repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled", Message: "Pod scheduled"},
		{Type: "Warning", Reason: "BackOff", Message: "Container restarting"},
	})

	// Press / to filter
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
}

// ============================================
// Logs Panel Additional Tests
// ============================================

func TestLogsPanel_Update_Filter(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetLogs([]repository.LogLine{
		{Content: "Starting application", Container: "app"},
		{Content: "Error occurred", Container: "app", IsError: true},
	})

	// Press / to enter filter mode
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
}

// ============================================
// KubectlCommands Tests
// ============================================

func TestKubectlCommands_Basic(t *testing.T) {
	items := KubectlCommands("default", "my-pod", "", nil)

	if len(items) == 0 {
		t.Error("KubectlCommands should return items")
	}

	// Check that basic commands are present
	hasLogs := false
	hasDescribe := false
	hasDelete := false
	for _, item := range items {
		if strings.Contains(item.Value, "kubectl logs") {
			hasLogs = true
		}
		if strings.Contains(item.Value, "kubectl describe") {
			hasDescribe = true
		}
		if strings.Contains(item.Value, "kubectl delete") {
			hasDelete = true
		}
	}

	if !hasLogs {
		t.Error("Should have logs command")
	}
	if !hasDescribe {
		t.Error("Should have describe command")
	}
	if !hasDelete {
		t.Error("Should have delete command")
	}
}

func TestKubectlCommands_WithContainer(t *testing.T) {
	containers := []string{"app", "sidecar"}
	items := KubectlCommands("production", "web-pod", "app", containers)

	if len(items) == 0 {
		t.Error("KubectlCommands should return items")
	}

	// Should have container-specific commands at the beginning
	hasContainerLogs := false
	hasContainerExec := false
	for _, item := range items {
		if strings.Contains(item.Label, "container 'app'") {
			hasContainerLogs = true
		}
		if strings.Contains(item.Label, "into 'app'") {
			hasContainerExec = true
		}
	}

	if !hasContainerLogs {
		t.Error("Should have container-specific logs command")
	}
	if !hasContainerExec {
		t.Error("Should have container-specific exec command")
	}
}

func TestKubectlCommands_WithContainerNoPrevious(t *testing.T) {
	// Test with single container but no containerName (edge case)
	containers := []string{"main"}
	items := KubectlCommands("default", "pod", "", containers)

	hasPrevious := false
	for _, item := range items {
		if strings.Contains(item.Label, "previous") {
			hasPrevious = true
		}
	}

	if !hasPrevious {
		t.Error("Should have previous logs command")
	}
}

// ============================================
// ScaleActions Tests
// ============================================

func TestScaleActions_Basic(t *testing.T) {
	items := ScaleActions("default", "web-app", "deployment", 3)

	if len(items) == 0 {
		t.Error("ScaleActions should return items")
	}

	// Should have scale options
	hasScale0 := false
	hasScale1 := false
	hasCopy := false
	for _, item := range items {
		if item.Label == "Scale to 0" {
			hasScale0 = true
		}
		if item.Label == "Scale to 1" {
			hasScale1 = true
		}
		if item.Action == "copy" {
			hasCopy = true
		}
	}

	if !hasScale0 {
		t.Error("Should have scale to 0 option")
	}
	if !hasScale1 {
		t.Error("Should have scale to 1 option")
	}
	if !hasCopy {
		t.Error("Should have copy command option")
	}
}

func TestScaleActions_CurrentPlus(t *testing.T) {
	items := ScaleActions("default", "app", "deployment", 2)

	// Should have current+1 (3)
	hasCurrentPlus := false
	for _, item := range items {
		if strings.Contains(item.Label, "current+1") {
			hasCurrentPlus = true
		}
	}

	if !hasCurrentPlus {
		t.Error("Should have current+1 option")
	}
}

func TestScaleActions_CurrentMinus(t *testing.T) {
	items := ScaleActions("default", "app", "deployment", 5)

	// Should have current-1 (4)
	hasCurrentMinus := false
	for _, item := range items {
		if strings.Contains(item.Label, "current-1") {
			hasCurrentMinus = true
		}
	}

	if !hasCurrentMinus {
		t.Error("Should have current-1 option")
	}
}

func TestScaleActions_ZeroReplicas(t *testing.T) {
	items := ScaleActions("default", "app", "deployment", 0)

	// Should NOT have current-1 when at 0
	hasCurrentMinus := false
	for _, item := range items {
		if strings.Contains(item.Label, "current-1") {
			hasCurrentMinus = true
		}
	}

	if hasCurrentMinus {
		t.Error("Should not have current-1 option when at 0 replicas")
	}
}

func TestScaleActions_HighReplicas(t *testing.T) {
	items := ScaleActions("default", "app", "deployment", 10)

	// Should NOT have current+1 when at 10
	hasCurrentPlus := false
	for _, item := range items {
		if strings.Contains(item.Label, "current+1") {
			hasCurrentPlus = true
		}
	}

	if hasCurrentPlus {
		t.Error("Should not have current+1 option when at 10 replicas")
	}
}

// ============================================
// PodActions Tests
// ============================================

func TestPodActions_SingleContainer(t *testing.T) {
	containers := []string{"app"}
	items := PodActions("default", "my-pod", containers)

	if len(items) == 0 {
		t.Error("PodActions should return items")
	}

	// Should have delete, exec, port-forward, describe
	hasDelete := false
	hasExec := false
	hasPortForward := false
	hasDescribe := false
	for _, item := range items {
		if item.Action == "delete" {
			hasDelete = true
		}
		if item.Action == "exec" {
			hasExec = true
		}
		if item.Action == "port-forward" {
			hasPortForward = true
		}
		if item.Action == "describe" {
			hasDescribe = true
		}
	}

	if !hasDelete {
		t.Error("Should have delete action")
	}
	if !hasExec {
		t.Error("Should have exec action")
	}
	if !hasPortForward {
		t.Error("Should have port-forward action")
	}
	if !hasDescribe {
		t.Error("Should have describe action")
	}
}

func TestPodActions_MultiContainer(t *testing.T) {
	containers := []string{"app", "sidecar", "init"}
	items := PodActions("default", "my-pod", containers)

	// Should have exec options for each container
	execCount := 0
	for _, item := range items {
		if item.Action == "exec" && strings.Contains(item.Label, "Exec into") {
			execCount++
		}
	}

	if execCount != len(containers) {
		t.Errorf("Should have exec option for each container, got %d, want %d", execCount, len(containers))
	}
}

func TestPodActions_NoContainers(t *testing.T) {
	items := PodActions("default", "my-pod", nil)

	// Should still have basic actions
	if len(items) == 0 {
		t.Error("PodActions should return items even with no containers")
	}

	hasDelete := false
	for _, item := range items {
		if item.Action == "delete" {
			hasDelete = true
		}
	}

	if !hasDelete {
		t.Error("Should have delete action even with no containers")
	}
}

// ============================================
// ActionMenu Update Tests
// ============================================

func TestActionMenu_Update_EnterSelection(t *testing.T) {
	menu := NewActionMenu()
	menu.Show("Test Menu", []MenuItem{
		{Label: "Option 1", Value: "opt1"},
		{Label: "Option 2", Value: "opt2"},
	})

	// Press Enter to select
	menu, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enter key should return selection command")
	}
}

// ============================================
// PodActionMenu Extended Tests
// ============================================

func TestPodActionMenu_Update_EnterKey(t *testing.T) {
	menu := NewPodActionMenu()
	menu.Show("test-pod", []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Exec", Action: "exec"},
	})

	// Press Enter to select
	menu, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enter key should return selection command")
	}
}

func TestPodActionMenu_Update_UpKey(t *testing.T) {
	menu := NewPodActionMenu()
	menu.Show("test-pod", []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Exec", Action: "exec"},
	})

	// Navigate down first
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestPodActionMenu_Update_KKey(t *testing.T) {
	menu := NewPodActionMenu()
	menu.Show("test-pod", []PodActionItem{
		{Label: "Delete", Action: "delete"},
		{Label: "Exec", Action: "exec"},
	})

	// Navigate down with j
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Navigate up with k
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
}

func TestPodActionMenu_Update_QKey(t *testing.T) {
	menu := NewPodActionMenu()
	menu.Show("test-pod", []PodActionItem{
		{Label: "Delete", Action: "delete"},
	})

	// Press q to close
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if menu.IsVisible() {
		t.Error("q key should close menu")
	}
}

func TestPodActionMenu_View_Visible(t *testing.T) {
	menu := NewPodActionMenu()
	menu.Show("test-pod", []PodActionItem{
		{Label: "Delete Pod", Action: "delete", Description: "removes pod"},
		{Label: "Exec into pod", Action: "exec", Description: "opens shell"},
	})

	view := menu.View()
	if view == "" {
		t.Error("Visible PodActionMenu View() should not return empty string")
	}
	if !strings.Contains(view, "test-pod") {
		t.Error("View should contain pod name")
	}
}

// ============================================
// WorkloadActionMenu Extended Tests
// ============================================

func TestWorkloadActionMenu_Update_EnterKey(t *testing.T) {
	menu := NewWorkloadActionMenu()
	menu.Show("test-deployment", []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
		{Label: "Scale to 1", Action: "scale", Replicas: 1},
	})

	// Press Enter to select
	menu, cmd := menu.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("Enter key should return selection command")
	}
}

func TestWorkloadActionMenu_Update_UpKey(t *testing.T) {
	menu := NewWorkloadActionMenu()
	menu.Show("test-deployment", []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
		{Label: "Scale to 1", Action: "scale", Replicas: 1},
	})

	// Navigate down first
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Navigate up
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyUp})
}

func TestWorkloadActionMenu_Update_JKKeys(t *testing.T) {
	menu := NewWorkloadActionMenu()
	menu.Show("test-deployment", []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
		{Label: "Scale to 1", Action: "scale", Replicas: 1},
	})

	// Navigate with j/k
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
}

func TestWorkloadActionMenu_Update_QKey(t *testing.T) {
	menu := NewWorkloadActionMenu()
	menu.Show("test-deployment", []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
	})

	// Press q to close
	menu, _ = menu.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if menu.IsVisible() {
		t.Error("q key should close menu")
	}
}

func TestWorkloadActionMenu_View_Visible(t *testing.T) {
	menu := NewWorkloadActionMenu()
	menu.Show("web-deployment", []WorkloadActionItem{
		{Label: "Scale to 0", Action: "scale", Replicas: 0},
		{Label: "Restart", Action: "restart"},
	})

	view := menu.View()
	if view == "" {
		t.Error("Visible WorkloadActionMenu View() should not return empty string")
	}
	if !strings.Contains(view, "web-deployment") {
		t.Error("View should contain workload name")
	}
}

// ============================================
// ConfigMapViewer Extended Tests
// ============================================

func TestConfigMapViewer_Update_EnterKey(t *testing.T) {
	cmv := NewConfigMapViewer()
	cmv.SetSize(80, 40)
	cmv.Show(&repository.ConfigMapData{
		Name:      "test-cm",
		Namespace: "default",
		Data: map[string]string{
			"key1": "value1",
		},
	}, "default")

	// Press Enter (copy to clipboard)
	cmv, _ = cmv.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestConfigMapViewer_Update_GKey(t *testing.T) {
	cmv := NewConfigMapViewer()
	cmv.SetSize(80, 40)
	cmv.Show(&repository.ConfigMapData{
		Name:      "test-cm",
		Namespace: "default",
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}, "default")

	// Press g to go to top
	cmv, _ = cmv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Press G to go to bottom
	cmv, _ = cmv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

func TestConfigMapViewer_Update_PageKeys(t *testing.T) {
	cmv := NewConfigMapViewer()
	cmv.SetSize(80, 20)
	cmv.Show(&repository.ConfigMapData{
		Name:      "test-cm",
		Namespace: "default",
		Data: map[string]string{
			"key1": "long value " + strings.Repeat("x", 500),
			"key2": "value2",
		},
	}, "default")

	// Press PgDn
	cmv, _ = cmv.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Press PgUp
	cmv, _ = cmv.Update(tea.KeyMsg{Type: tea.KeyPgUp})
}

// ============================================
// SecretViewer Extended Tests
// ============================================

func TestSecretViewer_Update_EnterKey(t *testing.T) {
	sv := NewSecretViewer()
	sv.SetSize(80, 40)
	sv.Show(&repository.SecretData{
		Name:      "test-secret",
		Namespace: "default",
		Data: map[string]string{
			"username": "admin",
		},
	}, "default")

	// Press Enter (copy to clipboard)
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestSecretViewer_Update_GKey(t *testing.T) {
	sv := NewSecretViewer()
	sv.SetSize(80, 40)
	sv.Show(&repository.SecretData{
		Name:      "test-secret",
		Namespace: "default",
		Data: map[string]string{
			"username": "admin",
			"password": "secret",
		},
	}, "default")

	// Press g to go to top
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Press G to go to bottom
	sv, _ = sv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

// ============================================
// DockerRegistryViewer Extended Tests
// ============================================

func TestDockerRegistryViewer_Update_EnterKey(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetSize(80, 40)
	drv.Show(&repository.SecretData{
		Name:      "registry-secret",
		Namespace: "default",
		Data: map[string]string{
			".dockerconfigjson": `{"auths":{"registry.io":{"auth":"dXNlcjpwYXNz"}}}`,
		},
	}, "default")

	// Press Enter (copy to clipboard)
	drv, _ = drv.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestDockerRegistryViewer_Update_GKey(t *testing.T) {
	drv := NewDockerRegistryViewer()
	drv.SetSize(80, 40)
	drv.Show(&repository.SecretData{
		Name:      "registry-secret",
		Namespace: "default",
		Data: map[string]string{
			".dockerconfigjson": `{"auths":{"registry.io":{"auth":"dXNlcjpwYXNz"}}}`,
		},
	}, "default")

	// Press g to go to top
	drv, _ = drv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Press G to go to bottom
	drv, _ = drv.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

// ============================================
// Navigator Extended Tests
// ============================================

func TestNavigator_Update_TabKey(t *testing.T) {
	nav := NewNavigator()
	nav.SetMode(ModeResources)
	nav.SetPods([]repository.PodInfo{
		{Name: "web-pod", Status: "Running"},
	})
	nav.SetHPAs([]repository.HPAInfo{
		{Name: "web-hpa"},
	})

	// Press Tab to cycle sections
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyTab})
}

func TestNavigator_Update_ShiftTabKey(t *testing.T) {
	nav := NewNavigator()
	nav.SetMode(ModeResources)
	nav.SetPods([]repository.PodInfo{
		{Name: "web-pod", Status: "Running"},
	})

	// Press Shift+Tab to cycle sections backwards
	nav, _ = nav.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
}

// ============================================
// EventsPanel Extended Tests
// ============================================

func TestEventsPanel_Update_WKey(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetEvents([]repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled"},
		{Type: "Warning", Reason: "BackOff"},
	})

	// Press w to toggle warnings filter
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
}

func TestEventsPanel_Update_CKey(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetEvents([]repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled"},
	})

	// Enter filter mode
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Press c to clear filter
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
}

func TestEventsPanel_Update_EscKey(t *testing.T) {
	ep := NewEventsPanel()
	ep.SetEvents([]repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled"},
	})

	// Enter filter mode
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Press Esc
	ep, _ = ep.Update(tea.KeyMsg{Type: tea.KeyEsc})
}

// ============================================
// LogsPanel Extended Tests
// ============================================

func TestLogsPanel_Update_WKey(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetLogs([]repository.LogLine{
		{Content: "Normal log", IsError: false},
		{Content: "Error log", IsError: true},
	})

	// Press w to toggle errors filter
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
}

func TestLogsPanel_Update_CKey(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetLogs([]repository.LogLine{
		{Content: "Normal log", IsError: false},
	})

	// Enter filter mode
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})

	// Press c to clear filter
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
}

func TestLogsPanel_Update_GKey(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetLogs([]repository.LogLine{
		{Content: "Log 1", IsError: false},
		{Content: "Log 2", IsError: false},
	})

	// Press g to go to top
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Press G to go to bottom
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
}

func TestLogsPanel_Update_EscKey(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetLogs([]repository.LogLine{
		{Content: "Normal log", IsError: false},
	})

	// Enter filter mode
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Press Esc
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyEsc})
}

func TestLogsPanel_Update_PageKeys(t *testing.T) {
	lp := NewLogsPanel()
	lp.SetSize(80, 10)
	lp.SetLogs([]repository.LogLine{
		{Content: "Log 1"},
		{Content: "Log 2"},
		{Content: "Log 3"},
	})

	// Press PgDn
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Press PgUp
	lp, _ = lp.Update(tea.KeyMsg{Type: tea.KeyPgUp})
}

// ============================================
// HPA Viewer Extended Tests
// ============================================

func TestHPAViewer_Update_EnterKey(t *testing.T) {
	hv := NewHPAViewer()
	hv.SetSize(80, 40)
	hv.Show(&repository.HPAData{
		Name:      "web-hpa",
		Namespace: "default",
	}, "default")

	// Press Enter (copy to clipboard)
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyEnter})
}

func TestHPAViewer_Update_PageKeys(t *testing.T) {
	hv := NewHPAViewer()
	hv.SetSize(80, 20)
	hv.Show(&repository.HPAData{
		Name:      "web-hpa",
		Namespace: "default",
		Metrics: []repository.HPAMetricDetail{
			{Type: "Resource", Name: "cpu"},
			{Type: "Resource", Name: "memory"},
		},
	}, "default")

	// Press PgDn
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyPgDown})

	// Press PgUp
	hv, _ = hv.Update(tea.KeyMsg{Type: tea.KeyPgUp})
}

// ============================================
// Struct Tests
// ============================================

func TestPodActionItem_Struct(t *testing.T) {
	item := PodActionItem{
		Label:       "Delete",
		Description: "removes pod",
		Action:      "delete",
		Command:     "kubectl delete pod test",
	}

	if item.Label != "Delete" {
		t.Errorf("Label = %q, want %q", item.Label, "Delete")
	}
	if item.Action != "delete" {
		t.Errorf("Action = %q, want %q", item.Action, "delete")
	}
}

func TestWorkloadActionItem_Struct(t *testing.T) {
	item := WorkloadActionItem{
		Label:    "Scale to 5",
		Action:   "scale",
		Replicas: 5,
		Command:  "kubectl scale",
	}

	if item.Label != "Scale to 5" {
		t.Errorf("Label = %q, want %q", item.Label, "Scale to 5")
	}
	if item.Replicas != 5 {
		t.Errorf("Replicas = %d, want %d", item.Replicas, 5)
	}
}

func TestPodActionMenuResult_Struct(t *testing.T) {
	result := PodActionMenuResult{
		Item: PodActionItem{
			Label:  "Delete",
			Action: "delete",
		},
	}

	if result.Item.Action != "delete" {
		t.Errorf("Item.Action = %q, want %q", result.Item.Action, "delete")
	}
}

func TestWorkloadActionMenuResult_Struct(t *testing.T) {
	result := WorkloadActionMenuResult{
		Item: WorkloadActionItem{
			Label:    "Restart",
			Action:   "restart",
			Replicas: 0,
		},
	}

	if result.Item.Action != "restart" {
		t.Errorf("Item.Action = %q, want %q", result.Item.Action, "restart")
	}
}
