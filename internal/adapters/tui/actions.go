// Package tui provides the terminal user interface for k1s.
// This file contains all action commands that perform operations
// on Kubernetes resources (delete, scale, restart, copy).
package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/andrebassi/k1s/internal/adapters/repository"
	"github.com/andrebassi/k1s/internal/adapters/tui/component"
)

// deletePod deletes a pod from the cluster.
// This is an async operation that returns a podDeletedMsg when complete.
// The pod is deleted using the Kubernetes API with default grace period.
// Returns a podDeletedMsg with the result (success or error).
func (m *Model) deletePod(namespace, podName string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.DeletePod(ctx, namespace, podName)
		return podDeletedMsg{
			namespace: namespace,
			podName:   podName,
			err:       err,
		}
	}
}

// scaleWorkload scales a workload to the specified number of replicas.
// Supports Deployments, StatefulSets, and Argo Rollouts.
// This is an async operation that triggers a rolling update if scaling up,
// or terminates pods if scaling down.
// Returns a workloadActionMsg with the scale action result.
func (m *Model) scaleWorkload(workload *repository.WorkloadInfo, replicas int32) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.ScaleWorkload(ctx, workload.Namespace, workload.Name, workload.Type, replicas)
		return workloadActionMsg{
			action:       "scale",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			replicas:     replicas,
			err:          err,
		}
	}
}

// restartWorkload performs a rolling restart of a workload.
// This is done by patching the pod template annotation with the current timestamp,
// which triggers Kubernetes to recreate all pods.
// Supports Deployments, StatefulSets, and Argo Rollouts.
// Returns a workloadActionMsg with the restart action result.
func (m *Model) restartWorkload(workload *repository.WorkloadInfo) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := m.k8sClient.RestartWorkload(ctx, workload.Namespace, workload.Name, workload.Type)
		return workloadActionMsg{
			action:       "restart",
			workloadName: workload.Name,
			namespace:    workload.Namespace,
			resourceType: workload.Type,
			err:          err,
		}
	}
}

// copySecretToSingleNamespace copies a secret to a target namespace.
// This function handles both single namespace copy and batch copy progress.
// When copying to multiple namespaces, it processes one at a time with a 300ms delay
// between each to provide visual feedback to the user.
//
// Parameters:
//   - sourceNs: source namespace where the secret exists
//   - secretName: name of the secret to copy
//   - targetNs: target namespace to copy to
//   - remaining: list of remaining namespaces for batch copy (nil for single copy)
//   - successCount: number of successful copies so far
//   - errorCount: number of failed copies so far
//
// Returns SecretCopyProgress if more namespaces remain, or SecretCopyResult when done.
func (m *Model) copySecretToSingleNamespace(sourceNs, secretName, targetNs string, remaining []string, successCount, errorCount int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Small delay so user can see the namespace name
		time.Sleep(300 * time.Millisecond)

		// Copy to current namespace
		err := repository.CopySecretToNamespace(ctx, m.k8sClient.Clientset(), sourceNs, secretName, targetNs)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}

		// If no more remaining, return final result
		if len(remaining) == 0 {
			if errorCount > 0 {
				return component.SecretCopyResult{
					Success: false,
					Message: fmt.Sprintf("Copied to %d namespaces, %d failed", successCount, errorCount),
				}
			}
			if successCount == 1 {
				return component.SecretCopyResult{
					Success: true,
					Message: fmt.Sprintf("Copied to %s", targetNs),
				}
			}
			return component.SecretCopyResult{
				Success: true,
				Message: fmt.Sprintf("Copied to %d namespaces", successCount),
			}
		}

		// Send progress for next namespace
		next := remaining[0]
		newRemaining := remaining[1:]
		return component.SecretCopyProgress{
			SecretName:       secretName,
			SourceNamespace:  sourceNs,
			CurrentNamespace: next,
			Remaining:        newRemaining,
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
		}
	}
}

// copyConfigMapToSingleNamespace copies a ConfigMap to a target namespace.
// This function handles both single namespace copy and batch copy progress.
// When copying to multiple namespaces, it processes one at a time with a 300ms delay
// between each to provide visual feedback to the user.
//
// Parameters:
//   - sourceNs: source namespace where the ConfigMap exists
//   - configMapName: name of the ConfigMap to copy
//   - targetNs: target namespace to copy to
//   - remaining: list of remaining namespaces for batch copy (nil for single copy)
//   - successCount: number of successful copies so far
//   - errorCount: number of failed copies so far
//
// Returns ConfigMapCopyProgress if more namespaces remain, or ConfigMapCopyResult when done.
func (m *Model) copyConfigMapToSingleNamespace(sourceNs, configMapName, targetNs string, remaining []string, successCount, errorCount int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Small delay so user can see the namespace name
		time.Sleep(300 * time.Millisecond)

		// Copy to current namespace
		err := repository.CopyConfigMapToNamespace(ctx, m.k8sClient.Clientset(), sourceNs, configMapName, targetNs)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}

		// If no more remaining, return final result
		if len(remaining) == 0 {
			if errorCount > 0 {
				return component.ConfigMapCopyResult{
					Success: false,
					Message: fmt.Sprintf("Copied to %d namespaces, %d failed", successCount, errorCount),
				}
			}
			if successCount == 1 {
				return component.ConfigMapCopyResult{
					Success: true,
					Message: fmt.Sprintf("Copied to %s", targetNs),
				}
			}
			return component.ConfigMapCopyResult{
				Success: true,
				Message: fmt.Sprintf("Copied to %d namespaces", successCount),
			}
		}

		// Send progress for next namespace
		next := remaining[0]
		newRemaining := remaining[1:]
		return component.ConfigMapCopyProgress{
			ConfigMapName:    configMapName,
			SourceNamespace:  sourceNs,
			CurrentNamespace: next,
			Remaining:        newRemaining,
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
		}
	}
}

// copyDockerRegistryToSingleNamespace copies a Docker Registry secret to a target namespace.
// Docker Registry secrets (type kubernetes.io/dockerconfigjson) contain image pull credentials.
// This function uses the same underlying copy mechanism as regular secrets.
//
// When copying to multiple namespaces, it processes one at a time with a 300ms delay
// between each to provide visual feedback to the user.
//
// Parameters:
//   - sourceNs: source namespace where the Docker Registry secret exists
//   - secretName: name of the Docker Registry secret to copy
//   - targetNs: target namespace to copy to
//   - remaining: list of remaining namespaces for batch copy (nil for single copy)
//   - successCount: number of successful copies so far
//   - errorCount: number of failed copies so far
//
// Returns DockerRegistryCopyProgress if more namespaces remain, or DockerRegistryCopyResult when done.
func (m *Model) copyDockerRegistryToSingleNamespace(sourceNs, secretName, targetNs string, remaining []string, successCount, errorCount int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Small delay so user can see the namespace name
		time.Sleep(300 * time.Millisecond)

		// Copy to current namespace (Docker Registry secrets are just secrets)
		err := repository.CopySecretToNamespace(ctx, m.k8sClient.Clientset(), sourceNs, secretName, targetNs)
		if err != nil {
			errorCount++
		} else {
			successCount++
		}

		// If no more remaining, return final result
		if len(remaining) == 0 {
			if errorCount > 0 {
				return component.DockerRegistryCopyResult{
					Success: false,
					Message: fmt.Sprintf("Copied to %d namespaces, %d failed", successCount, errorCount),
				}
			}
			if successCount == 1 {
				return component.DockerRegistryCopyResult{
					Success: true,
					Message: fmt.Sprintf("Copied to %s", targetNs),
				}
			}
			return component.DockerRegistryCopyResult{
				Success: true,
				Message: fmt.Sprintf("Copied to %d namespaces", successCount),
			}
		}

		// Send progress for next namespace
		next := remaining[0]
		newRemaining := remaining[1:]
		return component.DockerRegistryCopyProgress{
			SecretName:       secretName,
			SourceNamespace:  sourceNs,
			CurrentNamespace: next,
			Remaining:        newRemaining,
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
		}
	}
}

// forceDeleteNamespace forcefully deletes a stuck namespace.
// This is an async operation that deletes all resources in the namespace,
// removes finalizers, and then deletes the namespace itself.
// Used for namespaces stuck in Terminating state.
// Returns a namespaceDeletedMsg with the result (success or error).
func (m *Model) forceDeleteNamespace(namespace string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := repository.ForceDeleteNamespace(ctx, m.k8sClient.Clientset(), m.k8sClient.DynamicClient(), namespace)
		return namespaceDeletedMsg{
			namespace: namespace,
			err:       err,
		}
	}
}

// saveConfig persists the current application configuration to disk.
// This includes user preferences like last namespace, resource type, and refresh interval.
// Errors are silently ignored as config save is non-critical.
func (m *Model) saveConfig() {
	_ = m.config.Save()
}
