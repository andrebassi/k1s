package repository

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestListWorkloads_Deployments(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-app",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Errorf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_StatefulSets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "database",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "db"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 2,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Progressing" {
		t.Errorf("Status = %q, want 'Progressing'", workloads[0].Status)
	}
}

func TestListWorkloads_DaemonSets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "log-collector",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "logs"},
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 5,
				NumberReady:            5,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_Jobs(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "backup-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "backup"},
				},
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Completed" {
		t.Errorf("Status = %q, want 'Completed'", workloads[0].Status)
	}
}

func TestListWorkloads_CronJobs(t *testing.T) {
	suspended := false
	clientset := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "daily-report",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				Suspend:  &suspended,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Active" {
		t.Errorf("Status = %q, want 'Active'", workloads[0].Status)
	}
}

func TestListWorkloads_Pods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "standalone-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "main"}},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true, RestartCount: 2},
				},
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourcePods)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].RestartCount != 2 {
		t.Errorf("RestartCount = %d, want 2", workloads[0].RestartCount)
	}
}

func TestListWorkloads_UnknownType(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", "unknowntype")
	if err == nil {
		t.Error("ListWorkloads() should return error for unknown type")
	}
}

func TestGetScaleResourceType(t *testing.T) {
	tests := []struct {
		input    ResourceType
		expected string
	}{
		{ResourceDeployments, "deployment"},
		{ResourceStatefulSets, "statefulset"},
		{ResourceRollouts, "rollout"},
		{ResourceDaemonSets, "daemonsets"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := getScaleResourceType(tt.input)
			if result != tt.expected {
				t.Errorf("getScaleResourceType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	dep, err := GetDeployment(ctx, clientset, "default", "test-deployment")
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}

	if dep.Name != "test-deployment" {
		t.Errorf("Name = %q, want 'test-deployment'", dep.Name)
	}
}

func TestGetStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	sts, err := GetStatefulSet(ctx, clientset, "default", "test-statefulset")
	if err != nil {
		t.Fatalf("GetStatefulSet() error = %v", err)
	}

	if sts.Name != "test-statefulset" {
		t.Errorf("Name = %q, want 'test-statefulset'", sts.Name)
	}
}

func TestGetDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	ds, err := GetDaemonSet(ctx, clientset, "default", "test-daemonset")
	if err != nil {
		t.Fatalf("GetDaemonSet() error = %v", err)
	}

	if ds.Name != "test-daemonset" {
		t.Errorf("Name = %q, want 'test-daemonset'", ds.Name)
	}
}

func TestGetJob(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	job, err := GetJob(ctx, clientset, "default", "test-job")
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if job.Name != "test-job" {
		t.Errorf("Name = %q, want 'test-job'", job.Name)
	}
}

// Note: ScaleDeployment and ScaleStatefulSet tests are skipped because
// the fake clientset doesn't properly support the Scale subresource.
// These functions are tested in integration tests with a real cluster.

func TestRestartDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartDeployment(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartDeployment() error = %v", err)
	}

	dep, _ := clientset.AppsV1().Deployments("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if dep.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestRestartStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartStatefulSet(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartStatefulSet() error = %v", err)
	}

	sts, _ := clientset.AppsV1().StatefulSets("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if sts.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestRestartDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartDaemonSet(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartDaemonSet() error = %v", err)
	}

	ds, _ := clientset.AppsV1().DaemonSets("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if ds.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestGetWorkloadPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-pod-1",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-pod-2",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 2", len(pods))
	}
}

func TestGetWorkloadPods_ForDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "other"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Type:      ResourceDeployments,
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 2", len(pods))
	}
}

func TestGetWorkloadPods_ForStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "db"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db-0",
				Namespace: "default",
				Labels:    map[string]string{"app": "db"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "db",
		Namespace: "default",
		Type:      ResourceStatefulSets,
		Labels:    map[string]string{"app": "db"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 1", len(pods))
	}
}

func TestGetWorkloadPods_ForJob(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "batch-job",
				Namespace: "default",
			},
			Spec: batchv1.JobSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job-name": "batch-job"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "batch-job-xyz",
				Namespace: "default",
				Labels:    map[string]string{"job-name": "batch-job"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "batch-job",
		Namespace: "default",
		Type:      ResourceJobs,
		Labels:    map[string]string{"job-name": "batch-job"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) < 1 {
		t.Errorf("GetWorkloadPods() returned %d pods, want at least 1", len(pods))
	}
}

func TestScaleDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 2},
		}, nil
	})

	// Mock UpdateScale
	clientset.PrependReactor("update", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 5},
		}, nil
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deploy", 5)
	if err != nil {
		t.Fatalf("ScaleDeployment() error = %v", err)
	}
}

func TestScaleDeployment_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale to return error
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deploy", 5)
	if err == nil {
		t.Error("ScaleDeployment() should return error")
	}
}

func TestScaleStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale
	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 3},
		}, nil
	})

	// Mock UpdateScale
	clientset.PrependReactor("update", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 5},
		}, nil
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-sts", 5)
	if err != nil {
		t.Fatalf("ScaleStatefulSet() error = %v", err)
	}
}

func TestScaleStatefulSet_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-sts", 5)
	if err == nil {
		t.Error("ScaleStatefulSet() should return error")
	}
}

func TestGetWorkloadEvents_ForStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sts-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "StatefulSet",
				Name: "db",
			},
			Type:   "Normal",
			Reason: "SuccessfulCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "db",
		Namespace: "default",
		Type:      ResourceStatefulSets,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	// Fake clientset doesn't support FieldSelector, so we get all events
	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_ForDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ds-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "DaemonSet",
				Name: "logging",
			},
			Type:   "Warning",
			Reason: "FailedCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "logging",
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

func TestRestartDeployment_NotFoundError(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartDeployment(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartDeployment() should return error for nonexistent deployment")
	}
}

func TestRestartStatefulSet_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartStatefulSet(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartStatefulSet() should return error for nonexistent statefulset")
	}
}

func TestRestartDaemonSet_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartDaemonSet(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartDaemonSet() should return error for nonexistent daemonset")
	}
}

func TestListWorkloads_DeploymentError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_StatefulSetError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "statefulsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_DaemonSetError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "daemonsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_JobsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_CronJobsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "cronjobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_PodsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourcePods)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestGetWorkloadPods_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	_, err := GetWorkloadPods(ctx, clientset, workload)
	if err == nil {
		t.Error("GetWorkloadPods() should return error on API failure")
	}
}

func TestGetWorkloadPods_EmptyLabels(t *testing.T) {
	// With empty labels, GetWorkloadPods returns all pods (no label filter)
	clientset := fake.NewSimpleClientset()

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	// With no pods in the namespace, should return empty slice
	if len(pods) != 0 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 0", len(pods))
	}
}

func TestListWorkloads_Deployment_ZeroReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "scaled-down",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(0),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "scaled"},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:      0,
				ReadyReplicas: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 replicas, status is "Running" since no replicas are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_StatefulSet_ZeroReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "scaled-down-sts",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(0),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "scaled"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      0,
				ReadyReplicas: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 replicas, status is "Running" since no replicas are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_DaemonSet_ZeroScheduled(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "zero-ds",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "ds"},
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 0,
				NumberReady:            0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 scheduled, status is "Running" since no pods are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_Job_Failed(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "failed-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "failed"},
				},
			},
			Status: batchv1.JobStatus{
				Failed:    1,
				Succeeded: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Failed" {
		t.Errorf("Status = %q, want 'Failed'", workloads[0].Status)
	}
}

func TestListWorkloads_Job_Running(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "running-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "running"},
				},
			},
			Status: batchv1.JobStatus{
				Active:    1,
				Succeeded: 0,
				Failed:    0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_CronJob_Suspended(t *testing.T) {
	suspended := true
	clientset := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "suspended-cron",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				Suspend:  &suspended,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Suspended" {
		t.Errorf("Status = %q, want 'Suspended'", workloads[0].Status)
	}
}
