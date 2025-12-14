package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
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
// This is used for listing workloads in the navigation view.
type WorkloadInfo struct {
	Name         string            // Workload name
	Namespace    string            // Namespace containing the workload
	Type         ResourceType      // Type of workload (deployment, statefulset, etc.)
	Ready        string            // Ready status (e.g., "3/3")
	Replicas     int32             // Desired replica count
	Age          string            // Human-readable age
	Status       string            // Current status (Running, Progressing, Failed, etc.)
	Labels       map[string]string // Selector labels for finding pods
	RestartCount int32             // Total restart count across all pods
}

// PodInfo provides comprehensive information about a Kubernetes pod.
// This includes all details needed for debugging and inspection.
type PodInfo struct {
	Name                   string                 // Pod name
	Namespace              string                 // Namespace
	Node                   string                 // Node where the pod is scheduled
	Status                 string                 // Current status (Running, Pending, Failed, etc.)
	Ready                  string                 // Ready containers (e.g., "2/2")
	Restarts               int32                  // Total restart count
	Age                    string                 // Human-readable age
	IP                     string                 // Pod IP address
	HostIP                 string                 // Node IP address
	Labels                 map[string]string      // Pod labels
	Annotations            map[string]string      // Pod annotations
	Containers             []ContainerInfo        // Regular containers
	InitContainers         []ContainerInfo        // Init containers
	Conditions             []corev1.PodCondition  // Pod conditions
	Phase                  corev1.PodPhase        // Pod phase
	OwnerRef               string                 // Owner reference name
	OwnerKind              string                 // Owner reference kind
	QoSClass               string                 // Quality of Service class
	ServiceAccount         string                 // Service account name
	Volumes                []VolumeInfo           // Volume definitions
	RestartPolicy          string                 // Restart policy
	DNSPolicy              string                 // DNS policy
	PriorityClassName      string                 // Priority class name
	Priority               *int32                 // Scheduling priority
	NodeSelector           map[string]string      // Node selector constraints
	Tolerations            []TolerationInfo       // Node tolerations
	TerminationGracePeriod int64                  // Termination grace period in seconds
	StartTime              string                 // Pod start time
}

// ContainerInfo provides details about a container within a pod.
type ContainerInfo struct {
	Name            string               // Container name
	Image           string               // Container image
	ImagePullPolicy string               // Image pull policy
	Ready           bool                 // Whether the container is ready
	RestartCount    int32                // Number of restarts
	State           string               // Current state (Running, Waiting, Terminated)
	Reason          string               // Reason for current state
	Message         string               // Additional state message
	StartedAt       string               // Container start time
	FinishedAt      string               // Container finish time (if terminated)
	ExitCode        *int32               // Exit code (if terminated)
	Resources       ResourceRequirements // Resource requests and limits
	Ports           []ContainerPort      // Exposed ports
	LivenessProbe   *ProbeInfo           // Liveness probe configuration
	ReadinessProbe  *ProbeInfo           // Readiness probe configuration
	StartupProbe    *ProbeInfo           // Startup probe configuration
	SecurityContext *SecurityContextInfo // Security context settings
	EnvVarCount     int                  // Number of environment variables
	VolumeMounts    []VolumeMountInfo    // Volume mount configurations
}

// ContainerPort represents an exposed container port.
type ContainerPort struct {
	Name          string // Port name (optional)
	ContainerPort int32  // Port number
	Protocol      string // Protocol (TCP, UDP)
}

// VolumeMountInfo describes a volume mount within a container.
type VolumeMountInfo struct {
	Name      string // Volume name
	MountPath string // Mount path in the container
	ReadOnly  bool   // Whether the mount is read-only
}

// TolerationInfo describes a pod toleration for node taints.
type TolerationInfo struct {
	Key      string // Taint key to tolerate
	Operator string // Operator (Equal, Exists)
	Value    string // Taint value to match
	Effect   string // Taint effect (NoSchedule, NoExecute, PreferNoSchedule)
}

// ProbeInfo describes a container health probe configuration.
type ProbeInfo struct {
	Type             string   // Probe type: HTTP, TCP, Exec, or gRPC
	Path             string   // HTTP path (for HTTP probes)
	Port             int32    // Target port
	Scheme           string   // HTTP scheme (HTTP or HTTPS)
	Command          []string // Command to execute (for Exec probes)
	InitialDelay     int32    // Initial delay in seconds
	Period           int32    // Check period in seconds
	Timeout          int32    // Timeout in seconds
	SuccessThreshold int32    // Consecutive successes required
	FailureThreshold int32    // Consecutive failures required
}

// SecurityContextInfo contains container security settings.
type SecurityContextInfo struct {
	RunAsUser    *int64 // User ID to run as
	RunAsGroup   *int64 // Group ID to run as
	RunAsNonRoot *bool  // Whether to run as non-root
	Privileged   *bool  // Whether to run in privileged mode
	ReadOnlyRoot *bool  // Whether root filesystem is read-only
}

// VolumeInfo describes a volume attached to a pod.
type VolumeInfo struct {
	Name   string // Volume name
	Type   string // Volume type (ConfigMap, Secret, PVC, EmptyDir, etc.)
	Source string // Source name (ConfigMap/Secret/PVC name)
}

// ResourceRequirements contains CPU and memory requests and limits.
type ResourceRequirements struct {
	CPURequest    string // CPU request (e.g., "100m", "0.5")
	CPULimit      string // CPU limit
	MemoryRequest string // Memory request (e.g., "128Mi", "1Gi")
	MemoryLimit   string // Memory limit
}

// ConfigMapInfo provides a summary of a ConfigMap resource.
type ConfigMapInfo struct {
	Name string // ConfigMap name
	Age  string // Human-readable age
	Keys int    // Number of data keys
}

// NodeInfo provides information about a cluster node.
type NodeInfo struct {
	Name       string // Node name
	Status     string // Node status (Ready, NotReady)
	Roles      string // Node roles (master, worker, etc.)
	Age        string // Human-readable age
	Version    string // Kubelet version
	InternalIP string // Node internal IP address
	PodCount   int    // Number of pods on the node
	CPU        string // CPU capacity
	Memory     string // Memory capacity
}

// SecretInfo provides a summary of a Secret resource.
type SecretInfo struct {
	Name string // Secret name
	Type string // Secret type (Opaque, kubernetes.io/tls, etc.)
	Age  string // Human-readable age
	Keys int    // Number of data keys
}

// ListNamespaces returns all namespace names in the cluster, sorted alphabetically.
func ListNamespaces(ctx context.Context, clientset *kubernetes.Clientset) ([]string, error) {
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var namespaces []string
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}
	sort.Strings(namespaces)
	return namespaces, nil
}

// ListWorkloads returns all workloads of the specified type in a namespace.
// Supports pods, deployments, statefulsets, daemonsets, jobs, and cronjobs.
func ListWorkloads(ctx context.Context, clientset *kubernetes.Clientset, namespace string, resourceType ResourceType) ([]WorkloadInfo, error) {
	switch resourceType {
	case ResourceDeployments:
		return listDeployments(ctx, clientset, namespace)
	case ResourceStatefulSets:
		return listStatefulSets(ctx, clientset, namespace)
	case ResourceDaemonSets:
		return listDaemonSets(ctx, clientset, namespace)
	case ResourceJobs:
		return listJobs(ctx, clientset, namespace)
	case ResourceCronJobs:
		return listCronJobs(ctx, clientset, namespace)
	case ResourcePods:
		return listPodsAsWorkloads(ctx, clientset, namespace)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}
}

func listDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	deps, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, d := range deps.Items {
		status := "Running"
		if d.Status.ReadyReplicas < d.Status.Replicas {
			status = "Progressing"
		}
		if d.Status.ReadyReplicas == 0 && d.Status.Replicas > 0 {
			status = "NotReady"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Type:      ResourceDeployments,
			Ready:     fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas),
			Replicas:  d.Status.Replicas,
			Age:       formatAge(d.CreationTimestamp.Time),
			Status:    status,
			Labels:    d.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	sts, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, s := range sts.Items {
		status := "Running"
		if s.Status.ReadyReplicas < s.Status.Replicas {
			status = "Progressing"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      s.Name,
			Namespace: s.Namespace,
			Type:      ResourceStatefulSets,
			Ready:     fmt.Sprintf("%d/%d", s.Status.ReadyReplicas, s.Status.Replicas),
			Replicas:  s.Status.Replicas,
			Age:       formatAge(s.CreationTimestamp.Time),
			Status:    status,
			Labels:    s.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listDaemonSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	ds, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, d := range ds.Items {
		status := "Running"
		if d.Status.NumberReady < d.Status.DesiredNumberScheduled {
			status = "Progressing"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      d.Name,
			Namespace: d.Namespace,
			Type:      ResourceDaemonSets,
			Ready:     fmt.Sprintf("%d/%d", d.Status.NumberReady, d.Status.DesiredNumberScheduled),
			Replicas:  d.Status.DesiredNumberScheduled,
			Age:       formatAge(d.CreationTimestamp.Time),
			Status:    status,
			Labels:    d.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	jobs, err := clientset.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, j := range jobs.Items {
		status := "Running"
		if j.Status.Succeeded > 0 {
			status = "Completed"
		} else if j.Status.Failed > 0 {
			status = "Failed"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      j.Name,
			Namespace: j.Namespace,
			Type:      ResourceJobs,
			Ready:     fmt.Sprintf("%d/%d", j.Status.Succeeded, *j.Spec.Completions),
			Age:       formatAge(j.CreationTimestamp.Time),
			Status:    status,
			Labels:    j.Spec.Selector.MatchLabels,
		})
	}
	return workloads, nil
}

func listCronJobs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	cjs, err := clientset.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, cj := range cjs.Items {
		status := "Active"
		if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
			status = "Suspended"
		}

		workloads = append(workloads, WorkloadInfo{
			Name:      cj.Name,
			Namespace: cj.Namespace,
			Type:      ResourceCronJobs,
			Ready:     fmt.Sprintf("%d active", len(cj.Status.Active)),
			Age:       formatAge(cj.CreationTimestamp.Time),
			Status:    status,
		})
	}
	return workloads, nil
}

func listPodsAsWorkloads(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]WorkloadInfo, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var workloads []WorkloadInfo
	for _, p := range pods.Items {
		var restartCount int32
		for _, cs := range p.Status.ContainerStatuses {
			restartCount += cs.RestartCount
		}

		ready := 0
		for _, cs := range p.Status.ContainerStatuses {
			if cs.Ready {
				ready++
			}
		}

		workloads = append(workloads, WorkloadInfo{
			Name:         p.Name,
			Namespace:    p.Namespace,
			Type:         ResourcePods,
			Ready:        fmt.Sprintf("%d/%d", ready, len(p.Spec.Containers)),
			Age:          formatAge(p.CreationTimestamp.Time),
			Status:       string(p.Status.Phase),
			Labels:       p.Labels,
			RestartCount: restartCount,
		})
	}
	return workloads, nil
}

// GetWorkloadPods returns all pods belonging to a workload.
// Uses label selectors to find pods managed by the workload.
func GetWorkloadPods(ctx context.Context, clientset *kubernetes.Clientset, workload WorkloadInfo) ([]PodInfo, error) {
	if workload.Type == ResourcePods {
		pod, err := clientset.CoreV1().Pods(workload.Namespace).Get(ctx, workload.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		return []PodInfo{podToPodInfo(pod)}, nil
	}

	labelSelector := labels.SelectorFromSet(workload.Labels).String()
	pods, err := clientset.CoreV1().Pods(workload.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var podInfos []PodInfo
	for _, p := range pods.Items {
		podInfos = append(podInfos, podToPodInfo(&p))
	}
	return podInfos, nil
}

// GetPod retrieves detailed information about a specific pod.
func GetPod(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*PodInfo, error) {
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	info := podToPodInfo(pod)
	return &info, nil
}

// ListAllPods returns all pods in a namespace as PodInfo
func ListAllPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]PodInfo, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var podInfos []PodInfo
	for _, p := range pods.Items {
		podInfos = append(podInfos, podToPodInfo(&p))
	}

	// Sort pods by name for consistent display
	sort.Slice(podInfos, func(i, j int) bool {
		return podInfos[i].Name < podInfos[j].Name
	})

	return podInfos, nil
}

// ListConfigMaps returns all configmaps in a namespace
func ListConfigMaps(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]ConfigMapInfo, error) {
	cms, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var cmInfos []ConfigMapInfo
	for _, cm := range cms.Items {
		cmInfos = append(cmInfos, ConfigMapInfo{
			Name: cm.Name,
			Age:  formatAge(cm.CreationTimestamp.Time),
			Keys: len(cm.Data),
		})
	}

	sort.Slice(cmInfos, func(i, j int) bool {
		return cmInfos[i].Name < cmInfos[j].Name
	})

	return cmInfos, nil
}

// ConfigMapData holds full ConfigMap data
type ConfigMapData struct {
	Name      string
	Namespace string
	Age       string
	Data      map[string]string
}

// GetConfigMap returns full ConfigMap data
func GetConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*ConfigMapData, error) {
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &ConfigMapData{
		Name:      cm.Name,
		Namespace: cm.Namespace,
		Age:       formatAge(cm.CreationTimestamp.Time),
		Data:      cm.Data,
	}, nil
}

// ListSecrets returns all secrets in a namespace
func ListSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) ([]SecretInfo, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var secretInfos []SecretInfo
	for _, s := range secrets.Items {
		secretInfos = append(secretInfos, SecretInfo{
			Name: s.Name,
			Type: string(s.Type),
			Age:  formatAge(s.CreationTimestamp.Time),
			Keys: len(s.Data),
		})
	}

	sort.Slice(secretInfos, func(i, j int) bool {
		return secretInfos[i].Name < secretInfos[j].Name
	})

	return secretInfos, nil
}

// SecretData holds full Secret data with decoded values
type SecretData struct {
	Name      string
	Namespace string
	Type      string
	Age       string
	Data      map[string]string // Decoded from base64
}

// ListNodes returns all nodes in the cluster
func ListNodes(ctx context.Context, clientset *kubernetes.Clientset) ([]NodeInfo, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get pod counts per node
	pods, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	podCountByNode := make(map[string]int)
	if pods != nil {
		for _, p := range pods.Items {
			if p.Spec.NodeName != "" {
				podCountByNode[p.Spec.NodeName]++
			}
		}
	}

	var nodeInfos []NodeInfo
	for _, n := range nodes.Items {
		// Get node status
		status := "Unknown"
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}

		// Get node roles
		var roles []string
		for label := range n.Labels {
			if strings.HasPrefix(label, "node-role.kubernetes.io/") {
				role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
				if role != "" {
					roles = append(roles, role)
				}
			}
		}
		roleStr := strings.Join(roles, ",")
		if roleStr == "" {
			roleStr = "<none>"
		}

		// Get internal IP
		var internalIP string
		for _, addr := range n.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
				break
			}
		}

		// Get CPU and Memory capacity
		cpu := n.Status.Capacity.Cpu().String()
		memory := n.Status.Capacity.Memory().String()

		nodeInfos = append(nodeInfos, NodeInfo{
			Name:       n.Name,
			Status:     status,
			Roles:      roleStr,
			Age:        formatAge(n.CreationTimestamp.Time),
			Version:    n.Status.NodeInfo.KubeletVersion,
			InternalIP: internalIP,
			PodCount:   podCountByNode[n.Name],
			CPU:        cpu,
			Memory:     memory,
		})
	}

	sort.Slice(nodeInfos, func(i, j int) bool {
		return nodeInfos[i].Name < nodeInfos[j].Name
	})

	return nodeInfos, nil
}

// GetNode returns information about a specific node
func GetNode(ctx context.Context, clientset *kubernetes.Clientset, nodeName string) (*NodeInfo, error) {
	n, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Get pod count for this node
	pods, _ := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	podCount := 0
	if pods != nil {
		podCount = len(pods.Items)
	}

	// Get node status
	status := "Unknown"
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				status = "Ready"
			} else {
				status = "NotReady"
			}
			break
		}
	}

	// Get node roles
	var roles []string
	for label := range n.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	roleStr := strings.Join(roles, ",")
	if roleStr == "" {
		roleStr = "<none>"
	}

	// Get internal IP
	var internalIP string
	for _, addr := range n.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			internalIP = addr.Address
			break
		}
	}

	// Get CPU and Memory capacity
	cpu := n.Status.Capacity.Cpu().String()
	memory := n.Status.Capacity.Memory().String()

	return &NodeInfo{
		Name:       n.Name,
		Status:     status,
		Roles:      roleStr,
		Age:        formatAge(n.CreationTimestamp.Time),
		Version:    n.Status.NodeInfo.KubeletVersion,
		InternalIP: internalIP,
		PodCount:   podCount,
		CPU:        cpu,
		Memory:     memory,
	}, nil
}

// ListPodsByNode returns all pods running on a specific node
func ListPodsByNode(ctx context.Context, clientset *kubernetes.Clientset, nodeName string) ([]PodInfo, error) {
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, err
	}

	var podInfos []PodInfo
	for _, p := range pods.Items {
		podInfos = append(podInfos, podToPodInfo(&p))
	}

	sort.Slice(podInfos, func(i, j int) bool {
		return podInfos[i].Namespace + "/" + podInfos[i].Name < podInfos[j].Namespace + "/" + podInfos[j].Name
	})

	return podInfos, nil
}

// GetSecret returns full Secret data with decoded values
func GetSecret(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*SecretData, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Decode base64 values
	decodedData := make(map[string]string)
	for k, v := range secret.Data {
		decodedData[k] = string(v) // secret.Data is already []byte, not base64 encoded
	}

	return &SecretData{
		Name:      secret.Name,
		Namespace: secret.Namespace,
		Type:      string(secret.Type),
		Age:       formatAge(secret.CreationTimestamp.Time),
		Data:      decodedData,
	}, nil
}

func podToPodInfo(p *corev1.Pod) PodInfo {
	var restarts int32
	var containers []ContainerInfo

	// Build container status map for quick lookup
	statusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range p.Status.ContainerStatuses {
		statusMap[cs.Name] = cs
	}

	for _, c := range p.Spec.Containers {
		ci := ContainerInfo{
			Name:            c.Name,
			Image:           c.Image,
			ImagePullPolicy: string(c.ImagePullPolicy),
			EnvVarCount:     len(c.Env) + len(c.EnvFrom),
			Resources: ResourceRequirements{
				CPURequest:    c.Resources.Requests.Cpu().String(),
				CPULimit:      c.Resources.Limits.Cpu().String(),
				MemoryRequest: c.Resources.Requests.Memory().String(),
				MemoryLimit:   c.Resources.Limits.Memory().String(),
			},
		}

		// Parse ports
		for _, port := range c.Ports {
			ci.Ports = append(ci.Ports, ContainerPort{
				Name:          port.Name,
				ContainerPort: port.ContainerPort,
				Protocol:      string(port.Protocol),
			})
		}

		// Parse volume mounts
		for _, vm := range c.VolumeMounts {
			ci.VolumeMounts = append(ci.VolumeMounts, VolumeMountInfo{
				Name:      vm.Name,
				MountPath: vm.MountPath,
				ReadOnly:  vm.ReadOnly,
			})
		}

		// Parse probes
		ci.LivenessProbe = parseProbe(c.LivenessProbe)
		ci.ReadinessProbe = parseProbe(c.ReadinessProbe)
		ci.StartupProbe = parseProbe(c.StartupProbe)

		// Parse security context
		if c.SecurityContext != nil {
			ci.SecurityContext = &SecurityContextInfo{
				RunAsUser:    c.SecurityContext.RunAsUser,
				RunAsGroup:   c.SecurityContext.RunAsGroup,
				RunAsNonRoot: c.SecurityContext.RunAsNonRoot,
				Privileged:   c.SecurityContext.Privileged,
				ReadOnlyRoot: c.SecurityContext.ReadOnlyRootFilesystem,
			}
		}

		// Get status from status map
		if cs, ok := statusMap[c.Name]; ok {
			ci.Ready = cs.Ready
			ci.RestartCount = cs.RestartCount
			restarts += cs.RestartCount

			if cs.State.Running != nil {
				ci.State = "Running"
				ci.StartedAt = cs.State.Running.StartedAt.Format("2006-01-02 15:04:05")
			} else if cs.State.Waiting != nil {
				ci.State = "Waiting"
				ci.Reason = cs.State.Waiting.Reason
				ci.Message = cs.State.Waiting.Message
			} else if cs.State.Terminated != nil {
				ci.State = "Terminated"
				ci.Reason = cs.State.Terminated.Reason
				ci.Message = cs.State.Terminated.Message
				ci.ExitCode = &cs.State.Terminated.ExitCode
				ci.StartedAt = cs.State.Terminated.StartedAt.Format("2006-01-02 15:04:05")
				ci.FinishedAt = cs.State.Terminated.FinishedAt.Format("2006-01-02 15:04:05")
			}
		}

		containers = append(containers, ci)
	}

	// Parse init containers
	var initContainers []ContainerInfo
	initStatusMap := make(map[string]corev1.ContainerStatus)
	for _, cs := range p.Status.InitContainerStatuses {
		initStatusMap[cs.Name] = cs
	}
	for _, c := range p.Spec.InitContainers {
		ci := ContainerInfo{
			Name:            c.Name,
			Image:           c.Image,
			ImagePullPolicy: string(c.ImagePullPolicy),
		}
		if cs, ok := initStatusMap[c.Name]; ok {
			ci.Ready = cs.Ready
			ci.RestartCount = cs.RestartCount
			if cs.State.Running != nil {
				ci.State = "Running"
			} else if cs.State.Waiting != nil {
				ci.State = "Waiting"
				ci.Reason = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				ci.State = "Terminated"
				ci.Reason = cs.State.Terminated.Reason
				ci.ExitCode = &cs.State.Terminated.ExitCode
			}
		}
		initContainers = append(initContainers, ci)
	}

	ready := 0
	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
	}

	var ownerRef, ownerKind string
	if len(p.OwnerReferences) > 0 {
		ownerRef = p.OwnerReferences[0].Name
		ownerKind = p.OwnerReferences[0].Kind
	}

	// Parse volumes
	var volumes []VolumeInfo
	for _, v := range p.Spec.Volumes {
		vi := VolumeInfo{Name: v.Name}
		switch {
		case v.ConfigMap != nil:
			vi.Type = "ConfigMap"
			vi.Source = v.ConfigMap.Name
		case v.Secret != nil:
			vi.Type = "Secret"
			vi.Source = v.Secret.SecretName
		case v.PersistentVolumeClaim != nil:
			vi.Type = "PVC"
			vi.Source = v.PersistentVolumeClaim.ClaimName
		case v.EmptyDir != nil:
			vi.Type = "EmptyDir"
		case v.HostPath != nil:
			vi.Type = "HostPath"
			vi.Source = v.HostPath.Path
		case v.Projected != nil:
			vi.Type = "Projected"
		case v.DownwardAPI != nil:
			vi.Type = "DownwardAPI"
		default:
			vi.Type = "Other"
		}
		volumes = append(volumes, vi)
	}

	// Parse tolerations
	var tolerations []TolerationInfo
	for _, t := range p.Spec.Tolerations {
		tolerations = append(tolerations, TolerationInfo{
			Key:      t.Key,
			Operator: string(t.Operator),
			Value:    t.Value,
			Effect:   string(t.Effect),
		})
	}

	// Get termination grace period
	var terminationGrace int64 = 30 // default
	if p.Spec.TerminationGracePeriodSeconds != nil {
		terminationGrace = *p.Spec.TerminationGracePeriodSeconds
	}

	// Get start time
	var startTime string
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Format("2006-01-02 15:04:05")
	}

	return PodInfo{
		Name:                   p.Name,
		Namespace:              p.Namespace,
		Node:                   p.Spec.NodeName,
		Status:                 getPodStatus(p),
		Ready:                  fmt.Sprintf("%d/%d", ready, len(p.Spec.Containers)),
		Restarts:               restarts,
		Age:                    formatAge(p.CreationTimestamp.Time),
		IP:                     p.Status.PodIP,
		HostIP:                 p.Status.HostIP,
		Labels:                 p.Labels,
		Annotations:            p.Annotations,
		Containers:             containers,
		InitContainers:         initContainers,
		Conditions:             p.Status.Conditions,
		Phase:                  p.Status.Phase,
		OwnerRef:               ownerRef,
		OwnerKind:              ownerKind,
		QoSClass:               string(p.Status.QOSClass),
		ServiceAccount:         p.Spec.ServiceAccountName,
		Volumes:                volumes,
		RestartPolicy:          string(p.Spec.RestartPolicy),
		DNSPolicy:              string(p.Spec.DNSPolicy),
		PriorityClassName:      p.Spec.PriorityClassName,
		Priority:               p.Spec.Priority,
		NodeSelector:           p.Spec.NodeSelector,
		Tolerations:            tolerations,
		TerminationGracePeriod: terminationGrace,
		StartTime:              startTime,
	}
}

func parseProbe(probe *corev1.Probe) *ProbeInfo {
	if probe == nil {
		return nil
	}

	pi := &ProbeInfo{
		InitialDelay:     probe.InitialDelaySeconds,
		Period:           probe.PeriodSeconds,
		Timeout:          probe.TimeoutSeconds,
		SuccessThreshold: probe.SuccessThreshold,
		FailureThreshold: probe.FailureThreshold,
	}

	if probe.HTTPGet != nil {
		pi.Type = "HTTP"
		pi.Path = probe.HTTPGet.Path
		pi.Port = probe.HTTPGet.Port.IntVal
		pi.Scheme = string(probe.HTTPGet.Scheme)
	} else if probe.TCPSocket != nil {
		pi.Type = "TCP"
		pi.Port = probe.TCPSocket.Port.IntVal
	} else if probe.Exec != nil {
		pi.Type = "Exec"
		pi.Command = probe.Exec.Command
	} else if probe.GRPC != nil {
		pi.Type = "gRPC"
		pi.Port = probe.GRPC.Port
	}

	return pi
}

func getPodStatus(p *corev1.Pod) string {
	if p.DeletionTimestamp != nil {
		return "Terminating"
	}

	for _, cs := range p.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			if cs.State.Waiting.Reason != "" {
				return cs.State.Waiting.Reason
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}

	return string(p.Status.Phase)
}

type RelatedResources struct {
	Services        []ServiceInfo
	Ingresses       []IngressInfo
	VirtualServices []VirtualServiceInfo
	Gateways        []GatewayInfo
	ConfigMaps      []string
	Secrets         []string
	Owner           *OwnerInfo
}

type GatewayInfo struct {
	Name      string
	Namespace string
	Servers   []GatewayServer
}

type GatewayServer struct {
	Port     int32
	Protocol string
	Hosts    []string
	TLS      string // SIMPLE, MUTUAL, PASSTHROUGH, etc
}

type ServiceInfo struct {
	Name      string
	Type      string
	ClusterIP string
	Ports     string
	Endpoints int
}

type IngressInfo struct {
	Name        string
	Class       string   // Ingress class (nginx, traefik, istio, etc)
	Hosts       []string
	TLS         bool
	TLSSecrets  []string
	Rules       []IngressRuleInfo
	Annotations map[string]string // Important annotations for debugging
}

type IngressRuleInfo struct {
	Host    string
	Paths   []IngressPathInfo
}

type IngressPathInfo struct {
	Path        string
	PathType    string
	ServiceName string
	ServicePort string
}

type VirtualServiceInfo struct {
	Name     string
	Hosts    []string
	Gateways []string
	Routes   []VirtualServiceRoute
}

type VirtualServiceRoute struct {
	Match       string // Match conditions summary
	Destination string // Service destination
	Port        int32
	Weight      int32
}

type OwnerInfo struct {
	Kind         string
	Name         string
	WorkloadKind string // Parent of ReplicaSet (Deployment, etc)
	WorkloadName string
}

// GetRelatedResources discovers resources related to a pod.
// Returns services, ingresses, VirtualServices, gateways, ConfigMaps, and Secrets
// that are connected to the pod through labels or volume mounts.
func GetRelatedResources(ctx context.Context, clientset *kubernetes.Clientset, dynamicClient dynamic.Interface, pod PodInfo) (*RelatedResources, error) {
	related := &RelatedResources{}

	if pod.OwnerRef != "" {
		related.Owner = &OwnerInfo{
			Kind: pod.OwnerKind,
			Name: pod.OwnerRef,
		}

		// If owner is ReplicaSet, fetch the parent workload (Deployment, Rollout, etc)
		if pod.OwnerKind == "ReplicaSet" {
			rs, err := clientset.AppsV1().ReplicaSets(pod.Namespace).Get(ctx, pod.OwnerRef, metav1.GetOptions{})
			if err == nil && len(rs.OwnerReferences) > 0 {
				related.Owner.WorkloadKind = rs.OwnerReferences[0].Kind
				related.Owner.WorkloadName = rs.OwnerReferences[0].Name
			}
		}
	}

	svcs, err := clientset.CoreV1().Services(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, svc := range svcs.Items {
			if svc.Spec.Selector == nil {
				continue
			}
			if labelsMatch(svc.Spec.Selector, pod.Labels) {
				var ports []string
				for _, p := range svc.Spec.Ports {
					ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol))
				}

				// Use EndpointSlice instead of deprecated Endpoints API
				endpointCount := 0
				epSlices, _ := clientset.DiscoveryV1().EndpointSlices(pod.Namespace).List(ctx, metav1.ListOptions{
					LabelSelector: discoveryv1.LabelServiceName + "=" + svc.Name,
				})
				if epSlices != nil {
					for _, slice := range epSlices.Items {
						for _, endpoint := range slice.Endpoints {
							if endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready {
								endpointCount++
							}
						}
					}
				}

				related.Services = append(related.Services, ServiceInfo{
					Name:      svc.Name,
					Type:      string(svc.Spec.Type),
					ClusterIP: svc.Spec.ClusterIP,
					Ports:     strings.Join(ports, ", "),
					Endpoints: endpointCount,
				})
			}
		}
	}

	ings, err := clientset.NetworkingV1().Ingresses(pod.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, svc := range related.Services {
			for _, ing := range ings.Items {
				if ingressReferencesService(ing, svc.Name) {
					ingInfo := IngressInfo{
						Name:        ing.Name,
						Annotations: make(map[string]string),
					}

					// Get ingress class
					if ing.Spec.IngressClassName != nil {
						ingInfo.Class = *ing.Spec.IngressClassName
					} else if class, ok := ing.Annotations["kubernetes.io/ingress.class"]; ok {
						ingInfo.Class = class
					}

					// Get TLS info
					for _, tls := range ing.Spec.TLS {
						ingInfo.TLS = true
						if tls.SecretName != "" {
							ingInfo.TLSSecrets = append(ingInfo.TLSSecrets, tls.SecretName)
						}
						ingInfo.Hosts = append(ingInfo.Hosts, tls.Hosts...)
					}

					// Get rules with detailed path info
					for _, rule := range ing.Spec.Rules {
						if rule.Host != "" && !contains(ingInfo.Hosts, rule.Host) {
							ingInfo.Hosts = append(ingInfo.Hosts, rule.Host)
						}
						ruleInfo := IngressRuleInfo{Host: rule.Host}
						if rule.HTTP != nil {
							for _, p := range rule.HTTP.Paths {
								pathType := "Prefix"
								if p.PathType != nil {
									pathType = string(*p.PathType)
								}
								svcPort := ""
								if p.Backend.Service != nil {
									if p.Backend.Service.Port.Name != "" {
										svcPort = p.Backend.Service.Port.Name
									} else {
										svcPort = fmt.Sprintf("%d", p.Backend.Service.Port.Number)
									}
								}
								ruleInfo.Paths = append(ruleInfo.Paths, IngressPathInfo{
									Path:        p.Path,
									PathType:    pathType,
									ServiceName: p.Backend.Service.Name,
									ServicePort: svcPort,
								})
							}
						}
						ingInfo.Rules = append(ingInfo.Rules, ruleInfo)
					}

					// Extract important annotations for debugging
					debugAnnotations := []string{
						"nginx.ingress.kubernetes.io/rewrite-target",
						"nginx.ingress.kubernetes.io/ssl-redirect",
						"nginx.ingress.kubernetes.io/proxy-body-size",
						"nginx.ingress.kubernetes.io/proxy-read-timeout",
						"nginx.ingress.kubernetes.io/proxy-send-timeout",
						"nginx.ingress.kubernetes.io/backend-protocol",
						"nginx.ingress.kubernetes.io/cors-allow-origin",
						"nginx.ingress.kubernetes.io/whitelist-source-range",
						"nginx.ingress.kubernetes.io/limit-rps",
						"traefik.ingress.kubernetes.io/router.entrypoints",
						"traefik.ingress.kubernetes.io/router.middlewares",
						"cert-manager.io/cluster-issuer",
						"cert-manager.io/issuer",
						"external-dns.alpha.kubernetes.io/hostname",
					}
					for _, key := range debugAnnotations {
						if val, ok := ing.Annotations[key]; ok {
							ingInfo.Annotations[key] = val
						}
					}

					related.Ingresses = append(related.Ingresses, ingInfo)
				}
			}
		}
	}

	// Fetch Istio VirtualServices and Gateways using dynamic client
	if dynamicClient != nil {
		related.VirtualServices, related.Gateways = getIstioResources(ctx, dynamicClient, pod.Namespace, related.Services)
	}

	podObj, err := clientset.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
	if err == nil {
		for _, vol := range podObj.Spec.Volumes {
			if vol.ConfigMap != nil {
				related.ConfigMaps = append(related.ConfigMaps, vol.ConfigMap.Name)
			}
			if vol.Secret != nil {
				related.Secrets = append(related.Secrets, vol.Secret.SecretName)
			}
		}
		for _, c := range podObj.Spec.Containers {
			for _, env := range c.EnvFrom {
				if env.ConfigMapRef != nil {
					related.ConfigMaps = append(related.ConfigMaps, env.ConfigMapRef.Name)
				}
				if env.SecretRef != nil {
					related.Secrets = append(related.Secrets, env.SecretRef.Name)
				}
			}
		}
	}

	return related, nil
}

func labelsMatch(selector, labels map[string]string) bool {
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}

func ingressReferencesService(ing networkingv1.Ingress, svcName string) bool {
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			if path.Backend.Service != nil && path.Backend.Service.Name == svcName {
				return true
			}
		}
	}
	return false
}

func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// getIstioResources fetches Istio VirtualServices and Gateways using dynamic client
func getIstioResources(ctx context.Context, dynamicClient dynamic.Interface, namespace string, services []ServiceInfo) ([]VirtualServiceInfo, []GatewayInfo) {
	var virtualServices []VirtualServiceInfo
	var gateways []GatewayInfo
	gatewaySet := make(map[string]bool) // Track which gateways we need to fetch

	// Define VirtualService GVR
	vsGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "virtualservices",
	}

	// Fetch VirtualServices in the namespace
	vsList, err := dynamicClient.Resource(vsGVR).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		// Istio might not be installed, just return empty
		return virtualServices, gateways
	}

	serviceNames := make(map[string]bool)
	for _, svc := range services {
		serviceNames[svc.Name] = true
	}

	for _, vs := range vsList.Items {
		vsInfo := VirtualServiceInfo{
			Name: vs.GetName(),
		}

		// Extract hosts
		if spec, ok := vs.Object["spec"].(map[string]interface{}); ok {
			if hosts, ok := spec["hosts"].([]interface{}); ok {
				for _, h := range hosts {
					if host, ok := h.(string); ok {
						vsInfo.Hosts = append(vsInfo.Hosts, host)
					}
				}
			}

			// Extract gateways
			if gws, ok := spec["gateways"].([]interface{}); ok {
				for _, g := range gws {
					if gw, ok := g.(string); ok {
						vsInfo.Gateways = append(vsInfo.Gateways, gw)
						gatewaySet[gw] = true
					}
				}
			}

			// Extract HTTP routes
			if http, ok := spec["http"].([]interface{}); ok {
				for _, route := range http {
					if routeMap, ok := route.(map[string]interface{}); ok {
						vsRoute := VirtualServiceRoute{}

						// Extract match conditions
						if matches, ok := routeMap["match"].([]interface{}); ok && len(matches) > 0 {
							if match, ok := matches[0].(map[string]interface{}); ok {
								if uri, ok := match["uri"].(map[string]interface{}); ok {
									for matchType, val := range uri {
										vsRoute.Match = fmt.Sprintf("%s: %v", matchType, val)
										break
									}
								}
							}
						}
						if vsRoute.Match == "" {
							vsRoute.Match = "/*"
						}

						// Extract route destinations
						if routeDests, ok := routeMap["route"].([]interface{}); ok {
							for _, rd := range routeDests {
								if rdMap, ok := rd.(map[string]interface{}); ok {
									if dest, ok := rdMap["destination"].(map[string]interface{}); ok {
										if host, ok := dest["host"].(string); ok {
											vsRoute.Destination = host
											// Check if this VS routes to one of our services
											shortHost := strings.Split(host, ".")[0]
											if serviceNames[shortHost] || serviceNames[host] {
												// This VS is relevant to our pod
											}
										}
										if port, ok := dest["port"].(map[string]interface{}); ok {
											if number, ok := port["number"].(int64); ok {
												vsRoute.Port = int32(number)
											} else if number, ok := port["number"].(float64); ok {
												vsRoute.Port = int32(number)
											}
										}
									}
									if weight, ok := rdMap["weight"].(int64); ok {
										vsRoute.Weight = int32(weight)
									} else if weight, ok := rdMap["weight"].(float64); ok {
										vsRoute.Weight = int32(weight)
									}
								}
							}
						}

						vsInfo.Routes = append(vsInfo.Routes, vsRoute)
					}
				}
			}
		}

		// Only include VirtualServices that route to our services
		isRelevant := false
		for _, route := range vsInfo.Routes {
			shortHost := strings.Split(route.Destination, ".")[0]
			if serviceNames[shortHost] || serviceNames[route.Destination] {
				isRelevant = true
				break
			}
		}
		if isRelevant {
			virtualServices = append(virtualServices, vsInfo)
		}
	}

	// Fetch referenced Gateways
	gwGVR := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1beta1",
		Resource: "gateways",
	}

	for gwRef := range gatewaySet {
		// Parse gateway reference (can be "namespace/name" or just "name")
		gwNamespace := namespace
		gwName := gwRef
		if strings.Contains(gwRef, "/") {
			parts := strings.SplitN(gwRef, "/", 2)
			gwNamespace = parts[0]
			gwName = parts[1]
		}

		gw, err := dynamicClient.Resource(gwGVR).Namespace(gwNamespace).Get(ctx, gwName, metav1.GetOptions{})
		if err != nil {
			continue
		}

		gwInfo := GatewayInfo{
			Name:      gw.GetName(),
			Namespace: gwNamespace,
		}

		if spec, ok := gw.Object["spec"].(map[string]interface{}); ok {
			if servers, ok := spec["servers"].([]interface{}); ok {
				for _, srv := range servers {
					if srvMap, ok := srv.(map[string]interface{}); ok {
						server := GatewayServer{}

						if port, ok := srvMap["port"].(map[string]interface{}); ok {
							if number, ok := port["number"].(int64); ok {
								server.Port = int32(number)
							} else if number, ok := port["number"].(float64); ok {
								server.Port = int32(number)
							}
							if protocol, ok := port["protocol"].(string); ok {
								server.Protocol = protocol
							}
						}

						if hosts, ok := srvMap["hosts"].([]interface{}); ok {
							for _, h := range hosts {
								if host, ok := h.(string); ok {
									server.Hosts = append(server.Hosts, host)
								}
							}
						}

						if tls, ok := srvMap["tls"].(map[string]interface{}); ok {
							if mode, ok := tls["mode"].(string); ok {
								server.TLS = mode
							}
						}

						gwInfo.Servers = append(gwInfo.Servers, server)
					}
				}
			}
		}

		gateways = append(gateways, gwInfo)
	}

	return virtualServices, gateways
}

func GetDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.Deployment, error) {
	return clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.StatefulSet, error) {
	return clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*appsv1.DaemonSet, error) {
	return clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
}

func GetJob(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) (*batchv1.Job, error) {
	return clientset.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

func DeletePod(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	return clientset.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func ScaleDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
	scale, err := clientset.AppsV1().Deployments(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().Deployments(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func ScaleStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, replicas int32) error {
	scale, err := clientset.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func RestartDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	deploy, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if deploy.Spec.Template.Annotations == nil {
		deploy.Spec.Template.Annotations = make(map[string]string)
	}
	deploy.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deploy, metav1.UpdateOptions{})
	return err
}

func RestartStatefulSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	sts, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

func RestartDaemonSet(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string) error {
	ds, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}
	ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")

	_, err = clientset.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
	return err
}
