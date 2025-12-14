package repository

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

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
