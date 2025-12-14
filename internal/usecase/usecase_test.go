package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/andrebassi/k1s/internal/domain/entity"
	"github.com/andrebassi/k1s/internal/domain/port"
)

// Mock repository for testing
type mockRepository struct {
	namespaces    []string
	workloads     []entity.WorkloadInfo
	pods          []*entity.PodInfo
	configMaps    []entity.ConfigMapInfo
	secrets       []entity.SecretInfo
	configMapData *entity.ConfigMapData
	secretData    *entity.SecretData
	nodes         []entity.NodeInfo
	err           error
}

func (m *mockRepository) ListNamespaces(ctx context.Context) ([]string, error) {
	return m.namespaces, m.err
}

func (m *mockRepository) ListWorkloads(ctx context.Context, namespace string, resourceType entity.ResourceType) ([]entity.WorkloadInfo, error) {
	return m.workloads, m.err
}

func (m *mockRepository) GetWorkloadPods(ctx context.Context, workload entity.WorkloadInfo) ([]entity.PodInfo, error) {
	var result []entity.PodInfo
	for _, p := range m.pods {
		if p != nil {
			result = append(result, *p)
		}
	}
	return result, m.err
}

func (m *mockRepository) GetPod(ctx context.Context, namespace, name string) (*entity.PodInfo, error) {
	if len(m.pods) > 0 && m.pods[0] != nil {
		return m.pods[0], m.err
	}
	return nil, m.err
}

func (m *mockRepository) DeletePod(ctx context.Context, namespace, name string) error {
	return m.err
}

func (m *mockRepository) GetPodLogs(ctx context.Context, namespace, podName string, opts port.LogOptions) ([]entity.LogLine, error) {
	return nil, m.err
}

func (m *mockRepository) GetPodEvents(ctx context.Context, namespace, podName string) ([]entity.EventInfo, error) {
	return nil, m.err
}

func (m *mockRepository) GetPodMetrics(ctx context.Context, namespace, podName string) (*entity.PodMetrics, error) {
	return nil, m.err
}

func (m *mockRepository) GetRelatedResources(ctx context.Context, pod entity.PodInfo) (*entity.RelatedResources, error) {
	return nil, m.err
}

func (m *mockRepository) ListNodes(ctx context.Context) ([]entity.NodeInfo, error) {
	return m.nodes, m.err
}

func (m *mockRepository) GetNodeByName(ctx context.Context, name string) (*entity.NodeInfo, error) {
	for _, n := range m.nodes {
		if n.Name == name {
			return &n, nil
		}
	}
	return nil, m.err
}

func (m *mockRepository) GetNodePods(ctx context.Context, nodeName string) ([]entity.PodInfo, error) {
	return nil, m.err
}

func (m *mockRepository) ListConfigMaps(ctx context.Context, namespace string) ([]entity.ConfigMapInfo, error) {
	return m.configMaps, m.err
}

func (m *mockRepository) GetConfigMapData(ctx context.Context, namespace, name string) (*entity.ConfigMapData, error) {
	return m.configMapData, m.err
}

func (m *mockRepository) ListSecrets(ctx context.Context, namespace string) ([]entity.SecretInfo, error) {
	return m.secrets, m.err
}

func (m *mockRepository) GetSecretData(ctx context.Context, namespace, name string) (*entity.SecretData, error) {
	return m.secretData, m.err
}

func (m *mockRepository) ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	return m.err
}

func (m *mockRepository) ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error {
	return m.err
}

func (m *mockRepository) RestartDeployment(ctx context.Context, namespace, name string) error {
	return m.err
}

func (m *mockRepository) RestartStatefulSet(ctx context.Context, namespace, name string) error {
	return m.err
}

func (m *mockRepository) RestartDaemonSet(ctx context.Context, namespace, name string) error {
	return m.err
}

func (m *mockRepository) GetCurrentContext() string {
	return "test-context"
}

func (m *mockRepository) ListContexts() ([]string, string, error) {
	return []string{"test-context"}, "test-context", m.err
}

// Test FilterLogs function
func TestFilterLogs(t *testing.T) {
	now := time.Now()
	logs := []entity.LogLine{
		{Content: "Error: something failed", Container: "app", Timestamp: now.Add(-5 * time.Minute), IsError: true},
		{Content: "Info: processing request", Container: "app", Timestamp: now.Add(-3 * time.Minute)},
		{Content: "Debug: internal state", Container: "sidecar", Timestamp: now.Add(-1 * time.Minute)},
		{Content: "Error: another failure", Container: "sidecar", Timestamp: now, IsError: true},
	}

	tests := []struct {
		name          string
		filter        string
		container     string
		since         time.Duration
		expectedCount int
	}{
		{"no filter returns all", "", "", 0, 4},
		{"filter by content", "error", "", 0, 2},
		{"filter by container", "", "app", 0, 2},
		{"filter by content and container", "error", "app", 0, 1},
		{"filter by time (last 2 minutes)", "", "", 2 * time.Minute, 2},
		{"case insensitive filter", "ERROR", "", 0, 2},
		{"no match", "xyz123", "", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterLogs(logs, tt.filter, tt.container, tt.since)
			if len(result) != tt.expectedCount {
				t.Errorf("FilterLogs() returned %d logs, want %d", len(result), tt.expectedCount)
			}
		})
	}
}

func TestFilterErrorLogs(t *testing.T) {
	logs := []entity.LogLine{
		{Content: "Normal log", IsError: false},
		{Content: "Error log 1", IsError: true},
		{Content: "Another normal", IsError: false},
		{Content: "Error log 2", IsError: true},
	}

	result := FilterErrorLogs(logs)
	if len(result) != 2 {
		t.Errorf("FilterErrorLogs() returned %d logs, want 2", len(result))
	}

	for _, log := range result {
		if !log.IsError {
			t.Errorf("FilterErrorLogs() returned a non-error log: %q", log.Content)
		}
	}
}

func TestAnalyzePodIssues(t *testing.T) {
	tests := []struct {
		name         string
		pod          *entity.PodInfo
		events       []entity.EventInfo
		expectIssues int
		expectFirst  string
	}{
		{
			name:         "nil pod",
			pod:          nil,
			events:       nil,
			expectIssues: 0,
		},
		{
			name:         "CrashLoopBackOff",
			pod:          &entity.PodInfo{Status: "CrashLoopBackOff"},
			events:       nil,
			expectIssues: 1,
			expectFirst:  "CrashLoopBackOff",
		},
		{
			name:         "ImagePullBackOff",
			pod:          &entity.PodInfo{Status: "ImagePullBackOff"},
			events:       nil,
			expectIssues: 1,
			expectFirst:  "Image Pull Failed",
		},
		{
			name:         "ErrImagePull",
			pod:          &entity.PodInfo{Status: "ErrImagePull"},
			events:       nil,
			expectIssues: 1,
			expectFirst:  "Image Pull Failed",
		},
		{
			name:         "Pending",
			pod:          &entity.PodInfo{Status: "Pending"},
			events:       nil,
			expectIssues: 1,
			expectFirst:  "Pod Pending",
		},
		{
			name:         "OOMKilled",
			pod:          &entity.PodInfo{Status: "OOMKilled"},
			events:       nil,
			expectIssues: 1,
			expectFirst:  "Out of Memory",
		},
		{
			name: "missing memory limit",
			pod: &entity.PodInfo{
				Status: "Running",
				Containers: []entity.ContainerInfo{
					{Name: "app", Resources: entity.ResourceRequirements{MemoryLimit: ""}},
				},
			},
			events:       nil,
			expectIssues: 1,
		},
		{
			name: "FailedScheduling event",
			pod:  &entity.PodInfo{Status: "Pending"},
			events: []entity.EventInfo{
				{Type: "Warning", Reason: "FailedScheduling", Message: "insufficient memory"},
			},
			expectIssues: 2, // Pending + FailedScheduling
		},
		{
			name:         "healthy pod",
			pod:          &entity.PodInfo{Status: "Running"},
			events:       nil,
			expectIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnalyzePodIssues(tt.pod, tt.events)
			if len(result) != tt.expectIssues {
				t.Errorf("AnalyzePodIssues() returned %d issues, want %d", len(result), tt.expectIssues)
			}
			if tt.expectFirst != "" && len(result) > 0 && result[0].Issue != tt.expectFirst {
				t.Errorf("First issue = %q, want %q", result[0].Issue, tt.expectFirst)
			}
		})
	}
}

// Test UseCase constructors
func TestNewPodUseCase(t *testing.T) {
	mock := &mockRepository{}
	uc := NewPodUseCase(mock)
	if uc == nil {
		t.Error("NewPodUseCase should return non-nil")
	}
	if uc.repo != mock {
		t.Error("NewPodUseCase should store the repository")
	}
}

func TestNewWorkloadUseCase(t *testing.T) {
	mock := &mockRepository{}
	uc := NewWorkloadUseCase(mock)
	if uc == nil {
		t.Error("NewWorkloadUseCase should return non-nil")
	}
	if uc.repo != mock {
		t.Error("NewWorkloadUseCase should store the repository")
	}
}

func TestNewNamespaceUseCase(t *testing.T) {
	mock := &mockRepository{}
	uc := NewNamespaceUseCase(mock)
	if uc == nil {
		t.Error("NewNamespaceUseCase should return non-nil")
	}
	if uc.repo != mock {
		t.Error("NewNamespaceUseCase should store the repository")
	}
}

// Test NamespaceUseCase methods
func TestNamespaceUseCase_ListNamespaces(t *testing.T) {
	mock := &mockRepository{
		namespaces: []string{"default", "kube-system", "production"},
	}
	uc := NewNamespaceUseCase(mock)

	result, err := uc.ListNamespaces(context.Background())
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}
	if len(result) != 3 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 3", len(result))
	}
}

func TestNamespaceUseCase_GetConfigMapData(t *testing.T) {
	mock := &mockRepository{
		configMapData: &entity.ConfigMapData{
			Name:      "test-cm",
			Namespace: "default",
			Data:      map[string]string{"key": "value"},
		},
	}
	uc := NewNamespaceUseCase(mock)

	result, err := uc.GetConfigMapData(context.Background(), "default", "test-cm")
	if err != nil {
		t.Fatalf("GetConfigMapData() error = %v", err)
	}
	if result.Name != "test-cm" {
		t.Errorf("GetConfigMapData().Name = %q, want %q", result.Name, "test-cm")
	}
}

func TestNamespaceUseCase_GetSecretData(t *testing.T) {
	mock := &mockRepository{
		secretData: &entity.SecretData{
			Name:      "test-secret",
			Namespace: "default",
			Type:      "Opaque",
			Data:      map[string]string{"password": "secret123"},
		},
	}
	uc := NewNamespaceUseCase(mock)

	result, err := uc.GetSecretData(context.Background(), "default", "test-secret")
	if err != nil {
		t.Fatalf("GetSecretData() error = %v", err)
	}
	if result.Name != "test-secret" {
		t.Errorf("GetSecretData().Name = %q, want %q", result.Name, "test-secret")
	}
}

// Test WorkloadUseCase methods
func TestWorkloadUseCase_ListWorkloads(t *testing.T) {
	mock := &mockRepository{
		workloads: []entity.WorkloadInfo{
			{Name: "nginx", Type: entity.ResourceDeployments},
			{Name: "redis", Type: entity.ResourceDeployments},
		},
	}
	uc := NewWorkloadUseCase(mock)

	result, err := uc.ListWorkloads(context.Background(), "default", entity.ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("ListWorkloads() returned %d workloads, want 2", len(result))
	}
}

func TestWorkloadUseCase_GetWorkloadPods(t *testing.T) {
	mock := &mockRepository{
		pods: []*entity.PodInfo{
			{Name: "nginx-abc123", Status: "Running"},
			{Name: "nginx-def456", Status: "Running"},
		},
	}
	uc := NewWorkloadUseCase(mock)

	result, err := uc.GetWorkloadPods(context.Background(), entity.WorkloadInfo{Name: "nginx"})
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}
	if len(result) != 2 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 2", len(result))
	}
}

func TestWorkloadUseCase_ScaleWorkload(t *testing.T) {
	mock := &mockRepository{}
	uc := NewWorkloadUseCase(mock)

	tests := []struct {
		name         string
		resourceType entity.ResourceType
	}{
		{"scale deployment", entity.ResourceDeployments},
		{"scale statefulset", entity.ResourceStatefulSets},
		{"scale daemonset (noop)", entity.ResourceDaemonSets},
		{"scale job (noop)", entity.ResourceJobs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.ScaleWorkload(context.Background(), "default", "test", tt.resourceType, 3)
			if err != nil {
				t.Errorf("ScaleWorkload() error = %v", err)
			}
		})
	}
}

func TestWorkloadUseCase_RestartWorkload(t *testing.T) {
	mock := &mockRepository{}
	uc := NewWorkloadUseCase(mock)

	tests := []struct {
		name         string
		resourceType entity.ResourceType
	}{
		{"restart deployment", entity.ResourceDeployments},
		{"restart statefulset", entity.ResourceStatefulSets},
		{"restart daemonset", entity.ResourceDaemonSets},
		{"restart job (noop)", entity.ResourceJobs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.RestartWorkload(context.Background(), "default", "test", tt.resourceType)
			if err != nil {
				t.Errorf("RestartWorkload() error = %v", err)
			}
		})
	}
}

// Test PodUseCase methods
func TestPodUseCase_DeletePod(t *testing.T) {
	mock := &mockRepository{}
	uc := NewPodUseCase(mock)

	err := uc.DeletePod(context.Background(), "default", "test-pod")
	if err != nil {
		t.Errorf("DeletePod() error = %v", err)
	}
}

func TestPodUseCase_GetPodDetails(t *testing.T) {
	mock := &mockRepository{
		pods: []*entity.PodInfo{
			{Name: "test-pod", Namespace: "default", Status: "Running", Node: "node-1"},
		},
		nodes: []entity.NodeInfo{
			{Name: "node-1", Status: "Ready"},
		},
	}
	uc := NewPodUseCase(mock)

	details, err := uc.GetPodDetails(context.Background(), "default", "test-pod", port.LogOptions{})
	if err != nil {
		t.Fatalf("GetPodDetails() error = %v", err)
	}
	if details.Pod == nil {
		t.Error("GetPodDetails().Pod should not be nil")
	}
	if details.Pod.Name != "test-pod" {
		t.Errorf("GetPodDetails().Pod.Name = %q, want %q", details.Pod.Name, "test-pod")
	}
}

func TestNamespaceUseCase_GetNamespaceResources(t *testing.T) {
	mock := &mockRepository{
		workloads: []entity.WorkloadInfo{
			{Name: "pod-1"},
		},
		pods: []*entity.PodInfo{
			{Name: "pod-1", Namespace: "default"},
		},
		configMaps: []entity.ConfigMapInfo{
			{Name: "config-1"},
		},
		secrets: []entity.SecretInfo{
			{Name: "secret-1"},
		},
	}
	uc := NewNamespaceUseCase(mock)

	resources, err := uc.GetNamespaceResources(context.Background(), "default")
	if err != nil {
		t.Fatalf("GetNamespaceResources() error = %v", err)
	}
	if len(resources.Pods) != 1 {
		t.Errorf("GetNamespaceResources().Pods = %d, want 1", len(resources.Pods))
	}
	if len(resources.ConfigMaps) != 1 {
		t.Errorf("GetNamespaceResources().ConfigMaps = %d, want 1", len(resources.ConfigMaps))
	}
	if len(resources.Secrets) != 1 {
		t.Errorf("GetNamespaceResources().Secrets = %d, want 1", len(resources.Secrets))
	}
}

func TestNamespaceUseCase_GetNamespaceResources_Error(t *testing.T) {
	mock := &mockRepository{
		err: context.DeadlineExceeded,
	}
	uc := NewNamespaceUseCase(mock)

	_, err := uc.GetNamespaceResources(context.Background(), "default")
	if err == nil {
		t.Error("GetNamespaceResources() should return error when ListWorkloads fails")
	}
}

func TestPodUseCase_GetPodDetails_Error(t *testing.T) {
	mock := &mockRepository{
		err: context.DeadlineExceeded,
	}
	uc := NewPodUseCase(mock)

	_, err := uc.GetPodDetails(context.Background(), "default", "test-pod", port.LogOptions{})
	if err == nil {
		t.Error("GetPodDetails() should return error when GetPod fails")
	}
}

func TestPodUseCase_GetPodDetails_WithoutNode(t *testing.T) {
	mock := &mockRepository{
		pods: []*entity.PodInfo{
			{Name: "test-pod", Namespace: "default", Status: "Running", Node: ""},
		},
	}
	uc := NewPodUseCase(mock)

	details, err := uc.GetPodDetails(context.Background(), "default", "test-pod", port.LogOptions{})
	if err != nil {
		t.Fatalf("GetPodDetails() error = %v", err)
	}
	if details.Node != nil {
		t.Error("GetPodDetails().Node should be nil when pod has no node")
	}
}
