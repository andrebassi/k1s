// Package usecase implements the application use cases.
// Use cases orchestrate the flow of data between the domain and adapters,
// containing the application-specific business rules.
package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/andrebassi/k1s/internal/domain/entity"
	"github.com/andrebassi/k1s/internal/domain/port"
)

// PodUseCase handles pod-related operations.
type PodUseCase struct {
	repo port.KubernetesRepository
}

// NewPodUseCase creates a new PodUseCase.
func NewPodUseCase(repo port.KubernetesRepository) *PodUseCase {
	return &PodUseCase{repo: repo}
}

// GetPodDetails retrieves comprehensive pod information including logs, events, and metrics.
type PodDetails struct {
	Pod     *entity.PodInfo
	Logs    []entity.LogLine
	Events  []entity.EventInfo
	Metrics *entity.PodMetrics
	Related *entity.RelatedResources
	Node    *entity.NodeInfo
	Helpers []entity.DebugHelper
}

// GetPodDetails fetches all information needed for the pod dashboard.
func (uc *PodUseCase) GetPodDetails(ctx context.Context, namespace, podName string, logOpts port.LogOptions) (*PodDetails, error) {
	details := &PodDetails{}

	// Get pod info
	pod, err := uc.repo.GetPod(ctx, namespace, podName)
	if err != nil {
		return nil, err
	}
	details.Pod = pod

	// Get logs (ignore errors, pod might not have logs yet)
	logs, _ := uc.repo.GetPodLogs(ctx, namespace, podName, logOpts)
	details.Logs = logs

	// Get events
	events, _ := uc.repo.GetPodEvents(ctx, namespace, podName)
	details.Events = events

	// Get metrics (may fail if metrics-server not available)
	metrics, _ := uc.repo.GetPodMetrics(ctx, namespace, podName)
	details.Metrics = metrics

	// Get related resources
	related, _ := uc.repo.GetRelatedResources(ctx, *pod)
	details.Related = related

	// Get node info
	if pod.Node != "" {
		node, _ := uc.repo.GetNodeByName(ctx, pod.Node)
		details.Node = node
	}

	// Analyze issues
	details.Helpers = AnalyzePodIssues(pod, events)

	return details, nil
}

// DeletePod deletes a pod.
func (uc *PodUseCase) DeletePod(ctx context.Context, namespace, name string) error {
	return uc.repo.DeletePod(ctx, namespace, name)
}

// FilterLogs filters logs based on criteria.
func FilterLogs(logs []entity.LogLine, filter string, container string, since time.Duration) []entity.LogLine {
	var result []entity.LogLine

	cutoff := time.Time{}
	if since > 0 {
		cutoff = time.Now().Add(-since)
	}

	filterLower := strings.ToLower(filter)

	for _, log := range logs {
		// Filter by time
		if !cutoff.IsZero() && !log.Timestamp.IsZero() && log.Timestamp.Before(cutoff) {
			continue
		}

		// Filter by container
		if container != "" && log.Container != container {
			continue
		}

		// Filter by content
		if filter != "" && !strings.Contains(strings.ToLower(log.Content), filterLower) {
			continue
		}

		result = append(result, log)
	}

	return result
}

// FilterErrorLogs returns only error logs.
func FilterErrorLogs(logs []entity.LogLine) []entity.LogLine {
	var errors []entity.LogLine
	for _, log := range logs {
		if log.IsError {
			errors = append(errors, log)
		}
	}
	return errors
}

// AnalyzePodIssues examines pod state and events to identify common problems.
func AnalyzePodIssues(pod *entity.PodInfo, events []entity.EventInfo) []entity.DebugHelper {
	var helpers []entity.DebugHelper

	if pod == nil {
		return helpers
	}

	switch pod.Status {
	case "CrashLoopBackOff":
		helpers = append(helpers, entity.DebugHelper{
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
		helpers = append(helpers, entity.DebugHelper{
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
		helpers = append(helpers, entity.DebugHelper{
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
		helpers = append(helpers, entity.DebugHelper{
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

	// Check for missing resource limits
	for _, c := range pod.Containers {
		if c.Resources.MemoryLimit == "0" || c.Resources.MemoryLimit == "" {
			helpers = append(helpers, entity.DebugHelper{
				Issue:    "No memory limit on container " + c.Name,
				Severity: "Warning",
				Suggestions: []string{
					"Set memory limits to prevent OOM issues",
					"Memory limits help with resource planning",
				},
			})
		}
	}

	// Check events for scheduling failures
	for _, e := range events {
		if e.Type == "Warning" && e.Reason == "FailedScheduling" {
			helpers = append(helpers, entity.DebugHelper{
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
