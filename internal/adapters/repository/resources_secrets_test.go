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

func TestGetSecret_NotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset()

	ctx := context.Background()
	_, err := GetSecret(ctx, clientset, "default", "nonexistent")
	if err == nil {
		t.Error("GetSecret() should return error for nonexistent secret")
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
