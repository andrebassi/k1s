package k8s

import (
	"testing"
)

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max with ellipsis",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "very short max length",
			input:    "hello",
			maxLen:   2,
			expected: "he",
		},
		{
			name:     "max length 3 (edge case for ellipsis)",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestFormatLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		contains []string // Check contains since map iteration order is random
		isEmpty  bool
	}{
		{
			name:    "empty labels",
			labels:  map[string]string{},
			isEmpty: true,
		},
		{
			name:    "nil labels",
			labels:  nil,
			isEmpty: true,
		},
		{
			name:     "single label",
			labels:   map[string]string{"app": "nginx"},
			contains: []string{"app=nginx"},
		},
		{
			name: "more than three labels truncates",
			labels: map[string]string{
				"app":     "nginx",
				"env":     "prod",
				"version": "v1",
				"team":    "platform",
				"region":  "us-west",
			},
			contains: []string{"+2 more"}, // Should show truncation indicator
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLabels(tt.labels)

			if tt.isEmpty {
				if result != "<none>" {
					t.Errorf("FormatLabels(%v) = %q, want %q", tt.labels, result, "<none>")
				}
				return
			}

			for _, want := range tt.contains {
				if !containsSubstring(result, want) {
					t.Errorf("FormatLabels(%v) = %q, should contain %q", tt.labels, result, want)
				}
			}
		})
	}
}

func TestAnalyzePodIssues(t *testing.T) {
	tests := []struct {
		name           string
		pod            *PodInfo
		events         []EventInfo
		expectIssues   []string
		expectSeverity map[string]string
	}{
		{
			name: "CrashLoopBackOff status",
			pod: &PodInfo{
				Status:     "CrashLoopBackOff",
				Containers: []ContainerInfo{},
			},
			events:       []EventInfo{},
			expectIssues: []string{"CrashLoopBackOff"},
			expectSeverity: map[string]string{
				"CrashLoopBackOff": "High",
			},
		},
		{
			name: "ImagePullBackOff status",
			pod: &PodInfo{
				Status:     "ImagePullBackOff",
				Containers: []ContainerInfo{},
			},
			events:       []EventInfo{},
			expectIssues: []string{"Image Pull Failed"},
			expectSeverity: map[string]string{
				"Image Pull Failed": "High",
			},
		},
		{
			name: "ErrImagePull status",
			pod: &PodInfo{
				Status:     "ErrImagePull",
				Containers: []ContainerInfo{},
			},
			events:       []EventInfo{},
			expectIssues: []string{"Image Pull Failed"},
		},
		{
			name: "Pending status",
			pod: &PodInfo{
				Status:     "Pending",
				Containers: []ContainerInfo{},
			},
			events:       []EventInfo{},
			expectIssues: []string{"Pod Pending"},
			expectSeverity: map[string]string{
				"Pod Pending": "Medium",
			},
		},
		{
			name: "OOMKilled status",
			pod: &PodInfo{
				Status:     "OOMKilled",
				Containers: []ContainerInfo{},
			},
			events:       []EventInfo{},
			expectIssues: []string{"Out of Memory"},
			expectSeverity: map[string]string{
				"Out of Memory": "High",
			},
		},
		{
			name: "container without memory limit",
			pod: &PodInfo{
				Status: "Running",
				Containers: []ContainerInfo{
					{
						Name: "app",
						Resources: ResourceRequirements{
							MemoryLimit: "",
							CPULimit:    "100m",
						},
					},
				},
			},
			events:       []EventInfo{},
			expectIssues: []string{"No memory limit on container app"},
		},
		{
			name: "container without CPU limit",
			pod: &PodInfo{
				Status: "Running",
				Containers: []ContainerInfo{
					{
						Name: "app",
						Resources: ResourceRequirements{
							MemoryLimit: "128Mi",
							CPULimit:    "",
						},
					},
				},
			},
			events:       []EventInfo{},
			expectIssues: []string{"No CPU limit on container app"},
		},
		{
			name: "FailedScheduling event",
			pod: &PodInfo{
				Status:     "Pending",
				Containers: []ContainerInfo{},
			},
			events: []EventInfo{
				{
					Type:    "Warning",
					Reason:  "FailedScheduling",
					Message: "0/3 nodes available: insufficient memory",
				},
			},
			expectIssues: []string{"Pod Pending", "Scheduling Failed"},
		},
		{
			name: "healthy pod no issues",
			pod: &PodInfo{
				Status: "Running",
				Containers: []ContainerInfo{
					{
						Name: "app",
						Resources: ResourceRequirements{
							MemoryLimit: "128Mi",
							CPULimit:    "100m",
						},
					},
				},
			},
			events:       []EventInfo{},
			expectIssues: []string{}, // No issues expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helpers := AnalyzePodIssues(tt.pod, tt.events)

			// Check expected issues are present
			for _, expectedIssue := range tt.expectIssues {
				found := false
				for _, h := range helpers {
					if h.Issue == expectedIssue {
						found = true
						// Check severity if specified
						if expectedSev, ok := tt.expectSeverity[expectedIssue]; ok {
							if h.Severity != expectedSev {
								t.Errorf("Issue %q has severity %q, want %q", expectedIssue, h.Severity, expectedSev)
							}
						}
						// Verify suggestions exist
						if len(h.Suggestions) == 0 {
							t.Errorf("Issue %q has no suggestions", expectedIssue)
						}
						break
					}
				}
				if !found {
					t.Errorf("Expected issue %q not found in helpers", expectedIssue)
				}
			}

			// If we expect no issues, verify that's the case
			if len(tt.expectIssues) == 0 && len(helpers) > 0 {
				var issues []string
				for _, h := range helpers {
					issues = append(issues, h.Issue)
				}
				t.Errorf("Expected no issues but got: %v", issues)
			}
		})
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && contains(s, substr)))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
