package k8s

import (
	"context"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type EventInfo struct {
	Type      string
	Reason    string
	Message   string
	Source    string
	Age       string
	Count     int32
	FirstSeen time.Time
	LastSeen  time.Time
	Object    string
}

func GetPodEvents(ctx context.Context, clientset *kubernetes.Clientset, namespace, podName string) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	if err != nil {
		return nil, err
	}

	return eventsToEventInfo(events.Items), nil
}

func GetWorkloadEvents(ctx context.Context, clientset *kubernetes.Clientset, workload WorkloadInfo) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(workload.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var filtered []corev1.Event
	for _, e := range events.Items {
		if e.InvolvedObject.Name == workload.Name {
			filtered = append(filtered, e)
			continue
		}

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

func GetNamespaceEvents(ctx context.Context, clientset *kubernetes.Clientset, namespace string, limit int) ([]EventInfo, error) {
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
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

func eventsToEventInfo(events []corev1.Event) []EventInfo {
	var result []EventInfo
	for _, e := range events {
		firstSeen := e.FirstTimestamp.Time
		lastSeen := e.LastTimestamp.Time

		if firstSeen.IsZero() && e.EventTime.Time.IsZero() == false {
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].LastSeen.After(result[j].LastSeen)
	})

	return result
}

func IsWarningEvent(e EventInfo) bool {
	return e.Type == "Warning"
}

func GetRecentWarnings(ctx context.Context, clientset *kubernetes.Clientset, namespace string, since time.Duration) ([]EventInfo, error) {
	events, err := GetNamespaceEvents(ctx, clientset, namespace, 0)
	if err != nil {
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
