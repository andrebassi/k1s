package k8s

import (
	"fmt"
	"time"
)

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

func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

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

type DebugHelper struct {
	Issue       string
	Severity    string
	Suggestions []string
}

func AnalyzePodIssues(pod *PodInfo, events []EventInfo) []DebugHelper {
	var helpers []DebugHelper

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
