package usecase

import (
	"context"

	"github.com/andrebassi/k1s/internal/domain/entity"
	"github.com/andrebassi/k1s/internal/domain/port"
)

// WorkloadUseCase handles workload-related operations.
type WorkloadUseCase struct {
	repo port.KubernetesRepository
}

// NewWorkloadUseCase creates a new WorkloadUseCase.
func NewWorkloadUseCase(repo port.KubernetesRepository) *WorkloadUseCase {
	return &WorkloadUseCase{repo: repo}
}

// ListWorkloads returns all workloads of a given type in a namespace.
func (uc *WorkloadUseCase) ListWorkloads(ctx context.Context, namespace string, resourceType entity.ResourceType) ([]entity.WorkloadInfo, error) {
	return uc.repo.ListWorkloads(ctx, namespace, resourceType)
}

// GetWorkloadPods returns all pods belonging to a workload.
func (uc *WorkloadUseCase) GetWorkloadPods(ctx context.Context, workload entity.WorkloadInfo) ([]entity.PodInfo, error) {
	return uc.repo.GetWorkloadPods(ctx, workload)
}

// ScaleWorkload scales a workload to the specified replica count.
func (uc *WorkloadUseCase) ScaleWorkload(ctx context.Context, namespace, name string, resourceType entity.ResourceType, replicas int32) error {
	switch resourceType {
	case entity.ResourceDeployments:
		return uc.repo.ScaleDeployment(ctx, namespace, name, replicas)
	case entity.ResourceStatefulSets:
		return uc.repo.ScaleStatefulSet(ctx, namespace, name, replicas)
	default:
		return nil // DaemonSets, Jobs, CronJobs cannot be scaled
	}
}

// RestartWorkload triggers a rolling restart of a workload.
func (uc *WorkloadUseCase) RestartWorkload(ctx context.Context, namespace, name string, resourceType entity.ResourceType) error {
	switch resourceType {
	case entity.ResourceDeployments:
		return uc.repo.RestartDeployment(ctx, namespace, name)
	case entity.ResourceStatefulSets:
		return uc.repo.RestartStatefulSet(ctx, namespace, name)
	case entity.ResourceDaemonSets:
		return uc.repo.RestartDaemonSet(ctx, namespace, name)
	default:
		return nil // Jobs and CronJobs don't have restart concept
	}
}
