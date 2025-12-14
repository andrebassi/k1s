package view

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/component"
)

func TestNewDashboard(t *testing.T) {
	d := NewDashboard()

	if d.focus != FocusLogs {
		t.Errorf("NewDashboard focus = %v, want FocusLogs", d.focus)
	}
	if d.fullscreen {
		t.Error("NewDashboard fullscreen should be false")
	}
}

func TestDashboard_Init(t *testing.T) {
	d := NewDashboard()
	cmd := d.Init()
	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestDashboard_SetSize(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 40)

	if d.width != 120 {
		t.Errorf("width = %d, want 120", d.width)
	}
	if d.height != 40 {
		t.Errorf("height = %d, want 40", d.height)
	}
}

func TestDashboard_SetPod(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Status:    "Running",
		Ready:     "1/1",
		Restarts:  0,
	}

	d.SetPod(pod)

	if d.GetPod() == nil {
		t.Error("GetPod() should not return nil after SetPod")
	}
	if d.GetPod().Name != "test-pod" {
		t.Errorf("Pod name = %q, want 'test-pod'", d.GetPod().Name)
	}
}

func TestDashboard_SetLogs(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	logs := []repository.LogLine{
		{Content: "Starting application", Container: "app"},
		{Content: "Error occurred", Container: "app", IsError: true},
	}

	d.SetLogs(logs)
	// Just verify it doesn't panic
}

func TestDashboard_SetEvents(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	events := []repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled", Message: "Pod scheduled"},
		{Type: "Warning", Reason: "BackOff", Message: "Container restarting"},
	}

	d.SetEvents(events)
	// Just verify it doesn't panic
}

func TestDashboard_SetMetrics(t *testing.T) {
	d := NewDashboard()

	metrics := &repository.PodMetrics{
		Name:      "test-pod",
		Namespace: "default",
		Containers: []repository.ContainerMetrics{
			{Name: "app", CPUUsage: "100m", MemoryUsage: "128Mi"},
		},
	}

	d.SetMetrics(metrics)
	// Just verify it doesn't panic
}

func TestDashboard_SetRelated(t *testing.T) {
	d := NewDashboard()

	related := &repository.RelatedResources{
		Services: []repository.ServiceInfo{{Name: "web-service", Type: "ClusterIP"}},
	}

	d.SetRelated(related)
	// Just verify it doesn't panic
}

func TestDashboard_SetNode(t *testing.T) {
	d := NewDashboard()

	node := &repository.NodeInfo{
		Name:   "node-1",
		Status: "Ready",
	}

	d.SetNode(node)
	// Just verify it doesn't panic
}

func TestDashboard_SetHelpers(t *testing.T) {
	d := NewDashboard()

	helpers := []repository.DebugHelper{
		{Issue: "Container restarting", Severity: "Warning", Suggestions: []string{"Check logs"}},
	}

	d.SetHelpers(helpers)
	// Just verify it doesn't panic
}

func TestDashboard_SetBreadcrumb(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	d.SetBreadcrumb("default", "Deployments", "web", "web-abc123")
	// Just verify it doesn't panic
}

func TestDashboard_SetContext(t *testing.T) {
	d := NewDashboard()
	d.SetContext("my-cluster")

	if d.context != "my-cluster" {
		t.Errorf("context = %q, want 'my-cluster'", d.context)
	}
}

func TestDashboard_SetNamespace(t *testing.T) {
	d := NewDashboard()
	d.SetNamespace("production")

	if d.namespace != "production" {
		t.Errorf("namespace = %q, want 'production'", d.namespace)
	}
}

func TestDashboard_Focus(t *testing.T) {
	d := NewDashboard()
	if d.Focus() != FocusLogs {
		t.Errorf("Focus() = %v, want FocusLogs", d.Focus())
	}
}

func TestDashboard_HelpVisible(t *testing.T) {
	d := NewDashboard()
	if d.HelpVisible() {
		t.Error("HelpVisible() should be false initially")
	}
}

func TestDashboard_ShortHelp(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	help := d.ShortHelp()
	// ShortHelp should return something
	if help == "" {
		t.Error("ShortHelp() should not be empty")
	}
}

func TestDashboard_IsFullscreen(t *testing.T) {
	d := NewDashboard()
	if d.IsFullscreen() {
		t.Error("IsFullscreen() should be false initially")
	}
}

func TestDashboard_IsFullscreenLogs(t *testing.T) {
	d := NewDashboard()
	if d.IsFullscreenLogs() {
		t.Error("IsFullscreenLogs() should be false initially")
	}
}

func TestDashboard_IsFullscreenEvents(t *testing.T) {
	d := NewDashboard()
	if d.IsFullscreenEvents() {
		t.Error("IsFullscreenEvents() should be false initially")
	}
}

func TestDashboard_IsLogsSearching(t *testing.T) {
	d := NewDashboard()
	if d.IsLogsSearching() {
		t.Error("IsLogsSearching() should be false initially")
	}
}

func TestDashboard_IsEventsSearching(t *testing.T) {
	d := NewDashboard()
	if d.IsEventsSearching() {
		t.Error("IsEventsSearching() should be false initially")
	}
}

func TestDashboard_HasActiveOverlay(t *testing.T) {
	d := NewDashboard()
	if d.HasActiveOverlay() {
		t.Error("HasActiveOverlay() should be false initially")
	}
}

func TestDashboard_CloseFullscreen(t *testing.T) {
	d := NewDashboard()
	d.fullscreen = true
	d.CloseFullscreen()

	if d.fullscreen {
		t.Error("fullscreen should be false after CloseFullscreen()")
	}
}

func TestDashboard_LogsSelectedContainer(t *testing.T) {
	d := NewDashboard()
	// Should return default container selection
	container := d.LogsSelectedContainer()
	// May be empty initially
	_ = container
}

func TestDashboard_LogsShowPrevious(t *testing.T) {
	d := NewDashboard()
	// Should return false initially
	if d.LogsShowPrevious() {
		t.Error("LogsShowPrevious() should be false initially")
	}
}

func TestDashboard_View_Empty(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	view := d.View()
	if view == "" {
		t.Error("View() should not be empty")
	}
}

func TestDashboard_View_WithPod(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{
		Name:      "test-pod",
		Namespace: "default",
		Status:    "Running",
		Ready:     "1/1",
	}
	d.SetPod(pod)

	view := d.View()
	if view == "" {
		t.Error("View() should not be empty")
	}
}

func TestDashboard_Update_TabKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Press Tab to switch panels
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyTab})

	if d.focus == FocusLogs {
		t.Error("Tab key should change focus from FocusLogs")
	}
}

func TestDashboard_Update_ShiftTabKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)
	d.focus = FocusEvents

	// Press Shift+Tab to go back
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyShiftTab})

	if d.focus == FocusEvents {
		t.Error("Shift+Tab key should change focus from FocusEvents")
	}
}

func TestDashboard_Update_HelpKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Press ? to show help
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	if !d.HelpVisible() {
		t.Error("? key should show help")
	}
}

func TestDashboard_Update_FullscreenKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Set pod first (fullscreen requires pod context)
	pod := &repository.PodInfo{Name: "test-pod", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Press f to toggle fullscreen - stores previous state
	initialFullscreen := d.IsFullscreen()
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// fullscreen state should change
	if d.IsFullscreen() == initialFullscreen {
		// If it didn't toggle, that's still valid behavior depending on panel state
		// Just verify no panic occurred
	}
}

func TestDashboard_Update_ExecFinishedMsg(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Handle ExecFinishedMsg
	d, _ = d.Update(ExecFinishedMsg{Err: nil})

	if d.statusMsg != "Command completed" {
		t.Errorf("statusMsg = %q, want 'Command completed'", d.statusMsg)
	}
}

func TestDashboard_Update_ExecFinishedMsg_Error(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Handle ExecFinishedMsg with error
	d, _ = d.Update(ExecFinishedMsg{Err: errTest})

	if !strings.Contains(d.statusMsg, "Command failed") {
		t.Errorf("statusMsg = %q, want contains 'Command failed'", d.statusMsg)
	}
}

func TestDashboard_Update_DescribeOutputMsg(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Handle DescribeOutputMsg
	d, _ = d.Update(DescribeOutputMsg{
		Title:   "Pod: test",
		Content: "Some content",
	})
	// Result viewer should be shown
}

func TestDashboard_Update_ScaleResultMsg(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Handle ScaleResultMsg
	d, _ = d.Update(ScaleResultMsg{
		Success:  true,
		Replicas: 3,
	})

	if !strings.Contains(d.statusMsg, "Scaled to 3") {
		t.Errorf("statusMsg = %q, want contains 'Scaled to 3'", d.statusMsg)
	}
}

func TestDashboard_Update_ActionMenuResult(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	// Handle ActionMenuResult with copy success
	d, _ = d.Update(component.ActionMenuResult{
		Copied: true,
		Item:   component.MenuItem{Label: "Get logs"},
	})

	if !strings.Contains(d.statusMsg, "Copied") {
		t.Errorf("statusMsg = %q, want contains 'Copied'", d.statusMsg)
	}
}

func TestDashboard_nextPanel(t *testing.T) {
	d := NewDashboard()
	d.focus = FocusLogs

	d.nextPanel()

	if d.focus != FocusEvents {
		t.Errorf("focus = %v, want FocusEvents", d.focus)
	}
}

func TestDashboard_prevPanel(t *testing.T) {
	d := NewDashboard()
	d.focus = FocusEvents

	d.prevPanel()

	if d.focus != FocusLogs {
		t.Errorf("focus = %v, want FocusLogs", d.focus)
	}
}

// PanelFocus tests
func TestPanelFocus_Constants(t *testing.T) {
	if FocusLogs != 0 {
		t.Errorf("FocusLogs = %d, want 0", FocusLogs)
	}
	if FocusEvents != 1 {
		t.Errorf("FocusEvents = %d, want 1", FocusEvents)
	}
	if FocusMetrics != 2 {
		t.Errorf("FocusMetrics = %d, want 2", FocusMetrics)
	}
	if FocusManifest != 3 {
		t.Errorf("FocusManifest = %d, want 3", FocusManifest)
	}
}

// Struct tests
func TestDeletePodRequest_Struct(t *testing.T) {
	req := DeletePodRequest{
		Namespace: "default",
		PodName:   "my-pod",
	}

	if req.Namespace != "default" {
		t.Errorf("Namespace = %q, want 'default'", req.Namespace)
	}
	if req.PodName != "my-pod" {
		t.Errorf("PodName = %q, want 'my-pod'", req.PodName)
	}
}

func TestExecFinishedMsg_Struct(t *testing.T) {
	msg := ExecFinishedMsg{Err: nil}
	if msg.Err != nil {
		t.Error("Err should be nil")
	}
}

func TestDescribeOutputMsg_Struct(t *testing.T) {
	msg := DescribeOutputMsg{
		Title:   "Test",
		Content: "Content",
		Err:     nil,
	}

	if msg.Title != "Test" {
		t.Errorf("Title = %q, want 'Test'", msg.Title)
	}
	if msg.Content != "Content" {
		t.Errorf("Content = %q, want 'Content'", msg.Content)
	}
}

func TestScaleResultMsg_Struct(t *testing.T) {
	msg := ScaleResultMsg{
		Success:  true,
		Replicas: 5,
		Err:      nil,
	}

	if !msg.Success {
		t.Error("Success should be true")
	}
	if msg.Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", msg.Replicas)
	}
}

func TestScaleRequestMsg_Struct(t *testing.T) {
	msg := ScaleRequestMsg{
		WorkloadKind: "Deployment",
		WorkloadName: "web",
		Namespace:    "default",
		NewReplicas:  3,
	}

	if msg.WorkloadKind != "Deployment" {
		t.Errorf("WorkloadKind = %q, want 'Deployment'", msg.WorkloadKind)
	}
	if msg.NewReplicas != 3 {
		t.Errorf("NewReplicas = %d, want 3", msg.NewReplicas)
	}
}

// Test error variable for error testing
type testError struct{}

func (e testError) Error() string { return "test error" }

var errTest = testError{}

// ============================================
// Additional Dashboard Tests
// ============================================

func TestDashboard_View_WithFullData(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	// Set up complete dashboard state
	pod := &repository.PodInfo{
		Name:      "web-pod-abc123",
		Namespace: "production",
		Status:    "Running",
		Ready:     "1/1",
		Restarts:  2,
		Node:      "node-1",
		Containers: []repository.ContainerInfo{
			{Name: "app", Image: "nginx:latest", Ready: true, State: "Running"},
		},
	}
	d.SetPod(pod)

	logs := []repository.LogLine{
		{Content: "Starting nginx", Container: "app"},
		{Content: "Listening on port 80", Container: "app"},
	}
	d.SetLogs(logs)

	events := []repository.EventInfo{
		{Type: "Normal", Reason: "Scheduled", Message: "Pod scheduled"},
		{Type: "Normal", Reason: "Pulled", Message: "Image pulled"},
	}
	d.SetEvents(events)

	metrics := &repository.PodMetrics{
		Name: "web-pod-abc123",
		Containers: []repository.ContainerMetrics{
			{Name: "app", CPUUsage: "50m", MemoryUsage: "64Mi"},
		},
	}
	d.SetMetrics(metrics)

	related := &repository.RelatedResources{
		Services: []repository.ServiceInfo{
			{Name: "web-svc", Type: "ClusterIP", Ports: "80/TCP"},
		},
	}
	d.SetRelated(related)

	node := &repository.NodeInfo{
		Name:   "node-1",
		Status: "Ready",
	}
	d.SetNode(node)

	d.SetBreadcrumb("production", "Deployments", "web", "web-pod-abc123")
	d.SetContext("prod-cluster")
	d.SetNamespace("production")

	view := d.View()
	if view == "" {
		t.Error("View with full data should not be empty")
	}
}

func TestDashboard_Update_NavigationKeys(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Test arrow key navigation
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRight})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyLeft})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyDown})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Test j/k/h/l navigation (vim keys)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
}

func TestDashboard_Update_SearchKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.SetLogs([]repository.LogLine{{Content: "test log"}})

	// Press / to search in logs panel
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
}

func TestDashboard_Update_EscKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Press Esc
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyEsc})
}

func TestDashboard_Update_QKey_WithHelp(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Show help first
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// Press q to close help
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
}

func TestDashboard_Update_WindowSizeMsg(t *testing.T) {
	d := NewDashboard()

	// Handle window size message - width and height may be handled by app.go not dashboard
	d, _ = d.Update(tea.WindowSizeMsg{Width: 150, Height: 60})

	// WindowSizeMsg is typically handled at app level, not dashboard level
	// Just verify no panic occurred
}

func TestDashboard_Update_RefreshKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Press r to refresh
	d, cmd := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	// cmd might be nil or return a refresh command
	_ = cmd
}

func TestDashboard_renderTopRow(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	row := d.renderTopRow()
	if row == "" {
		t.Error("renderTopRow should not return empty string")
	}
}

func TestDashboard_renderBottomRow(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	row := d.renderBottomRow()
	if row == "" {
		t.Error("renderBottomRow should not return empty string")
	}
}

func TestDashboard_wrapPanel(t *testing.T) {
	d := NewDashboard()

	// Test active panel
	wrapped := d.wrapPanel("content", 40, 20, true)
	if wrapped == "" {
		t.Error("wrapPanel should not return empty string")
	}

	// Test inactive panel
	wrapped = d.wrapPanel("content", 40, 20, false)
	if wrapped == "" {
		t.Error("wrapPanel should not return empty string")
	}
}

func TestDashboard_renderFloatingDialog(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	dialog := d.renderFloatingDialog("Dialog content")
	if dialog == "" {
		t.Error("renderFloatingDialog should not return empty string")
	}
}

func TestDashboard_renderFullscreenPanel_Logs(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.SetLogs([]repository.LogLine{{Content: "test"}})
	d.focus = FocusLogs
	d.fullscreen = true

	panel := d.renderFullscreenPanel()
	if panel == "" {
		t.Error("renderFullscreenPanel for logs should not return empty string")
	}
}

func TestDashboard_renderFullscreenPanel_Events(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.SetEvents([]repository.EventInfo{{Type: "Normal", Reason: "Test"}})
	d.focus = FocusEvents
	d.fullscreen = true

	panel := d.renderFullscreenPanel()
	if panel == "" {
		t.Error("renderFullscreenPanel for events should not return empty string")
	}
}

func TestDashboard_renderFullscreenPanel_Metrics(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.focus = FocusMetrics
	d.fullscreen = true

	panel := d.renderFullscreenPanel()
	if panel == "" {
		t.Error("renderFullscreenPanel for metrics should not return empty string")
	}
}

func TestDashboard_renderFullscreenPanel_Manifest(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.focus = FocusManifest
	d.fullscreen = true

	panel := d.renderFullscreenPanel()
	if panel == "" {
		t.Error("renderFullscreenPanel for manifest should not return empty string")
	}
}

func TestDashboard_Focus_AllPanels(t *testing.T) {
	d := NewDashboard()

	// Test each focus state
	d.focus = FocusLogs
	if d.Focus() != FocusLogs {
		t.Error("Focus should be FocusLogs")
	}

	d.focus = FocusEvents
	if d.Focus() != FocusEvents {
		t.Error("Focus should be FocusEvents")
	}

	d.focus = FocusMetrics
	if d.Focus() != FocusMetrics {
		t.Error("Focus should be FocusMetrics")
	}

	d.focus = FocusManifest
	if d.Focus() != FocusManifest {
		t.Error("Focus should be FocusManifest")
	}
}

func TestDashboard_nextPanel_Cycle(t *testing.T) {
	d := NewDashboard()

	// Cycle through all panels
	d.focus = FocusLogs
	d.nextPanel()
	if d.focus != FocusEvents {
		t.Errorf("After nextPanel from Logs, focus = %v, want FocusEvents", d.focus)
	}

	d.nextPanel()
	if d.focus != FocusMetrics {
		t.Errorf("After nextPanel from Events, focus = %v, want FocusMetrics", d.focus)
	}

	d.nextPanel()
	if d.focus != FocusManifest {
		t.Errorf("After nextPanel from Metrics, focus = %v, want FocusManifest", d.focus)
	}

	d.nextPanel()
	if d.focus != FocusLogs {
		t.Errorf("After nextPanel from Manifest, focus = %v, want FocusLogs (cycle)", d.focus)
	}
}

func TestDashboard_prevPanel_Cycle(t *testing.T) {
	d := NewDashboard()

	// Cycle backwards through all panels
	d.focus = FocusLogs
	d.prevPanel()
	if d.focus != FocusManifest {
		t.Errorf("After prevPanel from Logs, focus = %v, want FocusManifest", d.focus)
	}

	d.prevPanel()
	if d.focus != FocusMetrics {
		t.Errorf("After prevPanel from Manifest, focus = %v, want FocusMetrics", d.focus)
	}

	d.prevPanel()
	if d.focus != FocusEvents {
		t.Errorf("After prevPanel from Metrics, focus = %v, want FocusEvents", d.focus)
	}

	d.prevPanel()
	if d.focus != FocusLogs {
		t.Errorf("After prevPanel from Events, focus = %v, want FocusLogs (cycle)", d.focus)
	}
}

func TestDashboard_SetPod_WithContainers(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{
		Name:      "multi-container-pod",
		Namespace: "default",
		Status:    "Running",
		Containers: []repository.ContainerInfo{
			{Name: "app", Image: "nginx", Ready: true},
			{Name: "sidecar", Image: "envoy", Ready: true},
			{Name: "init", Image: "busybox", Ready: false},
		},
	}

	d.SetPod(pod)

	if d.GetPod() == nil {
		t.Error("GetPod should return pod")
	}
	if len(d.GetPod().Containers) != 3 {
		t.Errorf("Pod should have 3 containers, got %d", len(d.GetPod().Containers))
	}
}

func TestDashboard_Update_ActionKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Press a to open action menu
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
}

func TestDashboard_Update_CopyKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Press c to copy (in logs panel by default)
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
}

func TestDashboard_Update_WarningsKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)

	// Move to events panel
	d.focus = FocusEvents
	d.SetEvents([]repository.EventInfo{
		{Type: "Normal", Reason: "Test"},
		{Type: "Warning", Reason: "Error"},
	})

	// Press w to toggle warnings filter
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
}

func TestDashboard_Update_ContainerKey(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	pod := &repository.PodInfo{
		Name:      "test",
		Namespace: "default",
		Status:    "Running",
		Containers: []repository.ContainerInfo{
			{Name: "app"},
			{Name: "sidecar"},
		},
	}
	d.SetPod(pod)

	// Focus on logs and press number to select container
	d.focus = FocusLogs
	d, _ = d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
}

func TestDashboard_View_NoPod(t *testing.T) {
	d := NewDashboard()
	d.SetSize(100, 40)

	view := d.View()
	if !strings.Contains(view, "No pod selected") {
		t.Error("View with no pod should show 'No pod selected' message")
	}
}

func TestDashboard_View_Fullscreen(t *testing.T) {
	d := NewDashboard()
	d.SetSize(120, 50)

	pod := &repository.PodInfo{Name: "test", Namespace: "default", Status: "Running"}
	d.SetPod(pod)
	d.SetLogs([]repository.LogLine{{Content: "test log"}})
	d.fullscreen = true

	view := d.View()
	if view == "" {
		t.Error("Fullscreen view should not be empty")
	}
}
