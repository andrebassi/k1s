package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestListNodes(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "node-1",
				CreationTimestamp: metav1.Time{Time: time.Now()},
				Labels:            map[string]string{"node-role.kubernetes.io/worker": ""},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
				},
				NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.28.0"},
			},
		},
	)

	ctx := context.Background()
	nodes, err := ListNodes(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("ListNodes() returned %d nodes, want 1", len(nodes))
	}

	if nodes[0].Status != "Ready" {
		t.Errorf("Status = %q, want 'Ready'", nodes[0].Status)
	}
}

func TestGetNode(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "master-node",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	ctx := context.Background()
	node, err := GetNode(ctx, clientset, "master-node")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if node.Status != "Ready" {
		t.Errorf("Status = %q, want 'Ready'", node.Status)
	}
}

func TestGetNode_Full(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "worker-1",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
			Labels: map[string]string{
				"kubernetes.io/os":   "linux",
				"kubernetes.io/arch": "amd64",
			},
		},
		Spec: corev1.NodeSpec{
			PodCIDR:    "10.244.1.0/24",
			ProviderID: "aws:///us-east-1a/i-12345",
			Taints: []corev1.Taint{
				{Key: "node-role.kubernetes.io/master", Effect: corev1.TaintEffectNoSchedule},
			},
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("3800m"),
				corev1.ResourceMemory: resource.MustParse("7Gi"),
				corev1.ResourcePods:   resource.MustParse("110"),
			},
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse},
				{Type: corev1.NodeDiskPressure, Status: corev1.ConditionFalse},
			},
			Addresses: []corev1.NodeAddress{
				{Type: corev1.NodeInternalIP, Address: "192.168.1.10"},
				{Type: corev1.NodeExternalIP, Address: "54.123.45.67"},
				{Type: corev1.NodeHostName, Address: "worker-1"},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion:          "v1.28.0",
				ContainerRuntimeVersion: "containerd://1.6.0",
				OSImage:                 "Ubuntu 22.04 LTS",
				KernelVersion:           "5.15.0-generic",
			},
		},
	}

	clientset := fake.NewSimpleClientset(node)

	ctx := context.Background()
	data, err := GetNode(ctx, clientset, "worker-1")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if data.Name != "worker-1" {
		t.Errorf("Name = %q, want 'worker-1'", data.Name)
	}
	if data.Version != "v1.28.0" {
		t.Errorf("Version = %q, want 'v1.28.0'", data.Version)
	}
	if data.InternalIP != "192.168.1.10" {
		t.Errorf("InternalIP = %q, want '192.168.1.10'", data.InternalIP)
	}
	if data.Status != "Ready" {
		t.Errorf("Status = %q, want 'Ready'", data.Status)
	}
}

func TestListNodes_Full(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
				Allocatable: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("4"),
					corev1.ResourceMemory: resource.MustParse("8Gi"),
				},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
				},
			},
		},
	)

	ctx := context.Background()
	nodes, err := ListNodes(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}

	if len(nodes) != 2 {
		t.Errorf("len(nodes) = %d, want 2", len(nodes))
	}

	// First node should be Ready
	if nodes[0].Status != "Ready" {
		t.Errorf("nodes[0].Status = %q, want 'Ready'", nodes[0].Status)
	}
}

func TestGetNode_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetNode(ctx, clientset, "nonexistent")
	if err == nil {
		t.Error("GetNode() should return error for nonexistent node")
	}
}

func TestListNodes_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "nodes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListNodes(ctx, clientset)
	if err == nil {
		t.Error("ListNodes() should return error on API failure")
	}
}

func TestGetNode_WithConditions(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "complex-node",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse, Reason: "KubeletNotReady"},
					{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
					{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
					{Type: corev1.NodePIDPressure, Status: corev1.ConditionFalse},
				},
				Addresses: []corev1.NodeAddress{
					{Type: corev1.NodeInternalIP, Address: "10.0.0.1"},
					{Type: corev1.NodeExternalIP, Address: "1.2.3.4"},
				},
			},
		},
	)

	ctx := context.Background()
	node, err := GetNode(ctx, clientset, "complex-node")
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}

	if node.Status != "NotReady" {
		t.Errorf("Status = %q, want 'NotReady'", node.Status)
	}
}

func TestListNodes_WithRoles(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "control-plane-node",
				CreationTimestamp: metav1.Time{Time: time.Now()},
				Labels: map[string]string{
					"node-role.kubernetes.io/control-plane": "",
					"node-role.kubernetes.io/master":        "",
				},
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
				},
			},
		},
	)

	ctx := context.Background()
	nodes, err := ListNodes(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}

	if len(nodes) != 1 {
		t.Fatalf("ListNodes() returned %d nodes, want 1", len(nodes))
	}

	if !strings.Contains(nodes[0].Roles, "control-plane") && !strings.Contains(nodes[0].Roles, "master") {
		t.Errorf("Roles should contain control-plane or master, got %q", nodes[0].Roles)
	}
}

func TestExtractNodeRoles(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: "<none>",
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: "<none>",
		},
		{
			name: "no role labels",
			labels: map[string]string{
				"kubernetes.io/hostname": "node1",
			},
			expected: "<none>",
		},
		{
			name: "single role",
			labels: map[string]string{
				"node-role.kubernetes.io/worker": "",
			},
			expected: "worker",
		},
		{
			name: "multiple roles",
			labels: map[string]string{
				"node-role.kubernetes.io/control-plane": "",
				"node-role.kubernetes.io/master":        "",
			},
			expected: "control-plane,master",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractNodeRoles(tt.labels)
			// For multiple roles, order may vary
			if tt.name == "multiple roles" {
				if !strings.Contains(result, "control-plane") || !strings.Contains(result, "master") {
					t.Errorf("extractNodeRoles() = %q, want to contain both roles", result)
				}
			} else if result != tt.expected {
				t.Errorf("extractNodeRoles() = %q, want %q", result, tt.expected)
			}
		})
	}
}
