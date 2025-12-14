// Package port defines the interfaces (ports) for the domain layer.
// These interfaces decouple the domain from external dependencies,
// following the dependency inversion principle.
package port

import (
	"context"

	"github.com/andrebassi/k1s/internal/domain/entity"
)

// KubernetesRepository defines the interface for Kubernetes operations.
// This is the primary port for accessing cluster resources.
type KubernetesRepository interface {
	// Namespace operations
	ListNamespaces(ctx context.Context) ([]string, error)

	// Workload operations
	ListWorkloads(ctx context.Context, namespace string, resourceType entity.ResourceType) ([]entity.WorkloadInfo, error)
	GetWorkloadPods(ctx context.Context, workload entity.WorkloadInfo) ([]entity.PodInfo, error)

	// Pod operations
	GetPod(ctx context.Context, namespace, name string) (*entity.PodInfo, error)
	DeletePod(ctx context.Context, namespace, name string) error
	GetPodLogs(ctx context.Context, namespace, podName string, opts LogOptions) ([]entity.LogLine, error)
	GetPodEvents(ctx context.Context, namespace, podName string) ([]entity.EventInfo, error)
	GetPodMetrics(ctx context.Context, namespace, podName string) (*entity.PodMetrics, error)
	GetRelatedResources(ctx context.Context, pod entity.PodInfo) (*entity.RelatedResources, error)

	// Node operations
	ListNodes(ctx context.Context) ([]entity.NodeInfo, error)
	GetNodeByName(ctx context.Context, name string) (*entity.NodeInfo, error)
	GetNodePods(ctx context.Context, nodeName string) ([]entity.PodInfo, error)

	// ConfigMap and Secret operations
	ListConfigMaps(ctx context.Context, namespace string) ([]entity.ConfigMapInfo, error)
	GetConfigMapData(ctx context.Context, namespace, name string) (*entity.ConfigMapData, error)
	ListSecrets(ctx context.Context, namespace string) ([]entity.SecretInfo, error)
	GetSecretData(ctx context.Context, namespace, name string) (*entity.SecretData, error)

	// Workload actions
	ScaleDeployment(ctx context.Context, namespace, name string, replicas int32) error
	ScaleStatefulSet(ctx context.Context, namespace, name string, replicas int32) error
	RestartDeployment(ctx context.Context, namespace, name string) error
	RestartStatefulSet(ctx context.Context, namespace, name string) error
	RestartDaemonSet(ctx context.Context, namespace, name string) error

	// Context operations
	GetCurrentContext() string
	ListContexts() ([]string, string, error)
}

// LogOptions configures how container logs are retrieved.
type LogOptions struct {
	Container  string
	TailLines  int64
	Since      int64 // seconds
	Previous   bool
	Timestamps bool
}

// ConfigRepository defines the interface for configuration persistence.
type ConfigRepository interface {
	Load() (*Config, error)
	Save(cfg *Config) error
}

// Config holds application configuration.
type Config struct {
	LastNamespace    string
	LastContext      string
	LastResourceType string
	FavoriteItems    []string
	LogLineLimit     int
	RefreshInterval  int
	Theme            string
}
