package usecase

import (
	"context"

	"github.com/andrebassi/k1s/internal/domain/entity"
	"github.com/andrebassi/k1s/internal/domain/port"
)

// NamespaceUseCase handles namespace-related operations.
type NamespaceUseCase struct {
	repo port.KubernetesRepository
}

// NewNamespaceUseCase creates a new NamespaceUseCase.
func NewNamespaceUseCase(repo port.KubernetesRepository) *NamespaceUseCase {
	return &NamespaceUseCase{repo: repo}
}

// ListNamespaces returns all namespace names in the cluster.
func (uc *NamespaceUseCase) ListNamespaces(ctx context.Context) ([]string, error) {
	return uc.repo.ListNamespaces(ctx)
}

// NamespaceResources holds all resources in a namespace.
type NamespaceResources struct {
	Pods       []entity.PodInfo
	ConfigMaps []entity.ConfigMapInfo
	Secrets    []entity.SecretInfo
}

// GetNamespaceResources retrieves pods, configmaps, and secrets for a namespace.
func (uc *NamespaceUseCase) GetNamespaceResources(ctx context.Context, namespace string) (*NamespaceResources, error) {
	resources := &NamespaceResources{}

	// Get pods
	workloads, err := uc.repo.ListWorkloads(ctx, namespace, entity.ResourcePods)
	if err != nil {
		return nil, err
	}

	for _, w := range workloads {
		pod, err := uc.repo.GetPod(ctx, namespace, w.Name)
		if err == nil && pod != nil {
			resources.Pods = append(resources.Pods, *pod)
		}
	}

	// Get configmaps
	configmaps, _ := uc.repo.ListConfigMaps(ctx, namespace)
	resources.ConfigMaps = configmaps

	// Get secrets
	secrets, _ := uc.repo.ListSecrets(ctx, namespace)
	resources.Secrets = secrets

	return resources, nil
}

// GetConfigMapData retrieves the full data of a ConfigMap.
func (uc *NamespaceUseCase) GetConfigMapData(ctx context.Context, namespace, name string) (*entity.ConfigMapData, error) {
	return uc.repo.GetConfigMapData(ctx, namespace, name)
}

// GetSecretData retrieves the full data of a Secret.
func (uc *NamespaceUseCase) GetSecretData(ctx context.Context, namespace, name string) (*entity.SecretData, error) {
	return uc.repo.GetSecretData(ctx, namespace, name)
}
