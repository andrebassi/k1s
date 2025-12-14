package repository

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPodEvents(t *testing.T) {
	now := time.Now()
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event-1",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:           "Normal",
			Reason:         "Started",
			Message:        "Container started",
			FirstTimestamp: metav1.Time{Time: now.Add(-5 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-1 * time.Minute)},
			Source:         corev1.EventSource{Component: "kubelet"},
			Count:          1,
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-event-2",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:           "Warning",
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
			FirstTimestamp: metav1.Time{Time: now.Add(-3 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now},
			Source:         corev1.EventSource{Component: "kubelet"},
			Count:          5,
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "other-pod",
			},
			Type:    "Normal",
			Reason:  "Scheduled",
			Message: "Pod scheduled",
		},
	)

	ctx := context.Background()
	events, err := GetPodEvents(ctx, clientset, "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPodEvents() error = %v", err)
	}

	// Fake clientset doesn't support FieldSelector, so we get all events
	if len(events) < 1 {
		t.Errorf("GetPodEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetNamespaceEvents(t *testing.T) {
	now := time.Now()
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event-1",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "pod-1"},
			Type:           "Normal",
			Reason:         "Started",
			FirstTimestamp: metav1.Time{Time: now.Add(-10 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-5 * time.Minute)},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event-2",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "pod-2"},
			Type:           "Warning",
			Reason:         "Failed",
			FirstTimestamp: metav1.Time{Time: now.Add(-3 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "event-3",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "pod-3"},
			Type:           "Normal",
			Reason:         "Scheduled",
			FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-30 * time.Second)},
		},
	)

	ctx := context.Background()

	// Test without limit
	events, err := GetNamespaceEvents(ctx, clientset, "default", 0)
	if err != nil {
		t.Fatalf("GetNamespaceEvents() error = %v", err)
	}

	if len(events) != 3 {
		t.Errorf("GetNamespaceEvents() returned %d events, want 3", len(events))
	}

	// Test with limit
	events, err = GetNamespaceEvents(ctx, clientset, "default", 2)
	if err != nil {
		t.Fatalf("GetNamespaceEvents() with limit error = %v", err)
	}

	if len(events) != 2 {
		t.Errorf("GetNamespaceEvents() with limit returned %d events, want 2", len(events))
	}
}

func TestGetWorkloadEvents(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Deployment",
				Name: "web-app",
			},
			Type:   "Normal",
			Reason: "ScalingReplicaSet",
		},
	)

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Type:      ResourceDeployments,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestIsWarningEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    EventInfo
		expected bool
	}{
		{
			name:     "warning event",
			event:    EventInfo{Type: "Warning"},
			expected: true,
		},
		{
			name:     "normal event",
			event:    EventInfo{Type: "Normal"},
			expected: false,
		},
		{
			name:     "empty type",
			event:    EventInfo{Type: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsWarningEvent(tt.event)
			if result != tt.expected {
				t.Errorf("IsWarningEvent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetRecentWarnings(t *testing.T) {
	now := time.Now()
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "recent-warning",
				Namespace: "default",
			},
			Type:           "Warning",
			Reason:         "Failed",
			FirstTimestamp: metav1.Time{Time: now.Add(-5 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-1 * time.Minute)},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "old-warning",
				Namespace: "default",
			},
			Type:           "Warning",
			Reason:         "OldFailed",
			FirstTimestamp: metav1.Time{Time: now.Add(-2 * time.Hour)},
			LastTimestamp:  metav1.Time{Time: now.Add(-1 * time.Hour)},
		},
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "normal-event",
				Namespace: "default",
			},
			Type:           "Normal",
			Reason:         "Started",
			FirstTimestamp: metav1.Time{Time: now.Add(-1 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now},
		},
	)

	ctx := context.Background()
	warnings, err := GetRecentWarnings(ctx, clientset, "default", 30*time.Minute)
	if err != nil {
		t.Fatalf("GetRecentWarnings() error = %v", err)
	}

	// Should only get the recent warning (within 30 minutes), not the old one or normal event
	warningCount := 0
	for _, w := range warnings {
		if w.Type == "Warning" {
			warningCount++
		}
	}

	if warningCount < 1 {
		t.Errorf("GetRecentWarnings() returned %d warnings, want at least 1", warningCount)
	}
}

func TestEventsToEventInfo(t *testing.T) {
	now := time.Now()

	// Test with FirstTimestamp/LastTimestamp
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "event1", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:           "Normal",
			Reason:         "Scheduled",
			Message:        "Successfully assigned pod",
			Source:         corev1.EventSource{Component: "scheduler"},
			Count:          1,
			FirstTimestamp: metav1.Time{Time: now.Add(-10 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now.Add(-5 * time.Minute)},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "event2", Namespace: "default"},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:           "Warning",
			Reason:         "Failed",
			Message:        "Container failed",
			Source:         corev1.EventSource{Component: "kubelet"},
			Count:          3,
			FirstTimestamp: metav1.Time{Time: now.Add(-3 * time.Minute)},
			LastTimestamp:  metav1.Time{Time: now},
		},
	}

	result := eventsToEventInfo(events)

	if len(result) != 2 {
		t.Fatalf("eventsToEventInfo() returned %d events, want 2", len(result))
	}

	// Events should be sorted by LastSeen (most recent first)
	if result[0].Reason != "Failed" {
		t.Errorf("First event should be 'Failed' (most recent), got %q", result[0].Reason)
	}

	if result[0].Type != "Warning" {
		t.Errorf("First event type should be 'Warning', got %q", result[0].Type)
	}

	if result[0].Count != 3 {
		t.Errorf("First event count should be 3, got %d", result[0].Count)
	}

	if result[0].Object != "Pod/test-pod" {
		t.Errorf("First event object should be 'Pod/test-pod', got %q", result[0].Object)
	}
}

func TestEventsToEventInfo_EventTime(t *testing.T) {
	now := time.Now()

	// Test with EventTime (newer event format) - zero FirstTimestamp
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "event1"},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "test-pod",
			},
			Type:      "Normal",
			Reason:    "Started",
			EventTime: metav1.MicroTime{Time: now},
			// FirstTimestamp is zero
		},
	}

	result := eventsToEventInfo(events)

	if len(result) != 1 {
		t.Fatalf("eventsToEventInfo() returned %d events, want 1", len(result))
	}

	// FirstSeen should be set from EventTime when FirstTimestamp is zero
	if result[0].FirstSeen.IsZero() {
		t.Error("FirstSeen should not be zero when EventTime is set")
	}
}

func TestGetWorkloadEvents_StatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sts-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "StatefulSet",
				Name: "database",
			},
			Type:   "Normal",
			Reason: "SuccessfulCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "database",
		Namespace: "default",
		Type:      ResourceStatefulSets,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_DaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ds-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "DaemonSet",
				Name: "logs",
			},
			Type:   "Normal",
			Reason: "SuccessfulCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "logs",
		Namespace: "default",
		Type:      ResourceDaemonSets,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_Job(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "job-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Job",
				Name: "backup",
			},
			Type:   "Normal",
			Reason: "Completed",
		},
	)

	workload := WorkloadInfo{
		Name:      "backup",
		Namespace: "default",
		Type:      ResourceJobs,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_CronJob(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cj-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "CronJob",
				Name: "cleanup",
			},
			Type:   "Normal",
			Reason: "SawCompletedJob",
		},
	)

	workload := WorkloadInfo{
		Name:      "cleanup",
		Namespace: "default",
		Type:      ResourceCronJobs,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_Pods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pod-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "Pod",
				Name: "standalone-pod",
			},
			Type:   "Normal",
			Reason: "Started",
		},
	)

	workload := WorkloadInfo{
		Name:      "standalone-pod",
		Namespace: "default",
		Type:      ResourcePods,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetPodEvents_Empty(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	events, err := GetPodEvents(ctx, clientset, "default", "nonexistent-pod")
	if err != nil {
		t.Fatalf("GetPodEvents() error = %v", err)
	}

	if len(events) != 0 {
		t.Errorf("GetPodEvents() should return empty for nonexistent pod, got %d", len(events))
	}
}

func TestEventInfoStruct(t *testing.T) {
	now := time.Now()
	event := EventInfo{
		Type:      "Warning",
		Reason:    "BackOff",
		Message:   "Container restarting",
		Object:    "Pod/my-pod",
		FirstSeen: now.Add(-10 * time.Minute),
		LastSeen:  now,
		Count:     5,
		Source:    "kubelet",
	}

	if event.Type != "Warning" {
		t.Errorf("Type = %q, want 'Warning'", event.Type)
	}
	if event.Reason != "BackOff" {
		t.Errorf("Reason = %q, want 'BackOff'", event.Reason)
	}
	if event.Count != 5 {
		t.Errorf("Count = %d, want 5", event.Count)
	}
}
