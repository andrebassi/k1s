package repository

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestGetPod(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				NodeName:           "node-1",
				ServiceAccountName: "default",
				Containers:         []corev1.Container{{Name: "main", Image: "nginx"}},
			},
			Status: corev1.PodStatus{
				Phase:  corev1.PodRunning,
				PodIP:  "10.0.0.1",
				HostIP: "192.168.1.1",
			},
		},
	)

	ctx := context.Background()
	pod, err := GetPod(ctx, clientset, "default", "test-pod")
	if err != nil {
		t.Fatalf("GetPod() error = %v", err)
	}

	if pod.Name != "test-pod" {
		t.Errorf("Name = %q, want 'test-pod'", pod.Name)
	}
	if pod.Node != "node-1" {
		t.Errorf("Node = %q, want 'node-1'", pod.Node)
	}
	if pod.IP != "10.0.0.1" {
		t.Errorf("IP = %q, want '10.0.0.1'", pod.IP)
	}
}

func TestGetPod_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetPod(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetPod() should return error for nonexistent pod")
	}
}

func TestListAllPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod-a",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod-b",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodPending},
		},
	)

	ctx := context.Background()
	pods, err := ListAllPods(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListAllPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("ListAllPods() returned %d pods, want 2", len(pods))
	}
}

func TestListPodsByNode(t *testing.T) {
	// Note: fake clientset doesn't support FieldSelector filtering
	// This test verifies the function doesn't error and returns pods
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pod-on-node1",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{NodeName: "node-1", Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	ctx := context.Background()
	pods, err := ListPodsByNode(ctx, clientset, "node-1")
	if err != nil {
		t.Fatalf("ListPodsByNode() error = %v", err)
	}

	// Fake clientset returns all pods (doesn't support FieldSelector)
	if len(pods) < 1 {
		t.Errorf("ListPodsByNode() returned %d pods, want at least 1", len(pods))
	}
}

func TestDeletePod(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-to-delete", Namespace: "default"}},
	)

	ctx := context.Background()
	err := DeletePod(ctx, clientset, "default", "pod-to-delete")
	if err != nil {
		t.Fatalf("DeletePod() error = %v", err)
	}

	_, err = clientset.CoreV1().Pods("default").Get(ctx, "pod-to-delete", metav1.GetOptions{})
	if err == nil {
		t.Error("Pod should have been deleted")
	}
}

func TestGetPodStatus(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "terminating",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{DeletionTimestamp: &now},
				Status:     corev1.PodStatus{Phase: corev1.PodRunning},
			},
			expected: "Terminating",
		},
		{
			name: "waiting with reason",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}},
					},
				},
			},
			expected: "ImagePullBackOff",
		},
		{
			name: "running",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
					},
				},
			},
			expected: "Running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPodStatus(tt.pod)
			if result != tt.expected {
				t.Errorf("getPodStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPodToPodInfo(t *testing.T) {
	now := metav1.Now()
	startTime := metav1.NewTime(time.Now().Add(-30 * time.Minute))
	grace := int64(30)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-pod",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"note": "test"},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: "web-abc123",
				},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "main",
					Image:           "nginx:1.21",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: []corev1.EnvVar{
						{Name: "PORT", Value: "8080"},
					},
					Ports: []corev1.ContainerPort{
						{Name: "http", ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
					},
					VolumeMounts: []corev1.VolumeMount{
						{Name: "config", MountPath: "/etc/config", ReadOnly: true},
					},
					LivenessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{Path: "/health", Port: intstr.FromInt(8080)},
						},
					},
					SecurityContext: &corev1.SecurityContext{
						RunAsUser:              int64Ptr(1000),
						RunAsNonRoot:           boolPtr(true),
						ReadOnlyRootFilesystem: boolPtr(true),
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    mustParseQuantity("100m"),
							corev1.ResourceMemory: mustParseQuantity("128Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    mustParseQuantity("500m"),
							corev1.ResourceMemory: mustParseQuantity("512Mi"),
						},
					},
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:  "init",
					Image: "busybox",
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
						},
					},
				},
			},
			ServiceAccountName:            "default",
			RestartPolicy:                 corev1.RestartPolicyAlways,
			DNSPolicy:                     corev1.DNSClusterFirst,
			TerminationGracePeriodSeconds: &grace,
			NodeSelector:                  map[string]string{"disktype": "ssd"},
			Tolerations: []corev1.Toleration{
				{Key: "node-role.kubernetes.io/master", Operator: corev1.TolerationOpExists},
			},
		},
		Status: corev1.PodStatus{
			Phase:     corev1.PodRunning,
			PodIP:     "10.244.0.5",
			HostIP:    "192.168.1.10",
			StartTime: &startTime,
			QOSClass:  corev1.PodQOSBurstable,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "main",
					Ready:        true,
					RestartCount: 2,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{StartedAt: now},
					},
				},
			},
			InitContainerStatuses: []corev1.ContainerStatus{
				{
					Name:  "init",
					Ready: true,
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   0,
							Reason:     "Completed",
							StartedAt:  metav1.NewTime(time.Now().Add(-1 * time.Hour)),
							FinishedAt: metav1.NewTime(time.Now().Add(-55 * time.Minute)),
						},
					},
				},
			},
		},
	}

	info := podToPodInfo(pod)

	if info.Name != "test-pod" {
		t.Errorf("Name = %q, want 'test-pod'", info.Name)
	}
	if info.Namespace != "default" {
		t.Errorf("Namespace = %q, want 'default'", info.Namespace)
	}
	if info.IP != "10.244.0.5" {
		t.Errorf("IP = %q, want '10.244.0.5'", info.IP)
	}
	if info.OwnerKind != "ReplicaSet" {
		t.Errorf("OwnerKind = %q, want 'ReplicaSet'", info.OwnerKind)
	}
	if info.OwnerRef != "web-abc123" {
		t.Errorf("OwnerRef = %q, want 'web-abc123'", info.OwnerRef)
	}
	if info.Restarts != 2 {
		t.Errorf("Restarts = %d, want 2", info.Restarts)
	}
	if len(info.Containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(info.Containers))
	}
	if info.Containers[0].Name != "main" {
		t.Errorf("Container name = %q, want 'main'", info.Containers[0].Name)
	}
	if info.Containers[0].State != "Running" {
		t.Errorf("Container state = %q, want 'Running'", info.Containers[0].State)
	}
	if len(info.InitContainers) != 1 {
		t.Fatalf("len(InitContainers) = %d, want 1", len(info.InitContainers))
	}
	if info.ServiceAccount != "default" {
		t.Errorf("ServiceAccount = %q, want 'default'", info.ServiceAccount)
	}
}

func TestPodToPodInfo_WaitingContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "waiting-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason:  "ImagePullBackOff",
							Message: "Back-off pulling image",
						},
					},
				},
			},
		},
	}

	info := podToPodInfo(pod)
	if len(info.Containers) != 1 {
		t.Fatalf("len(Containers) = %d, want 1", len(info.Containers))
	}
	if info.Containers[0].State != "Waiting" {
		t.Errorf("Container state = %q, want 'Waiting'", info.Containers[0].State)
	}
	if info.Containers[0].Reason != "ImagePullBackOff" {
		t.Errorf("Container reason = %q, want 'ImagePullBackOff'", info.Containers[0].Reason)
	}
}

func TestPodToPodInfo_TerminatedContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "terminated-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "nginx"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "app",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:   1,
							Reason:     "Error",
							Message:    "Container crashed",
							StartedAt:  metav1.NewTime(time.Now().Add(-10 * time.Minute)),
							FinishedAt: metav1.NewTime(time.Now().Add(-5 * time.Minute)),
						},
					},
				},
			},
		},
	}

	info := podToPodInfo(pod)
	if info.Containers[0].State != "Terminated" {
		t.Errorf("Container state = %q, want 'Terminated'", info.Containers[0].State)
	}
	if info.Containers[0].ExitCode == nil || *info.Containers[0].ExitCode != 1 {
		t.Errorf("Container exit code = %v, want 1", info.Containers[0].ExitCode)
	}
}

func TestGetPodStatus_MoreCases(t *testing.T) {
	now := metav1.Now()

	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected string
	}{
		{
			name: "succeeded phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodSucceeded,
				},
			},
			expected: "Succeeded",
		},
		{
			name: "failed phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
				},
			},
			expected: "Failed",
		},
		{
			name: "pending no status",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodPending,
				},
			},
			expected: "Pending",
		},
		{
			name: "container terminated with reason",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									Reason: "OOMKilled",
								},
							},
						},
					},
				},
			},
			expected: "OOMKilled",
		},
		{
			name: "container terminated no reason falls through to phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodFailed,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 137,
								},
							},
						},
					},
				},
			},
			expected: "Failed",
		},
		{
			name: "unknown phase",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodUnknown,
				},
			},
			expected: "Unknown",
		},
		{
			name: "terminating with grace period",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					DeletionTimestamp:          &now,
					DeletionGracePeriodSeconds: int64Ptr(30),
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
			expected: "Terminating",
		},
		{
			name: "container waiting with reason CrashLoopBackOff",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{
									Reason: "CrashLoopBackOff",
								},
							},
						},
					},
				},
			},
			expected: "CrashLoopBackOff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPodStatus(tt.pod)
			if result != tt.expected {
				t.Errorf("getPodStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestListAllPods_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListAllPods(ctx, clientset, "default")
	if err == nil {
		t.Error("ListAllPods() should return error on API failure")
	}
}

func TestListPodsByNode_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListPodsByNode(ctx, clientset, "node-1")
	if err == nil {
		t.Error("ListPodsByNode() should return error on API failure")
	}
}

func TestCountPodsPerNode(t *testing.T) {
	tests := []struct {
		name     string
		pods     *corev1.PodList
		expected map[string]int
	}{
		{
			name:     "nil pods",
			pods:     nil,
			expected: map[string]int{},
		},
		{
			name:     "empty pods",
			pods:     &corev1.PodList{},
			expected: map[string]int{},
		},
		{
			name: "pods on nodes",
			pods: &corev1.PodList{
				Items: []corev1.Pod{
					{Spec: corev1.PodSpec{NodeName: "node1"}},
					{Spec: corev1.PodSpec{NodeName: "node1"}},
					{Spec: corev1.PodSpec{NodeName: "node2"}},
					{Spec: corev1.PodSpec{NodeName: ""}}, // Pending pod
				},
			},
			expected: map[string]int{"node1": 2, "node2": 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countPodsPerNode(tt.pods)
			if len(result) != len(tt.expected) {
				t.Errorf("countPodsPerNode() len = %d, want %d", len(result), len(tt.expected))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("countPodsPerNode()[%s] = %d, want %d", k, result[k], v)
				}
			}
		})
	}
}
