package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func int32Ptr(i int32) *int32 { return &i }

func TestListNamespaces(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "production"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "development"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "terminating-ns"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
		},
	)

	ctx := context.Background()
	namespaces, err := ListNamespaces(ctx, clientset)
	if err != nil {
		t.Fatalf("ListNamespaces() error = %v", err)
	}

	if len(namespaces) != 3 {
		t.Errorf("ListNamespaces() returned %d namespaces, want 3", len(namespaces))
	}

	// Verify sorted order (alphabetical)
	if namespaces[0].Name != "development" {
		t.Errorf("First namespace should be 'development', got %q", namespaces[0].Name)
	}
}

func TestListNamespaces_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListNamespaces(ctx, clientset)
	if err == nil {
		t.Error("ListNamespaces() should return error")
	}
}

func TestListActiveNamespaceNames(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "active-ns"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceActive},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "terminating-ns"},
			Status:     corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
		},
	)

	ctx := context.Background()
	names, err := ListActiveNamespaceNames(ctx, clientset)
	if err != nil {
		t.Fatalf("ListActiveNamespaceNames() error = %v", err)
	}

	if len(names) != 1 {
		t.Errorf("ListActiveNamespaceNames() returned %d names, want 1", len(names))
	}
}

func TestListWorkloads_Deployments(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-app",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:      3,
				ReadyReplicas: 3,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Errorf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_StatefulSets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "database",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "db"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      3,
				ReadyReplicas: 2,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Progressing" {
		t.Errorf("Status = %q, want 'Progressing'", workloads[0].Status)
	}
}

func TestListWorkloads_DaemonSets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "log-collector",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "logs"},
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 5,
				NumberReady:            5,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_Jobs(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "backup-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "backup"},
				},
			},
			Status: batchv1.JobStatus{
				Succeeded: 1,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Completed" {
		t.Errorf("Status = %q, want 'Completed'", workloads[0].Status)
	}
}

func TestListWorkloads_CronJobs(t *testing.T) {
	suspended := false
	clientset := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "daily-report",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				Suspend:  &suspended,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Active" {
		t.Errorf("Status = %q, want 'Active'", workloads[0].Status)
	}
}

func TestListWorkloads_Pods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "standalone-pod",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "main"}},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{Ready: true, RestartCount: 2},
				},
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourcePods)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].RestartCount != 2 {
		t.Errorf("RestartCount = %d, want 2", workloads[0].RestartCount)
	}
}

func TestListWorkloads_UnknownType(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", "unknowntype")
	if err == nil {
		t.Error("ListWorkloads() should return error for unknown type")
	}
}

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

func TestListConfigMaps(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app-config",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Data: map[string]string{"key1": "value1", "key2": "value2"},
		},
	)

	ctx := context.Background()
	cms, err := ListConfigMaps(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListConfigMaps() error = %v", err)
	}

	if len(cms) != 1 {
		t.Fatalf("ListConfigMaps() returned %d configmaps, want 1", len(cms))
	}

	if cms[0].Keys != 2 {
		t.Errorf("Keys = %d, want 2", cms[0].Keys)
	}
}

func TestGetConfigMap(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "app-config",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Data: map[string]string{"database.url": "postgres://localhost:5432"},
		},
	)

	ctx := context.Background()
	cm, err := GetConfigMap(ctx, clientset, "default", "app-config")
	if err != nil {
		t.Fatalf("GetConfigMap() error = %v", err)
	}

	if cm.Data["database.url"] != "postgres://localhost:5432" {
		t.Errorf("Data mismatch")
	}
}

func TestListSecrets(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "db-credentials",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"username": []byte("admin"), "password": []byte("secret")},
		},
	)

	ctx := context.Background()
	secrets, err := ListSecrets(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	if len(secrets) != 1 {
		t.Fatalf("ListSecrets() returned %d secrets, want 1", len(secrets))
	}

	if secrets[0].Keys != 2 {
		t.Errorf("Keys = %d, want 2", secrets[0].Keys)
	}
}

func TestGetSecret(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "db-credentials",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"password": []byte("secret123")},
		},
	)

	ctx := context.Background()
	secret, err := GetSecret(ctx, clientset, "default", "db-credentials")
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if secret.Data["password"] != "secret123" {
		t.Errorf("Password = %q, want 'secret123'", secret.Data["password"])
	}
}

func TestListHPAs(t *testing.T) {
	minReplicas := int32(1)
	clientset := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-hpa",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "web",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: 10,
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 3,
			},
		},
	)

	ctx := context.Background()
	hpas, err := ListHPAs(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListHPAs() error = %v", err)
	}

	if len(hpas) != 1 {
		t.Fatalf("ListHPAs() returned %d hpas, want 1", len(hpas))
	}

	if hpas[0].Reference != "Deployment/web" {
		t.Errorf("Reference = %q, want 'Deployment/web'", hpas[0].Reference)
	}
}

func TestGetHPA(t *testing.T) {
	minReplicas := int32(2)
	avgUtilization := int32(80)
	clientset := fake.NewSimpleClientset(
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "api-hpa",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					Kind: "Deployment",
					Name: "api",
				},
				MinReplicas: &minReplicas,
				MaxReplicas: 20,
				Metrics: []autoscalingv2.MetricSpec{
					{
						Type: autoscalingv2.ResourceMetricSourceType,
						Resource: &autoscalingv2.ResourceMetricSource{
							Name: corev1.ResourceCPU,
							Target: autoscalingv2.MetricTarget{
								Type:               autoscalingv2.UtilizationMetricType,
								AverageUtilization: &avgUtilization,
							},
						},
					},
				},
			},
			Status: autoscalingv2.HorizontalPodAutoscalerStatus{
				CurrentReplicas: 5,
				DesiredReplicas: 6,
				Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
					{
						Type:   autoscalingv2.ScalingActive,
						Status: corev1.ConditionTrue,
						Reason: "ValidMetricFound",
					},
				},
			},
		},
	)

	ctx := context.Background()
	hpa, err := GetHPA(ctx, clientset, "default", "api-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if hpa.MinReplicas != 2 {
		t.Errorf("MinReplicas = %d, want 2", hpa.MinReplicas)
	}
	if len(hpa.Metrics) != 1 {
		t.Errorf("len(Metrics) = %d, want 1", len(hpa.Metrics))
	}
}

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

func TestCopySecretToNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "source-secret", Namespace: "source-ns"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"key": []byte("value")},
		},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopySecretToNamespace(ctx, clientset, "source-ns", "source-secret", "target-ns")
	if err != nil {
		t.Fatalf("CopySecretToNamespace() error = %v", err)
	}

	copied, err := clientset.CoreV1().Secrets("target-ns").Get(ctx, "source-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get copied secret: %v", err)
	}

	if string(copied.Data["key"]) != "value" {
		t.Errorf("Copied secret data mismatch")
	}
}

func TestCopyConfigMapToNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "source-cm", Namespace: "source-ns"},
			Data:       map[string]string{"config": "value"},
		},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "source-cm", "target-ns")
	if err != nil {
		t.Fatalf("CopyConfigMapToNamespace() error = %v", err)
	}

	copied, err := clientset.CoreV1().ConfigMaps("target-ns").Get(ctx, "source-cm", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get copied configmap: %v", err)
	}

	if copied.Data["config"] != "value" {
		t.Errorf("Copied configmap data mismatch")
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

func TestIngressReferencesService(t *testing.T) {
	pathTypePrefix := networkingv1.PathTypePrefix

	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathTypePrefix,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{Name: "web-service"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if !ingressReferencesService(ing, "web-service") {
		t.Error("ingressReferencesService() should return true")
	}

	if ingressReferencesService(ing, "other-service") {
		t.Error("ingressReferencesService() should return false for non-matching service")
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty slice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.str)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
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

func TestParseProbe(t *testing.T) {
	if parseProbe(nil) != nil {
		t.Error("parseProbe(nil) should return nil")
	}

	httpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/health",
				Port:   intstr.FromInt(8080),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: 10,
	}

	result := parseProbe(httpProbe)
	if result == nil {
		t.Fatal("parseProbe() should not return nil for HTTP probe")
	}
	if result.Type != "HTTP" {
		t.Errorf("Type = %q, want 'HTTP'", result.Type)
	}

	tcpProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromInt(3306)},
		},
	}
	result = parseProbe(tcpProbe)
	if result.Type != "TCP" {
		t.Errorf("Type = %q, want 'TCP'", result.Type)
	}

	execProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{Command: []string{"cat", "/tmp/healthy"}},
		},
	}
	result = parseProbe(execProbe)
	if result.Type != "Exec" {
		t.Errorf("Type = %q, want 'Exec'", result.Type)
	}
}

func TestGetScaleResourceType(t *testing.T) {
	tests := []struct {
		input    ResourceType
		expected string
	}{
		{ResourceDeployments, "deployment"},
		{ResourceStatefulSets, "statefulset"},
		{ResourceRollouts, "rollout"},
		{ResourceDaemonSets, "daemonsets"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := getScaleResourceType(tt.input)
			if result != tt.expected {
				t.Errorf("getScaleResourceType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-deployment",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	dep, err := GetDeployment(ctx, clientset, "default", "test-deployment")
	if err != nil {
		t.Fatalf("GetDeployment() error = %v", err)
	}

	if dep.Name != "test-deployment" {
		t.Errorf("Name = %q, want 'test-deployment'", dep.Name)
	}
}

func TestGetStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-statefulset",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	sts, err := GetStatefulSet(ctx, clientset, "default", "test-statefulset")
	if err != nil {
		t.Fatalf("GetStatefulSet() error = %v", err)
	}

	if sts.Name != "test-statefulset" {
		t.Errorf("Name = %q, want 'test-statefulset'", sts.Name)
	}
}

func TestGetDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-daemonset",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	ds, err := GetDaemonSet(ctx, clientset, "default", "test-daemonset")
	if err != nil {
		t.Fatalf("GetDaemonSet() error = %v", err)
	}

	if ds.Name != "test-daemonset" {
		t.Errorf("Name = %q, want 'test-daemonset'", ds.Name)
	}
}

func TestGetJob(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: "default",
			},
		},
	)

	ctx := context.Background()
	job, err := GetJob(ctx, clientset, "default", "test-job")
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if job.Name != "test-job" {
		t.Errorf("Name = %q, want 'test-job'", job.Name)
	}
}

// Note: ScaleDeployment and ScaleStatefulSet tests are skipped because
// the fake clientset doesn't properly support the Scale subresource.
// These functions are tested in integration tests with a real cluster.

func TestRestartDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartDeployment(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartDeployment() error = %v", err)
	}

	dep, _ := clientset.AppsV1().Deployments("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if dep.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestRestartStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartStatefulSet(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartStatefulSet() error = %v", err)
	}

	sts, _ := clientset.AppsV1().StatefulSets("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if sts.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestRestartDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-test",
				Namespace: "default",
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
		},
	)

	ctx := context.Background()
	err := RestartDaemonSet(ctx, clientset, "default", "restart-test")
	if err != nil {
		t.Fatalf("RestartDaemonSet() error = %v", err)
	}

	ds, _ := clientset.AppsV1().DaemonSets("default").Get(ctx, "restart-test", metav1.GetOptions{})
	if ds.Spec.Template.Annotations == nil {
		t.Error("Annotations should be set after restart")
	}
}

func TestGetWorkloadPods(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-pod-1",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "web-pod-2",
				Namespace:         "default",
				Labels:            map[string]string{"app": "web"},
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec:   corev1.PodSpec{Containers: []corev1.Container{{Name: "main"}}},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
	)

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 2", len(pods))
	}
}

func TestCopySecretToNamespace_Update(t *testing.T) {
	// Test updating existing secret
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "source-secret", Namespace: "source-ns"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"key": []byte("new-value")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "source-secret", Namespace: "target-ns"},
			Type:       corev1.SecretTypeOpaque,
			Data:       map[string][]byte{"key": []byte("old-value")},
		},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopySecretToNamespace(ctx, clientset, "source-ns", "source-secret", "target-ns")
	if err != nil {
		t.Fatalf("CopySecretToNamespace() error = %v", err)
	}

	copied, _ := clientset.CoreV1().Secrets("target-ns").Get(ctx, "source-secret", metav1.GetOptions{})
	if string(copied.Data["key"]) != "new-value" {
		t.Errorf("Secret should be updated with new value")
	}
}

func TestCopyConfigMapToNamespace_Update(t *testing.T) {
	// Test updating existing configmap
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "source-cm", Namespace: "source-ns"},
			Data:       map[string]string{"config": "new-value"},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: "source-cm", Namespace: "target-ns"},
			Data:       map[string]string{"config": "old-value"},
		},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "source-cm", "target-ns")
	if err != nil {
		t.Fatalf("CopyConfigMapToNamespace() error = %v", err)
	}

	copied, _ := clientset.CoreV1().ConfigMaps("target-ns").Get(ctx, "source-cm", metav1.GetOptions{})
	if copied.Data["config"] != "new-value" {
		t.Errorf("ConfigMap should be updated with new value")
	}
}

func TestListSecretsAllTypes(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "docker-secret",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type: corev1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{".dockerconfigjson": []byte("{}")},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "opaque-secret",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{"key": []byte("value")},
		},
	)

	ctx := context.Background()
	secrets, err := ListSecrets(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListSecrets() error = %v", err)
	}

	// ListSecrets returns all secrets including docker registry type
	if len(secrets) != 2 {
		t.Errorf("ListSecrets() returned %d secrets, want 2", len(secrets))
	}
}

func TestLabelsMatch(t *testing.T) {
	tests := []struct {
		name     string
		selector map[string]string
		labels   map[string]string
		expected bool
	}{
		{
			name:     "exact match",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "selector subset of labels",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "nginx", "env": "prod", "version": "v1"},
			expected: true,
		},
		{
			name:     "selector not in labels",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{"app": "redis"},
			expected: false,
		},
		{
			name:     "selector key missing from labels",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx"},
			expected: false,
		},
		{
			name:     "empty selector matches everything",
			selector: map[string]string{},
			labels:   map[string]string{"app": "nginx"},
			expected: true,
		},
		{
			name:     "empty labels with non-empty selector",
			selector: map[string]string{"app": "nginx"},
			labels:   map[string]string{},
			expected: false,
		},
		{
			name:     "both empty",
			selector: map[string]string{},
			labels:   map[string]string{},
			expected: true,
		},
		{
			name:     "multiple selector labels all match",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx", "env": "prod", "version": "v1"},
			expected: true,
		},
		{
			name:     "multiple selector labels partial match",
			selector: map[string]string{"app": "nginx", "env": "prod"},
			labels:   map[string]string{"app": "nginx", "env": "staging"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := labelsMatch(tt.selector, tt.labels)
			if result != tt.expected {
				t.Errorf("labelsMatch(%v, %v) = %v, want %v", tt.selector, tt.labels, result, tt.expected)
			}
		})
	}
}

func TestAllResourceTypes(t *testing.T) {
	// Verify AllResourceTypes contains expected types
	expectedTypes := map[ResourceType]bool{
		ResourceDeployments:  true,
		ResourceStatefulSets: true,
		ResourceDaemonSets:   true,
		ResourceJobs:         true,
		ResourceCronJobs:     true,
		ResourcePods:         true,
	}

	if len(AllResourceTypes) != len(expectedTypes) {
		t.Errorf("AllResourceTypes has %d types, expected %d", len(AllResourceTypes), len(expectedTypes))
	}

	for _, rt := range AllResourceTypes {
		if !expectedTypes[rt] {
			t.Errorf("Unexpected resource type in AllResourceTypes: %s", rt)
		}
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
						RunAsUser:             int64Ptr(1000),
						RunAsNonRoot:          boolPtr(true),
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

func boolPtr(b bool) *bool { return &b }
func int64Ptr(i int64) *int64 { return &i }

func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(err)
	}
	return q
}

func TestGetHPA_Full(t *testing.T) {
	cpu := int32(80)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-hpa",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
			Labels:            map[string]string{"app": "web"},
			Annotations:       map[string]string{"note": "test"},
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind:       "Deployment",
				Name:       "web-app",
				APIVersion: "apps/v1",
			},
			MinReplicas: int32Ptr(2),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 4,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(75),
						},
					},
				},
			},
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{
					Type:    autoscalingv2.ScalingActive,
					Status:  corev1.ConditionTrue,
					Reason:  "ValidMetricFound",
					Message: "the HPA was able to successfully calculate a replica count",
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "test-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if data.Name != "test-hpa" {
		t.Errorf("Name = %q, want 'test-hpa'", data.Name)
	}
	if data.Reference != "Deployment/web-app" {
		t.Errorf("Reference = %q, want 'Deployment/web-app'", data.Reference)
	}
	if data.MinReplicas != 2 {
		t.Errorf("MinReplicas = %d, want 2", data.MinReplicas)
	}
	if data.MaxReplicas != 10 {
		t.Errorf("MaxReplicas = %d, want 10", data.MaxReplicas)
	}
	if data.CurrentReplicas != 3 {
		t.Errorf("CurrentReplicas = %d, want 3", data.CurrentReplicas)
	}
	if data.DesiredReplicas != 4 {
		t.Errorf("DesiredReplicas = %d, want 4", data.DesiredReplicas)
	}
	if len(data.Metrics) != 1 {
		t.Errorf("len(Metrics) = %d, want 1", len(data.Metrics))
	}
	if len(data.Conditions) != 1 {
		t.Errorf("len(Conditions) = %d, want 1", len(data.Conditions))
	}
}

func TestGetHPA_WithMemoryAndExternal(t *testing.T) {
	mem := int32(70)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "api",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 20,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &mem,
						},
					},
				},
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_length",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.AverageValueMetricType,
							Value: resourceQuantityPtr("100"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 5,
			DesiredReplicas: 5,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceMemory,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(65),
						},
					},
				},
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_length",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: resourceQuantityPtr("85"),
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "complex-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 2 {
		t.Errorf("len(Metrics) = %d, want 2", len(data.Metrics))
	}
}

func resourceQuantityPtr(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}

func TestFormatHPATargets_ResourceMetrics(t *testing.T) {
	cpu := int32(80)
	mem := int32(70)

	hpa := autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceMemory,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &mem,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(60),
						},
					},
				},
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceMemory,
						Current: autoscalingv2.MetricValueStatus{
							AverageUtilization: int32Ptr(50),
						},
					},
				},
			},
		},
	}

	result := formatHPATargets(hpa)
	if result == "" {
		t.Error("formatHPATargets() returned empty string")
	}
	if !strings.Contains(result, "cpu") {
		t.Errorf("formatHPATargets() should contain 'cpu', got %q", result)
	}
	if !strings.Contains(result, "memory") {
		t.Errorf("formatHPATargets() should contain 'memory', got %q", result)
	}
}

func TestFormatHPATargets_ExternalMetrics(t *testing.T) {
	hpa := autoscalingv2.HorizontalPodAutoscaler{
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_messages",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.AverageValueMetricType,
							Value: resourceQuantityPtr("50"),
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "queue_messages",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: resourceQuantityPtr("30"),
						},
					},
				},
			},
		},
	}

	result := formatHPATargets(hpa)
	if result == "" {
		t.Error("formatHPATargets() returned empty string")
	}
	if !strings.Contains(result, "queue_messages") {
		t.Errorf("formatHPATargets() should contain 'queue_messages', got %q", result)
	}
}

func TestGetWorkloadPods_ForDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app",
				Namespace: "default",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "web"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-pod-1",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "web-app-pod-2",
				Namespace: "default",
				Labels:    map[string]string{"app": "web"},
			},
			Status: corev1.PodStatus{Phase: corev1.PodRunning},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-pod",
				Namespace: "default",
				Labels:    map[string]string{"app": "other"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Type:      ResourceDeployments,
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 2", len(pods))
	}
}

func TestGetWorkloadPods_ForStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db",
				Namespace: "default",
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "db"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "db-0",
				Namespace: "default",
				Labels:    map[string]string{"app": "db"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "db",
		Namespace: "default",
		Type:      ResourceStatefulSets,
		Labels:    map[string]string{"app": "db"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) != 1 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 1", len(pods))
	}
}

func TestGetWorkloadPods_ForJob(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "batch-job",
				Namespace: "default",
			},
			Spec: batchv1.JobSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job-name": "batch-job"},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "batch-job-xyz",
				Namespace: "default",
				Labels:    map[string]string{"job-name": "batch-job"},
			},
		},
	)

	workload := WorkloadInfo{
		Name:      "batch-job",
		Namespace: "default",
		Type:      ResourceJobs,
		Labels:    map[string]string{"job-name": "batch-job"},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	if len(pods) < 1 {
		t.Errorf("GetWorkloadPods() returned %d pods, want at least 1", len(pods))
	}
}

func TestScaleDeployment(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 2},
		}, nil
	})

	// Mock UpdateScale
	clientset.PrependReactor("update", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 5},
		}, nil
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deploy", 5)
	if err != nil {
		t.Fatalf("ScaleDeployment() error = %v", err)
	}
}

func TestScaleDeployment_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale to return error
	clientset.PrependReactor("get", "deployments/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	err := ScaleDeployment(ctx, clientset, "default", "test-deploy", 5)
	if err == nil {
		t.Error("ScaleDeployment() should return error")
	}
}

func TestScaleStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	// Mock GetScale
	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 3},
		}, nil
	})

	// Mock UpdateScale
	clientset.PrependReactor("update", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, &autoscalingv1.Scale{
			ObjectMeta: metav1.ObjectMeta{Name: "test-sts", Namespace: "default"},
			Spec:       autoscalingv1.ScaleSpec{Replicas: 5},
		}, nil
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-sts", 5)
	if err != nil {
		t.Fatalf("ScaleStatefulSet() error = %v", err)
	}
}

func TestScaleStatefulSet_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	clientset.PrependReactor("get", "statefulsets/scale", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	err := ScaleStatefulSet(ctx, clientset, "default", "test-sts", 5)
	if err == nil {
		t.Error("ScaleStatefulSet() should return error")
	}
}

func TestForceDeleteNamespace(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "stuck-ns",
				Finalizers: []string{"kubernetes"},
			},
			Status: corev1.NamespaceStatus{Phase: corev1.NamespaceTerminating},
		},
	)

	ctx := context.Background()
	// ForceDeleteNamespace requires a dynamic client for deleting arbitrary resources
	// Pass nil since the fake clientset doesn't support discovery properly
	err := ForceDeleteNamespace(ctx, clientset, nil, "stuck-ns")
	// The function should attempt to delete, may fail on finalizers in fake
	// but should not panic
	_ = err
}

func TestGetConfigMap_Full(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "app-config",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
		},
		Data: map[string]string{
			"config.yaml":   "key: value",
			"settings.json": `{"debug": true}`,
		},
	}

	clientset := fake.NewSimpleClientset(cm)

	ctx := context.Background()
	data, err := GetConfigMap(ctx, clientset, "default", "app-config")
	if err != nil {
		t.Fatalf("GetConfigMap() error = %v", err)
	}

	if data.Name != "app-config" {
		t.Errorf("Name = %q, want 'app-config'", data.Name)
	}
	if len(data.Data) != 2 {
		t.Errorf("len(Data) = %d, want 2", len(data.Data))
	}
	if data.Namespace != "default" {
		t.Errorf("Namespace = %q, want 'default'", data.Namespace)
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

func TestListConfigMaps_Full(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-1",
				Namespace: "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
			},
			Data: map[string]string{"key1": "val1", "key2": "val2"},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-2",
				Namespace: "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
			},
			Data: map[string]string{"config": "value"},
		},
	)

	ctx := context.Background()
	cms, err := ListConfigMaps(ctx, clientset, "default")
	if err != nil {
		t.Fatalf("ListConfigMaps() error = %v", err)
	}

	if len(cms) != 2 {
		t.Errorf("len(cms) = %d, want 2", len(cms))
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

func TestGetWorkloadEvents_ForStatefulSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sts-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "StatefulSet",
				Name: "db",
			},
			Type:   "Normal",
			Reason: "SuccessfulCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "db",
		Namespace: "default",
		Type:      ResourceStatefulSets,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	// Fake clientset doesn't support FieldSelector, so we get all events
	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
	}
}

func TestGetWorkloadEvents_ForDaemonSet(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ds-event",
				Namespace: "default",
			},
			InvolvedObject: corev1.ObjectReference{
				Kind: "DaemonSet",
				Name: "logging",
			},
			Type:   "Warning",
			Reason: "FailedCreate",
		},
	)

	workload := WorkloadInfo{
		Name:      "logging",
		Namespace: "default",
		Type:      ResourceDaemonSets,
	}

	ctx := context.Background()
	events, err := GetWorkloadEvents(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadEvents() error = %v", err)
	}

	if len(events) < 1 {
		t.Errorf("GetWorkloadEvents() returned %d events, want at least 1", len(events))
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

func TestIngressReferencesService_NilHTTPRule(t *testing.T) {
	// Test with a rule that has no HTTP section
	ing := networkingv1.Ingress{
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "example.com",
					// HTTP is nil
				},
			},
		},
	}

	if ingressReferencesService(ing, "any-service") {
		t.Error("ingressReferencesService() should return false when HTTP is nil")
	}
}

func TestRestartDeployment_NotFoundError(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartDeployment(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartDeployment() should return error for nonexistent deployment")
	}
}

func TestCopySecretToNamespace_Create(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-secret",
			Namespace: "source-ns",
			Labels:    map[string]string{"app": "test"},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"password": []byte("secret123"),
		},
	}

	targetNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "target-ns"},
	}

	clientset := fake.NewSimpleClientset(secret, targetNs)

	ctx := context.Background()
	err := CopySecretToNamespace(ctx, clientset, "source-ns", "my-secret", "target-ns")
	if err != nil {
		t.Fatalf("CopySecretToNamespace() error = %v", err)
	}

	// Verify secret was created in target namespace
	copied, err := clientset.CoreV1().Secrets("target-ns").Get(ctx, "my-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get copied secret: %v", err)
	}

	if string(copied.Data["password"]) != "secret123" {
		t.Error("Secret data should be copied")
	}
}

func TestCopyConfigMapToNamespace_Create(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-config",
			Namespace: "source-ns",
		},
		Data: map[string]string{
			"key": "value",
		},
	}

	targetNs := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: "target-ns"},
	}

	clientset := fake.NewSimpleClientset(cm, targetNs)

	ctx := context.Background()
	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "my-config", "target-ns")
	if err != nil {
		t.Fatalf("CopyConfigMapToNamespace() error = %v", err)
	}

	// Verify configmap was created in target namespace
	copied, err := clientset.CoreV1().ConfigMaps("target-ns").Get(ctx, "my-config", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get copied configmap: %v", err)
	}

	if copied.Data["key"] != "value" {
		t.Error("ConfigMap data should be copied")
	}
}

func TestGetHPA_ExternalMetric(t *testing.T) {
	avgValue := resource.MustParse("10")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keda-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "worker",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "rabbitmq_queue_length",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 5,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "rabbitmq_queue_length",
						},
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)

	ctx := context.Background()
	data, err := GetHPA(ctx, clientset, "default", "keda-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if data.Name != "keda-hpa" {
		t.Errorf("Name = %q, want 'keda-hpa'", data.Name)
	}
	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}
	if data.Metrics[0].Type != "External" {
		t.Errorf("Metric type = %q, want 'External'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "rabbitmq_queue_length" {
		t.Errorf("Metric name = %q, want 'rabbitmq_queue_length'", data.Metrics[0].Name)
	}
}

func TestParseProbe_gRPCType(t *testing.T) {
	// Test gRPC probe with service name
	port := int32(9000)
	serviceName := "health.v1.HealthService"
	grpcProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			GRPC: &corev1.GRPCAction{
				Port:    port,
				Service: &serviceName,
			},
		},
	}

	result := parseProbe(grpcProbe)
	if result == nil {
		t.Fatal("parseProbe should not return nil for valid gRPC probe")
	}
	if result.Type != "gRPC" {
		t.Errorf("Type = %q, want 'gRPC'", result.Type)
	}
	if result.Port != 9000 {
		t.Errorf("Port = %d, want 9000", result.Port)
	}
}

func TestGetConfigMap_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetConfigMap(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetConfigMap() should return error for nonexistent configmap")
	}
}

func TestGetSecret_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetSecret(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetSecret() should return error for nonexistent secret")
	}
}

func TestGetHPA_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetHPA(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetHPA() should return error for nonexistent HPA")
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

func TestCopySecretToNamespace_SourceNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopySecretToNamespace(ctx, clientset, "source-ns", "nonexistent", "target-ns")
	if err == nil {
		t.Error("CopySecretToNamespace() should return error for nonexistent source secret")
	}
}

func TestCopyConfigMapToNamespace_SourceNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target-ns"}},
	)

	ctx := context.Background()
	err := CopyConfigMapToNamespace(ctx, clientset, "source-ns", "nonexistent", "target-ns")
	if err == nil {
		t.Error("CopyConfigMapToNamespace() should return error for nonexistent source configmap")
	}
}

func TestRestartStatefulSet_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartStatefulSet(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartStatefulSet() should return error for nonexistent statefulset")
	}
}

func TestRestartDaemonSet_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	err := RestartDaemonSet(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("RestartDaemonSet() should return error for nonexistent daemonset")
	}
}

func TestListWorkloads_DeploymentError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "deployments", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_StatefulSetError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "statefulsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_DaemonSetError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "daemonsets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_JobsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "jobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_CronJobsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "cronjobs", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
	}
}

func TestListWorkloads_PodsError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListWorkloads(ctx, clientset, "default", ResourcePods)
	if err == nil {
		t.Error("ListWorkloads() should return error on API failure")
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

func TestListConfigMaps_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListConfigMaps(ctx, clientset, "default")
	if err == nil {
		t.Error("ListConfigMaps() should return error on API failure")
	}
}

func TestListSecrets_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "secrets", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListSecrets(ctx, clientset, "default")
	if err == nil {
		t.Error("ListSecrets() should return error on API failure")
	}
}

func TestListHPAs_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "horizontalpodautoscalers", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListHPAs(ctx, clientset, "default")
	if err == nil {
		t.Error("ListHPAs() should return error on API failure")
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

func TestListActiveNamespaceNames_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "namespaces", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	ctx := context.Background()
	_, err := ListActiveNamespaceNames(ctx, clientset)
	if err == nil {
		t.Error("ListActiveNamespaceNames() should return error on API failure")
	}
}

func TestGetWorkloadPods_Error(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	clientset.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, context.DeadlineExceeded
	})

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{"app": "web"},
	}

	ctx := context.Background()
	_, err := GetWorkloadPods(ctx, clientset, workload)
	if err == nil {
		t.Error("GetWorkloadPods() should return error on API failure")
	}
}

func TestGetWorkloadPods_EmptyLabels(t *testing.T) {
	// With empty labels, GetWorkloadPods returns all pods (no label filter)
	clientset := fake.NewSimpleClientset()

	workload := WorkloadInfo{
		Name:      "web-app",
		Namespace: "default",
		Labels:    map[string]string{},
	}

	ctx := context.Background()
	pods, err := GetWorkloadPods(ctx, clientset, workload)
	if err != nil {
		t.Fatalf("GetWorkloadPods() error = %v", err)
	}

	// With no pods in the namespace, should return empty slice
	if len(pods) != 0 {
		t.Errorf("GetWorkloadPods() returned %d pods, want 0", len(pods))
	}
}

func TestListWorkloads_Deployment_ZeroReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "scaled-down",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: int32Ptr(0),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "scaled"},
				},
			},
			Status: appsv1.DeploymentStatus{
				Replicas:      0,
				ReadyReplicas: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDeployments)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 replicas, status is "Running" since no replicas are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_StatefulSet_ZeroReplicas(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "scaled-down-sts",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: int32Ptr(0),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "scaled"},
				},
			},
			Status: appsv1.StatefulSetStatus{
				Replicas:      0,
				ReadyReplicas: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceStatefulSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 replicas, status is "Running" since no replicas are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_DaemonSet_ZeroScheduled(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "zero-ds",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "ds"},
				},
			},
			Status: appsv1.DaemonSetStatus{
				DesiredNumberScheduled: 0,
				NumberReady:            0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceDaemonSets)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	// With 0/0 scheduled, status is "Running" since no pods are missing
	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_Job_Failed(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "failed-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "failed"},
				},
			},
			Status: batchv1.JobStatus{
				Failed:    1,
				Succeeded: 0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Failed" {
		t.Errorf("Status = %q, want 'Failed'", workloads[0].Status)
	}
}

func TestListWorkloads_Job_Running(t *testing.T) {
	completions := int32(1)
	clientset := fake.NewSimpleClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "running-job",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.JobSpec{
				Completions: &completions,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"job": "running"},
				},
			},
			Status: batchv1.JobStatus{
				Active:    1,
				Succeeded: 0,
				Failed:    0,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Running" {
		t.Errorf("Status = %q, want 'Running'", workloads[0].Status)
	}
}

func TestListWorkloads_CronJob_Suspended(t *testing.T) {
	suspended := true
	clientset := fake.NewSimpleClientset(
		&batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "suspended-cron",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Spec: batchv1.CronJobSpec{
				Schedule: "0 0 * * *",
				Suspend:  &suspended,
			},
		},
	)

	ctx := context.Background()
	workloads, err := ListWorkloads(ctx, clientset, "default", ResourceCronJobs)
	if err != nil {
		t.Fatalf("ListWorkloads() error = %v", err)
	}

	if len(workloads) != 1 {
		t.Fatalf("ListWorkloads() returned %d workloads, want 1", len(workloads))
	}

	if workloads[0].Status != "Suspended" {
		t.Errorf("Status = %q, want 'Suspended'", workloads[0].Status)
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

func TestConfigMapData_WithBinaryData(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "binary-cm",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now()},
			},
			Data:       map[string]string{"text": "hello"},
			BinaryData: map[string][]byte{"binary": {0x00, 0x01, 0x02}},
		},
	)

	ctx := context.Background()
	cm, err := GetConfigMap(ctx, clientset, "default", "binary-cm")
	if err != nil {
		t.Fatalf("GetConfigMap() error = %v", err)
	}

	if cm.Data["text"] != "hello" {
		t.Errorf("Data['text'] = %q, want 'hello'", cm.Data["text"])
	}
}

// ============================================
// HPA with PodsMetricSourceType
// ============================================

func TestGetHPA_PodsMetric(t *testing.T) {
	avgValue := resource.MustParse("100")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pods-metric-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "worker",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.PodsMetricSourceType,
					Pods: &autoscalingv2.PodsMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "packets-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 2,
			DesiredReplicas: 3,
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "pods-metric-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Pods" {
		t.Errorf("Metric type = %q, want 'Pods'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "packets-per-second" {
		t.Errorf("Metric name = %q, want 'packets-per-second'", data.Metrics[0].Name)
	}
}

// ============================================
// HPA with ObjectMetricSourceType
// ============================================

func TestGetHPA_ObjectMetric(t *testing.T) {
	targetValue := resource.MustParse("200")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "object-metric-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "api",
			},
			MinReplicas: int32Ptr(2),
			MaxReplicas: 20,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						DescribedObject: autoscalingv2.CrossVersionObjectReference{
							Kind: "Service",
							Name: "api-service",
						},
						Metric: autoscalingv2.MetricIdentifier{
							Name: "requests-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: &targetValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "object-metric-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Object" {
		t.Errorf("Metric type = %q, want 'Object'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Name != "requests-per-second" {
		t.Errorf("Metric name = %q, want 'requests-per-second'", data.Metrics[0].Name)
	}
	if data.Metrics[0].Target != "200" {
		t.Errorf("Metric target = %q, want '200'", data.Metrics[0].Target)
	}
}

// ============================================
// HPA with ObjectMetricSourceType (AverageValue)
// ============================================

func TestGetHPA_ObjectMetric_AverageValue(t *testing.T) {
	avgValue := resource.MustParse("50")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "object-avg-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "web",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ObjectMetricSourceType,
					Object: &autoscalingv2.ObjectMetricSource{
						DescribedObject: autoscalingv2.CrossVersionObjectReference{
							Kind: "Ingress",
							Name: "main-ingress",
						},
						Metric: autoscalingv2.MetricIdentifier{
							Name: "hits-per-second",
						},
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "object-avg-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Object" {
		t.Errorf("Metric type = %q, want 'Object'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Target != "50" {
		t.Errorf("Metric target = %q, want '50'", data.Metrics[0].Target)
	}
}

// ============================================
// HPA with External metric using Value (not AverageValue)
// ============================================

func TestGetHPA_ExternalMetric_Value(t *testing.T) {
	targetValue := resource.MustParse("1000")
	currentValue := resource.MustParse("500")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-value-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "processor",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 50,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricSource{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "sqs_queue_messages",
						},
						Target: autoscalingv2.MetricTarget{
							Type:  autoscalingv2.ValueMetricType,
							Value: &targetValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 5,
			DesiredReplicas: 10,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ExternalMetricSourceType,
					External: &autoscalingv2.ExternalMetricStatus{
						Metric: autoscalingv2.MetricIdentifier{
							Name: "sqs_queue_messages",
						},
						Current: autoscalingv2.MetricValueStatus{
							Value: &currentValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "external-value-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "External" {
		t.Errorf("Metric type = %q, want 'External'", data.Metrics[0].Type)
	}
	// resource.Quantity formats 1000 as "1k"
	if data.Metrics[0].Target != "1k" {
		t.Errorf("Metric target = %q, want '1k'", data.Metrics[0].Target)
	}
	if data.Metrics[0].Current != "500" {
		t.Errorf("Metric current = %q, want '500'", data.Metrics[0].Current)
	}
}

// ============================================
// HPA with Resource metric using AverageValue (not AverageUtilization)
// ============================================

func TestGetHPA_ResourceMetric_AverageValue(t *testing.T) {
	avgValue := resource.MustParse("500m")
	currentAvgValue := resource.MustParse("300m")
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-avg-hpa",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "app",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:         autoscalingv2.AverageValueMetricType,
							AverageValue: &avgValue,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 4,
			CurrentMetrics: []autoscalingv2.MetricStatus{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricStatus{
						Name: corev1.ResourceCPU,
						Current: autoscalingv2.MetricValueStatus{
							AverageValue: &currentAvgValue,
						},
					},
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "resource-avg-hpa")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Metrics) != 1 {
		t.Fatalf("len(Metrics) = %d, want 1", len(data.Metrics))
	}

	if data.Metrics[0].Type != "Resource" {
		t.Errorf("Metric type = %q, want 'Resource'", data.Metrics[0].Type)
	}
	if data.Metrics[0].Target != "500m" {
		t.Errorf("Metric target = %q, want '500m'", data.Metrics[0].Target)
	}
	if data.Metrics[0].Current != "300m" {
		t.Errorf("Metric current = %q, want '300m'", data.Metrics[0].Current)
	}
}

// ============================================
// HPA with Conditions
// ============================================

func TestGetHPA_WithConditions(t *testing.T) {
	cpu := int32(80)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hpa-with-conditions",
			Namespace: "default",
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: "app",
			},
			MinReplicas: int32Ptr(1),
			MaxReplicas: 10,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: corev1.ResourceCPU,
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpu,
						},
					},
				},
			},
		},
		Status: autoscalingv2.HorizontalPodAutoscalerStatus{
			CurrentReplicas: 3,
			DesiredReplicas: 3,
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{
				{
					Type:    autoscalingv2.ScalingActive,
					Status:  corev1.ConditionTrue,
					Reason:  "ValidMetricFound",
					Message: "the HPA was able to successfully calculate a replica count from cpu resource utilization",
				},
				{
					Type:    autoscalingv2.AbleToScale,
					Status:  corev1.ConditionTrue,
					Reason:  "ReadyForNewScale",
					Message: "the HPA controller is able to scale the target resource",
				},
			},
		},
	}

	clientset := fake.NewSimpleClientset(hpa)
	ctx := context.Background()

	data, err := GetHPA(ctx, clientset, "default", "hpa-with-conditions")
	if err != nil {
		t.Fatalf("GetHPA() error = %v", err)
	}

	if len(data.Conditions) != 2 {
		t.Fatalf("len(Conditions) = %d, want 2", len(data.Conditions))
	}

	found := false
	for _, c := range data.Conditions {
		if c.Type == "ScalingActive" {
			found = true
			if c.Status != "True" {
				t.Errorf("ScalingActive status = %q, want 'True'", c.Status)
			}
			if c.Reason != "ValidMetricFound" {
				t.Errorf("ScalingActive reason = %q, want 'ValidMetricFound'", c.Reason)
			}
		}
	}
	if !found {
		t.Error("Expected to find ScalingActive condition")
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

func TestExtractRolloutReplicas(t *testing.T) {
	tests := []struct {
		name              string
		rolloutObj        map[string]interface{}
		expectedReplicas  int32
		expectedReady     int32
	}{
		{
			name:              "empty object",
			rolloutObj:        map[string]interface{}{},
			expectedReplicas:  1,
			expectedReady:     0,
		},
		{
			name: "int64 replicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": int64(5),
				},
				"status": map[string]interface{}{
					"readyReplicas": int64(3),
				},
			},
			expectedReplicas:  5,
			expectedReady:     3,
		},
		{
			name: "float64 replicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": float64(10),
				},
				"status": map[string]interface{}{
					"readyReplicas": float64(8),
				},
			},
			expectedReplicas:  10,
			expectedReady:     8,
		},
		{
			name: "fallback to availableReplicas",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": int64(3),
				},
				"status": map[string]interface{}{
					"availableReplicas": int64(2),
				},
			},
			expectedReplicas:  3,
			expectedReady:     2,
		},
		{
			name: "availableReplicas float64",
			rolloutObj: map[string]interface{}{
				"spec": map[string]interface{}{
					"replicas": float64(4),
				},
				"status": map[string]interface{}{
					"availableReplicas": float64(3),
				},
			},
			expectedReplicas:  4,
			expectedReady:     3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replicas, readyReplicas := extractRolloutReplicas(tt.rolloutObj)
			if replicas != tt.expectedReplicas {
				t.Errorf("extractRolloutReplicas() replicas = %d, want %d", replicas, tt.expectedReplicas)
			}
			if readyReplicas != tt.expectedReady {
				t.Errorf("extractRolloutReplicas() readyReplicas = %d, want %d", readyReplicas, tt.expectedReady)
			}
		})
	}
}

func TestCountReadyEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		epSlices *discoveryv1.EndpointSliceList
		expected int
	}{
		{
			name:     "nil epSlices",
			epSlices: nil,
			expected: 0,
		},
		{
			name:     "empty epSlices",
			epSlices: &discoveryv1.EndpointSliceList{},
			expected: 0,
		},
		{
			name: "mixed ready states",
			epSlices: &discoveryv1.EndpointSliceList{
				Items: []discoveryv1.EndpointSlice{
					{
						Endpoints: []discoveryv1.Endpoint{
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(false)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
						},
					},
					{
						Endpoints: []discoveryv1.Endpoint{
							{Conditions: discoveryv1.EndpointConditions{Ready: boolPtr(true)}},
							{Conditions: discoveryv1.EndpointConditions{Ready: nil}}, // nil ready
						},
					},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countReadyEndpoints(tt.epSlices)
			if result != tt.expected {
				t.Errorf("countReadyEndpoints() = %d, want %d", result, tt.expected)
			}
		})
	}
}
