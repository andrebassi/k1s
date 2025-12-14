package repository

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

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

func TestListConfigMaps_Full(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "cm-1",
				Namespace:         "default",
				CreationTimestamp: metav1.Time{Time: time.Now().Add(-2 * time.Hour)},
			},
			Data: map[string]string{"key1": "val1", "key2": "val2"},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "cm-2",
				Namespace:         "default",
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

func TestGetConfigMap_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetConfigMap(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetConfigMap() should return error for nonexistent configmap")
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
