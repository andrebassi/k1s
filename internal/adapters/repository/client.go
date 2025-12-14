// Package k8s provides Kubernetes client operations for k1s.
//
// This package encapsulates all Kubernetes API interactions including:
//   - Cluster connection and authentication
//   - Resource listing and retrieval (pods, deployments, services, etc.)
//   - Container logs and events
//   - Metrics collection from metrics-server
//   - Workload management (scale, restart, delete)
//
// The package uses the official Kubernetes client-go library and supports
// both kubeconfig-based authentication and in-cluster service account tokens.
package repository

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client wraps the Kubernetes clientset with additional functionality.
// It provides a unified interface for interacting with the Kubernetes API,
// including standard resources, custom resources (via dynamic client), and metrics.
type Client struct {
	clientset     *kubernetes.Clientset
	metricsClient *metricsv.Clientset
	dynamicClient dynamic.Interface
	config        *rest.Config
	context       string
	namespace     string
}

// NewClient creates a new Kubernetes client using the default kubeconfigs.
// It first attempts to use ~/.kube/config, falling back to in-cluster config
// if running inside a Kubernetes cluster. The client includes:
//   - Standard Kubernetes clientset for core resources
//   - Metrics client for CPU/memory usage data
//   - Dynamic client for custom resources (e.g., Istio VirtualServices)
func NewClient() (*Client, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
		}
	}

	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Metrics client may fail if metrics-server is not installed
	metricsClient, _ := metricsv.NewForConfig(config)

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	rawConfig, _ := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	currentContext := ""
	if rawConfig != nil {
		currentContext = rawConfig.CurrentContext
	}

	return &Client{
		clientset:     clientset,
		metricsClient: metricsClient,
		dynamicClient: dynamicClient,
		config:        config,
		context:       currentContext,
		namespace:     "default",
	}, nil
}

// DynamicClient returns the dynamic client for custom resource operations.
// Use this for Istio resources, custom CRDs, and other non-standard resources.
func (c *Client) DynamicClient() dynamic.Interface {
	return c.dynamicClient
}

// Clientset returns the standard Kubernetes clientset.
// Use this for core Kubernetes resources (pods, services, deployments, etc.).
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

// MetricsClient returns the metrics client for resource usage data.
// May return nil if metrics-server is not available in the cluster.
func (c *Client) MetricsClient() *metricsv.Clientset {
	return c.metricsClient
}

// Context returns the current Kubernetes context name.
func (c *Client) Context() string {
	return c.context
}

// Namespace returns the currently selected namespace.
func (c *Client) Namespace() string {
	return c.namespace
}

// SetNamespace changes the currently selected namespace.
func (c *Client) SetNamespace(ns string) {
	c.namespace = ns
}

// ListNamespaces returns all namespace names in the cluster, sorted alphabetically.
func (c *Client) ListNamespaces(ctx context.Context) ([]string, error) {
	return ListNamespaces(ctx, c.clientset)
}

// ListContexts returns all available Kubernetes contexts from kubeconfig
// along with the currently active context name.
func (c *Client) ListContexts() ([]string, string, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	config, err := rules.Load()
	if err != nil {
		return nil, "", err
	}

	var contexts []string
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	return contexts, config.CurrentContext, nil
}

// DeletePod deletes a pod by name in the specified namespace.
func (c *Client) DeletePod(ctx context.Context, namespace, name string) error {
	return DeletePod(ctx, c.clientset, namespace, name)
}

// ScaleWorkload scales a workload (Deployment or StatefulSet) to the specified replica count.
// DaemonSets, Jobs, and CronJobs cannot be scaled and will return nil without error.
func (c *Client) ScaleWorkload(ctx context.Context, namespace, name string, resourceType ResourceType, replicas int32) error {
	switch resourceType {
	case ResourceDeployments:
		return ScaleDeployment(ctx, c.clientset, namespace, name, replicas)
	case ResourceStatefulSets:
		return ScaleStatefulSet(ctx, c.clientset, namespace, name, replicas)
	default:
		return nil // DaemonSets, Jobs, CronJobs cannot be scaled
	}
}

// RestartWorkload triggers a rolling restart of the specified workload.
// This is done by updating the pod template annotation, forcing new pods to be created.
// Jobs and CronJobs do not support restart and will return nil without error.
func (c *Client) RestartWorkload(ctx context.Context, namespace, name string, resourceType ResourceType) error {
	switch resourceType {
	case ResourceDeployments:
		return RestartDeployment(ctx, c.clientset, namespace, name)
	case ResourceStatefulSets:
		return RestartStatefulSet(ctx, c.clientset, namespace, name)
	case ResourceDaemonSets:
		return RestartDaemonSet(ctx, c.clientset, namespace, name)
	default:
		return nil // Jobs and CronJobs don't have restart concept
	}
}
