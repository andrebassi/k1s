package repository

import (
	"context"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// EventInfo represents a Kubernetes event with formatted fields.
// Events provide insight into what's happening with pods and other resources.
type EventInfo struct {
	Type      string    // "Normal" or "Warning"
	Reason    string    // Short reason code (e.g., "Pulled", "Started", "Failed")
	Message   string    // Human-readable description of the event
	Source    string    // Component that generated the event (e.g., "kubelet")
	Age       string    // Human-readable age (e.g., "5m", "2h", "3d")
	Count     int32     // Number of times this event has occurred
	FirstSeen time.Time // When the event was first observed
	LastSeen  time.Time // When the event was most recently observed
	Object    string    // The object this event is about (e.g., "Pod/my-pod")
}

// GetPodEvents retrieves all events related to a specific pod.
// Events are sorted by LastSeen time, with most recent first.
func GetPodEvents(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	return eventsToEventInfo(events.Items), nil
}

// GetWorkloadEvents retrieves events for a workload and its managed pods.
// This is useful for seeing the full picture of deployment or statefulset health.
func GetWorkloadEvents(ctx context.Context, clientset kubernetes.Interface, workload WorkloadInfo) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(workload.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	var filtered []corev1.Event
	for _, e := range events.Items {
		// Include events for the workload itself
		if e.InvolvedObject.Name == workload.Name {
			filtered = append(filtered, e)
			continue
		}

		// Include events for pods belonging to this workload
		if workload.Labels != nil {
			pods, _ := GetWorkloadPods(ctx, clientset, workload)
			for _, pod := range pods {
				if e.InvolvedObject.Name == pod.Name {
					filtered = append(filtered, e)
					break
				}
			}
		}
	}

	return eventsToEventInfo(filtered), nil
}

// GetNamespaceEvents retrieves all events in a namespace.
// Results are sorted by LastSeen time with most recent first.
// Use limit > 0 to cap the number of returned events.
func GetNamespaceEvents(ctx context.Context, clientset kubernetes.Interface, namespace string, limit int) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	result := eventsToEventInfo(events.Items)

	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// eventsToEventInfo converts Kubernetes Event objects to EventInfo structs.
// Handles both old-style (FirstTimestamp/LastTimestamp) and new-style (EventTime) events.
func eventsToEventInfo(events []corev1.Event) []EventInfo {
	var result []EventInfo
	for _, e := range events {
		firstSeen := e.FirstTimestamp.Time
		lastSeen := e.LastTimestamp.Time

		// Fall back to EventTime for newer event format
		if firstSeen.IsZero() && !e.EventTime.Time.IsZero() {
			firstSeen = e.EventTime.Time
		}
		if lastSeen.IsZero() {
			lastSeen = firstSeen
		}

		result = append(result, EventInfo{
			Type:      e.Type,
			Reason:    e.Reason,
			Message:   e.Message,
			Source:    e.Source.Component,
			Age:       formatAge(lastSeen),
			Count:     e.Count,
			FirstSeen: firstSeen,
			LastSeen:  lastSeen,
			Object:    e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name,
		})
	}

	// Sort by LastSeen, most recent first
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	return result
}

// IsWarningEvent returns true if the event is a Warning type.
// Warning events typically indicate problems that may need attention.
func IsWarningEvent(e EventInfo) bool {
	return e.Type == "Warning"
}

// GetRecentWarnings retrieves Warning events from the past duration.
// Useful for quickly identifying recent problems in a namespace.
func GetRecentWarnings(ctx context.Context, clientset kubernetes.Interface, namespace string, since time.Duration) ([]EventInfo, error) {
	events, err := GetNamespaceEvents(ctx, clientset, namespace, 0)
	if err != nil {
		//coverage:ignore
		return nil, err
	}

	cutoff := time.Now().Add(-since)
	var warnings []EventInfo
	for _, e := range events {
		if e.Type == "Warning" && e.LastSeen.After(cutoff) {
			warnings = append(warnings, e)
		}
	}
	return warnings, nil
}
