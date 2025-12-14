package repository

import (
	"strings"
	"testing"
	"time"
)

func TestDefaultLogOptions(t *testing.T) {
	opts := DefaultLogOptions()

	if opts.TailLines != 100 {
		t.Errorf("DefaultLogOptions().TailLines = %d, want 100", opts.TailLines)
	}
	if !opts.Timestamps {
		t.Error("DefaultLogOptions().Timestamps should be true")
	}
	if opts.Previous {
		t.Error("DefaultLogOptions().Previous should be false")
	}
	if opts.Follow {
		t.Error("DefaultLogOptions().Follow should be false")
	}
}

func TestIsErrorLine(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"contains error", "Something went error here", true},
		{"contains Error (case insensitive)", "Error occurred", true},
		{"contains ERROR (uppercase)", "ERROR: something failed", true},
		{"contains err:", "err: connection refused", true},
		{"contains fatal", "Fatal exception in application", true},
		{"contains panic", "panic: runtime error", true},
		{"contains exception", "Unhandled exception", true},
		{"contains failed", "Request failed", true},
		{"contains failure", "Connection failure", true},
		{"contains crash", "Application crash detected", true},
		{"contains critical", "Critical error", true},
		{"normal info log", "INFO: Processing request", false},
		{"normal debug log", "DEBUG: Variable value = 42", false},
		{"empty string", "", false},
		{"just whitespace", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isErrorLine(tt.content)
			if result != tt.expected {
				t.Errorf("isErrorLine(%q) = %v, want %v", tt.content, result, tt.expected)
			}
		})
	}
}

func TestSearchLogs(t *testing.T) {
	logs := []LogLine{
		{Content: "Starting application", Container: "app"},
		{Content: "Processing request for user 123", Container: "app"},
		{Content: "Error connecting to database", Container: "app"},
		{Content: "Request completed successfully", Container: "app"},
		{Content: "User 456 logged in", Container: "app"},
	}

	tests := []struct {
		name           string
		query          string
		expectedCount  int
		shouldContain  []string
	}{
		{"empty query returns all", "", 5, nil},
		{"find user", "user", 2, []string{"user 123", "User 456"}},
		{"case insensitive", "USER", 2, nil},
		{"find error", "error", 1, []string{"Error connecting"}},
		{"no match", "xyz123", 0, nil},
		{"partial match", "request", 2, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchLogs(logs, tt.query)
			if len(result) != tt.expectedCount {
				t.Errorf("SearchLogs() returned %d logs, want %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestFilterErrorLogs(t *testing.T) {
	logs := []LogLine{
		{Content: "Normal log", IsError: false},
		{Content: "Error log", IsError: true},
		{Content: "Another normal", IsError: false},
		{Content: "Fatal error", IsError: true},
		{Content: "Info message", IsError: false},
	}

	result := FilterErrorLogs(logs)
	if len(result) != 2 {
		t.Errorf("FilterErrorLogs() returned %d logs, want 2", len(result))
	}

	for _, log := range result {
		if !log.IsError {
			t.Errorf("FilterErrorLogs() returned a non-error log: %q", log.Content)
		}
	}
}

func TestFilterErrorLogs_Empty(t *testing.T) {
	logs := []LogLine{
		{Content: "Normal log", IsError: false},
		{Content: "Another normal", IsError: false},
	}

	result := FilterErrorLogs(logs)
	if len(result) != 0 {
		t.Errorf("FilterErrorLogs() should return empty slice when no errors, got %d", len(result))
	}
}

func TestGetLogsAroundTime(t *testing.T) {
	now := time.Now()
	logs := []LogLine{
		{Content: "Log 1", Timestamp: now.Add(-60 * time.Minute)},
		{Content: "Log 2", Timestamp: now.Add(-10 * time.Minute)},
		{Content: "Log 3", Timestamp: now.Add(-4 * time.Minute)},  // Within 5 min window
		{Content: "Log 4", Timestamp: now.Add(4 * time.Minute)},   // Within 5 min window
		{Content: "Log 5", Timestamp: now.Add(10 * time.Minute)},
		{Content: "Log 6", Timestamp: now.Add(60 * time.Minute)},
		{Content: "No timestamp", Timestamp: time.Time{}},
	}

	tests := []struct {
		name          string
		target        time.Time
		windowMinutes int
		expectedCount int
	}{
		{"15 minute window around now", now, 15, 4}, // Log 2,3,4,5
		{"5 minute window around now", now, 5, 2},   // Log 3,4 (within ±5 min)
		{"1 minute window around now", now, 1, 0},   // None exactly at now
		{"very large window", now, 120, 6},          // All except no-timestamp
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLogsAroundTime(logs, tt.target, tt.windowMinutes)
			if len(result) != tt.expectedCount {
				t.Errorf("GetLogsAroundTime() returned %d logs, want %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestSortLogsByTime(t *testing.T) {
	now := time.Now()
	logs := []LogLine{
		{Content: "Third", Timestamp: now.Add(10 * time.Minute)},
		{Content: "First", Timestamp: now.Add(-10 * time.Minute)},
		{Content: "Second", Timestamp: now},
	}

	sortLogsByTime(logs)

	if logs[0].Content != "First" {
		t.Errorf("After sort, first log should be 'First', got %q", logs[0].Content)
	}
	if logs[1].Content != "Second" {
		t.Errorf("After sort, second log should be 'Second', got %q", logs[1].Content)
	}
	if logs[2].Content != "Third" {
		t.Errorf("After sort, third log should be 'Third', got %q", logs[2].Content)
	}
}

func TestSortLogsByTime_Empty(t *testing.T) {
	var logs []LogLine
	// Should not panic
	sortLogsByTime(logs)
	if len(logs) != 0 {
		t.Error("Empty slice should remain empty after sort")
	}
}

func TestSortLogsByTime_Single(t *testing.T) {
	logs := []LogLine{{Content: "Only one"}}
	// Should not panic
	sortLogsByTime(logs)
	if len(logs) != 1 {
		t.Error("Single element slice should remain unchanged after sort")
	}
}

func TestParseLogStream(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		container     string
		hasTimestamps bool
		expectedCount int
		checkFirst    func(LogLine) bool
	}{
		{
			name:          "simple log without timestamps",
			input:         "line one\nline two\nline three",
			container:     "app",
			hasTimestamps: false,
			expectedCount: 3,
			checkFirst: func(l LogLine) bool {
				return l.Content == "line one" && l.Container == "app"
			},
		},
		{
			name:          "log with RFC3339Nano timestamps",
			input:         "2024-01-15T10:30:45.123456789Z INFO: Starting\n2024-01-15T10:30:46.123456789Z DEBUG: Processing",
			container:     "web",
			hasTimestamps: true,
			expectedCount: 2,
			checkFirst: func(l LogLine) bool {
				return strings.Contains(l.Content, "INFO") && l.Container == "web" && !l.Timestamp.IsZero()
			},
		},
		{
			name:          "empty input",
			input:         "",
			container:     "test",
			hasTimestamps: false,
			expectedCount: 0,
			checkFirst:    nil,
		},
		{
			name:          "error detection",
			input:         "Normal log\nError occurred here\nAnother normal",
			container:     "app",
			hasTimestamps: false,
			expectedCount: 3,
			checkFirst:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			logs, err := parseLogStream(reader, tt.container, tt.hasTimestamps)
			if err != nil {
				t.Fatalf("parseLogStream() error = %v", err)
			}
			if len(logs) != tt.expectedCount {
				t.Errorf("parseLogStream() returned %d logs, want %d", len(logs), tt.expectedCount)
			}
			if tt.checkFirst != nil && len(logs) > 0 && !tt.checkFirst(logs[0]) {
				t.Errorf("First log line check failed: %+v", logs[0])
			}
		})
	}
}

func TestParseLogStream_ErrorDetection(t *testing.T) {
	input := "Normal line\nError: something failed\nAnother normal"
	reader := strings.NewReader(input)
	logs, _ := parseLogStream(reader, "app", false)

	errorCount := 0
	for _, l := range logs {
		if l.IsError {
			errorCount++
		}
	}

	if errorCount != 1 {
		t.Errorf("Expected 1 error line, got %d", errorCount)
	}
}

func TestParseLogStream_RFC3339Timestamp(t *testing.T) {
	// Test RFC3339 (without nanoseconds) - exactly 20 chars timestamp
	input := "2024-01-15T10:30:45Z INFO: Starting app"
	reader := strings.NewReader(input)
	logs, err := parseLogStream(reader, "app", true)
	if err != nil {
		t.Fatalf("parseLogStream() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}
	if logs[0].Timestamp.IsZero() {
		t.Error("Timestamp should be parsed for RFC3339 format")
	}
}

func TestParseLogStream_ShortLine(t *testing.T) {
	// Test short lines (< 30 chars) with timestamps enabled - should not crash
	input := "Short line"
	reader := strings.NewReader(input)
	logs, err := parseLogStream(reader, "app", true)
	if err != nil {
		t.Fatalf("parseLogStream() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}
	if logs[0].Content != "Short line" {
		t.Errorf("Content = %q, want 'Short line'", logs[0].Content)
	}
}

func TestParseLogStream_InvalidTimestamp(t *testing.T) {
	// Test invalid timestamp format - should keep original content
	input := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxx Not a timestamp"
	reader := strings.NewReader(input)
	logs, err := parseLogStream(reader, "app", true)
	if err != nil {
		t.Fatalf("parseLogStream() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log, got %d", len(logs))
	}
	if !strings.Contains(logs[0].Content, "Not a timestamp") {
		t.Errorf("Content should contain original text")
	}
}

func TestLogLineStruct(t *testing.T) {
	now := time.Now()
	line := LogLine{
		Timestamp: now,
		Container: "web",
		Content:   "Test content",
		IsError:   true,
	}

	if line.Timestamp != now {
		t.Errorf("Timestamp mismatch")
	}
	if line.Container != "web" {
		t.Errorf("Container = %q, want 'web'", line.Container)
	}
	if line.Content != "Test content" {
		t.Errorf("Content = %q, want 'Test content'", line.Content)
	}
	if !line.IsError {
		t.Error("IsError should be true")
	}
}

func TestLogOptionsStruct(t *testing.T) {
	opts := LogOptions{
		Container:  "app",
		TailLines:  500,
		Since:      5 * time.Minute,
		Previous:   true,
		Follow:     true,
		Timestamps: true,
	}

	if opts.Container != "app" {
		t.Errorf("Container = %q, want 'app'", opts.Container)
	}
	if opts.TailLines != 500 {
		t.Errorf("TailLines = %d, want 500", opts.TailLines)
	}
	if opts.Since != 5*time.Minute {
		t.Errorf("Since = %v, want 5m", opts.Since)
	}
	if !opts.Previous {
		t.Error("Previous should be true")
	}
	if !opts.Follow {
		t.Error("Follow should be true")
	}
	if !opts.Timestamps {
		t.Error("Timestamps should be true")
	}
}
