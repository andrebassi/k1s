package k8s

import (
	"fmt"
	"time"
)

// formatAge converts a timestamp to a human-readable age string.
// Outputs formats like "45s", "5m", "2h", "3d" depending on the duration.
func formatAge(t time.Time) string {
	if t.IsZero() {
		return "Unknown"
	}

	d := time.Since(t)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d"
		}
		return fmt.Sprintf("%dd", days)
	}
}

// TruncateString shortens a string to maxLen characters, adding "..." if truncated.
// If maxLen is 3 or less, no ellipsis is added to preserve the limited space.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// FormatLabels converts a label map to a human-readable string.
// Shows up to 3 labels with a "(+N more)" suffix if there are more.
// Returns "<none>" if the map is empty.
func FormatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "<none>"
	}

	result := ""
	i := 0
	for k, v := range labels {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s=%s", k, v)
		i++
		if i >= 3 {
			remaining := len(labels) - 3
			if remaining > 0 {
				result += fmt.Sprintf(" (+%d more)", remaining)
			}
			break
		}
	}
	return result
}

// DebugHelper provides diagnostic information about a pod issue.
// Used to display helpful suggestions in the UI when problems are detected.
type DebugHelper struct {
	Issue       string   // Brief description of the problem
	Severity    string   // "High", "Medium", "Warning", or "Info"
	Suggestions []string // Actionable steps to investigate or resolve
}

// AnalyzePodIssues examines pod state and events to identify common problems.
// Returns a list of DebugHelper structs with diagnostic information.
// Detects issues like CrashLoopBackOff, ImagePullBackOff, Pending pods,
// OOMKilled containers, and missing resource limits.
func AnalyzePodIssues(pod *PodInfo, events []EventInfo) []DebugHelper {
	var helpers []DebugHelper

	// Check pod status for common problems
	switch pod.Status {
	case "CrashLoopBackOff":
		helpers = append(helpers, DebugHelper{
			Issue:    "CrashLoopBackOff",
			Severity: "High",
			Suggestions: []string{
				"Check container logs for crash reason",
				"Verify resource limits aren't too restrictive",
				"Check liveness probe configuration",
				"Look for application startup errors",
			},
		})

	case "ImagePullBackOff", "ErrImagePull":
		helpers = append(helpers, DebugHelper{
			Issue:    "Image Pull Failed",
			Severity: "High",
			Suggestions: []string{
				"Verify image name and tag are correct",
				"Check image registry credentials",
				"Ensure node has network access to registry",
				"Verify image exists in the registry",
			},
		})

	case "Pending":
		helpers = append(helpers, DebugHelper{
			Issue:    "Pod Pending",
			Severity: "Medium",
			Suggestions: []string{
				"Check scheduler events for scheduling failures",
				"Verify node resources are available",
				"Check node selectors and tolerations",
				"Review resource requests against available capacity",
			},
		})

	case "OOMKilled":
		helpers = append(helpers, DebugHelper{
			Issue:    "Out of Memory",
			Severity: "High",
			Suggestions: []string{
				"Increase memory limits for the container",
				"Check for memory leaks in application",
				"Review memory usage patterns in metrics",
				"Consider horizontal scaling instead",
			},
		})
	}

	// Check for missing resource limits (best practice warnings)
	for _, c := range pod.Containers {
		if c.Resources.MemoryLimit == "0" || c.Resources.MemoryLimit == "" {
			helpers = append(helpers, DebugHelper{
				Issue:    fmt.Sprintf("No memory limit on container %s", c.Name),
				Severity: "Warning",
				Suggestions: []string{
					"Set memory limits to prevent OOM issues",
					"Memory limits help with resource planning",
				},
			})
		}
		if c.Resources.CPULimit == "0" || c.Resources.CPULimit == "" {
			helpers = append(helpers, DebugHelper{
				Issue:    fmt.Sprintf("No CPU limit on container %s", c.Name),
				Severity: "Info",
				Suggestions: []string{
					"Consider setting CPU limits for predictable performance",
				},
			})
		}
	}

	// Check events for scheduling failures
	for _, e := range events {
		if e.Type == "Warning" && e.Reason == "FailedScheduling" {
			helpers = append(helpers, DebugHelper{
				Issue:    "Scheduling Failed",
				Severity: "High",
				Suggestions: []string{
					e.Message,
					"Check node resources and selectors",
				},
			})
		}
	}

	return helpers
}
