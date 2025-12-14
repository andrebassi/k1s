// Package entity defines the core domain entities for k1s.
// These entities represent Kubernetes resources and are independent
// of any external framework or infrastructure.
package entity

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// ResourceType identifies the kind of Kubernetes workload resource.
type ResourceType string

// Supported workload resource types.
const (
	ResourcePods         ResourceType = "pods"
	ResourceDeployments  ResourceType = "deployments"
	ResourceStatefulSets ResourceType = "statefulsets"
	ResourceDaemonSets   ResourceType = "daemonsets"
	ResourceJobs         ResourceType = "jobs"
	ResourceCronJobs     ResourceType = "cronjobs"
)

// AllResourceTypes lists all supported workload types in display order.
var AllResourceTypes = []ResourceType{
	ResourceDeployments,
	ResourceStatefulSets,
	ResourceDaemonSets,
	ResourceJobs,
	ResourceCronJobs,
	ResourcePods,
}

// WorkloadInfo provides a summary view of a Kubernetes workload.
type WorkloadInfo struct {
	Name         string
	Namespace    string
	Type         ResourceType
	Ready        string
	Replicas     int32
	Age          string
	Status       string
	Labels       map[string]string
	RestartCount int32
}

// PodInfo provides comprehensive information about a Kubernetes pod.
type PodInfo struct {
	Name                   string
	Namespace              string
	Node                   string
	Status                 string
	Ready                  string
	Restarts               int32
	Age                    string
	IP                     string
	HostIP                 string
	Labels                 map[string]string
	Annotations            map[string]string
	Containers             []ContainerInfo
	InitContainers         []ContainerInfo
	Conditions             []corev1.PodCondition
	Phase                  corev1.PodPhase
	OwnerRef               string
	OwnerKind              string
	QoSClass               string
	ServiceAccount         string
	Volumes                []VolumeInfo
	RestartPolicy          string
	DNSPolicy              string
	PriorityClassName      string
	Priority               *int32
	NodeSelector           map[string]string
	Tolerations            []TolerationInfo
	TerminationGracePeriod int64
	StartTime              string
}

// ContainerInfo provides details about a container within a pod.
type ContainerInfo struct {
	Name            string
	Image           string
	ImagePullPolicy string
	Ready           bool
	RestartCount    int32
	State           string
	Reason          string
	Message         string
	StartedAt       string
	FinishedAt      string
	ExitCode        *int32
	Resources       ResourceRequirements
	Ports           []ContainerPort
	LivenessProbe   *ProbeInfo
	ReadinessProbe  *ProbeInfo
	StartupProbe    *ProbeInfo
	SecurityContext *SecurityContextInfo
	EnvVarCount     int
	VolumeMounts    []VolumeMountInfo
}

// ContainerPort represents an exposed container port.
type ContainerPort struct {
	Name          string
	ContainerPort int32
	Protocol      string
}

// VolumeMountInfo describes a volume mount within a container.
type VolumeMountInfo struct {
	Name      string
	MountPath string
	ReadOnly  bool
}

// TolerationInfo describes a pod toleration for node taints.
type TolerationInfo struct {
	Key      string
	Operator string
	Value    string
	Effect   string
}

// ProbeInfo describes a container health probe configuration.
type ProbeInfo struct {
	Type             string
	Path             string
	Port             int32
	Scheme           string
	Command          []string
	InitialDelay     int32
	Period           int32
	Timeout          int32
	SuccessThreshold int32
	FailureThreshold int32
}

// SecurityContextInfo contains container security settings.
type SecurityContextInfo struct {
	RunAsUser    *int64
	RunAsGroup   *int64
	RunAsNonRoot *bool
	Privileged   *bool
	ReadOnlyRoot *bool
}

// VolumeInfo describes a volume attached to a pod.
type VolumeInfo struct {
	Name   string
	Type   string
	Source string
}

// ResourceRequirements contains CPU and memory requests and limits.
type ResourceRequirements struct {
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
}

// ConfigMapInfo provides a summary of a ConfigMap resource.
type ConfigMapInfo struct {
	Name string
	Age  string
	Keys int
}

// ConfigMapData contains the full data of a ConfigMap.
type ConfigMapData struct {
	Name      string
	Namespace string
	Data      map[string]string
	Age       string
}

// NodeInfo provides information about a cluster node.
type NodeInfo struct {
	Name       string
	Status     string
	Roles      string
	Age        string
	Version    string
	InternalIP string
	PodCount   int
	CPU        string
	Memory     string
}

// SecretInfo provides a summary of a Secret resource.
type SecretInfo struct {
	Name string
	Type string
	Age  string
	Keys int
}

// SecretData contains the full data of a Secret.
type SecretData struct {
	Name      string
	Namespace string
	Type      string
	Data      map[string]string
	Age       string
}

// LogLine represents a single line from container logs.
type LogLine struct {
	Timestamp time.Time
	Container string
	Content   string
	IsError   bool
}

// EventInfo represents a Kubernetes event.
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

// PodMetrics contains resource usage data for a pod.
type PodMetrics struct {
	Name       string
	Namespace  string
	Containers []ContainerMetrics
}

// ContainerMetrics contains CPU and memory usage for a container.
type ContainerMetrics struct {
	Name        string
	CPUUsage    string
	MemoryUsage string
	CPUPercent  float64
	MemPercent  float64
}

// RelatedResources contains resources related to a pod.
type RelatedResources struct {
	Services        []ServiceInfo
	Ingresses       []IngressInfo
	VirtualServices []VirtualServiceInfo
	Gateways        []GatewayInfo
	ConfigMaps      []string
	Secrets         []string
	Owner           *OwnerInfo
}

// ServiceInfo represents a Kubernetes Service.
type ServiceInfo struct {
	Name      string
	Type      string
	ClusterIP string
	Ports     []string
	Endpoints []string
}

// IngressInfo represents a Kubernetes Ingress.
type IngressInfo struct {
	Name  string
	Hosts []string
	Paths []string
}

// VirtualServiceInfo represents an Istio VirtualService.
type VirtualServiceInfo struct {
	Name     string
	Hosts    []string
	Gateways []string
	Routes   []RouteInfo
}

// RouteInfo represents a route in an Istio VirtualService.
type RouteInfo struct {
	Match        string
	Destinations []DestinationInfo
}

// DestinationInfo represents a destination in an Istio route.
type DestinationInfo struct {
	Host   string
	Subset string
	Port   int32
	Weight int32
}

// GatewayInfo represents an Istio Gateway.
type GatewayInfo struct {
	Name    string
	Servers []GatewayServerInfo
}

// GatewayServerInfo represents a server in an Istio Gateway.
type GatewayServerInfo struct {
	Port     int32
	Protocol string
	Hosts    []string
	TLSMode  string
}

// OwnerInfo contains owner reference information.
type OwnerInfo struct {
	Kind         string
	Name         string
	WorkloadKind string
	WorkloadName string
}

// DebugHelper provides diagnostic information about a pod issue.
type DebugHelper struct {
	Issue       string
	Severity    string
	Suggestions []string
}
